package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/spf13/cobra"
	"pluto/internal/annotation"
	"pluto/internal/diff"
	"pluto/internal/handoff"
	"pluto/internal/history"
	"pluto/internal/hook"
	"pluto/internal/spawn"
	"pluto/internal/tui"
)

func main() {
	rootCmd := &cobra.Command{
		Use:   "pluto",
		Short: "Vim-style plan reviewer for Claude Code",
		Long: `Pluto hooks into Claude Code's PreToolUse permission system to intercept
ExitPlanMode calls, letting you read, annotate, approve or reject plans
before any code is written.`,
	}

	var handoffIn, handoffOut string

	reviewCmd := &cobra.Command{
		Use:   "review [file]",
		Short: "Review a plan — pipe stdin for hook mode, pass a file for standalone",
		Long: `Review a Claude Code plan.

Hook mode (used in Claude Code settings):
  pluto review          — reads ExitPlanMode JSON from stdin

Standalone mode:
  pluto review plan.md  — opens a saved plan file for review`,
		Args: cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			// Internal: spawned by hook mode to show TUI in a new terminal window.
			if handoffIn != "" && handoffOut != "" {
				runReviewMode(handoffIn, handoffOut)
				return nil
			}
			// Standalone: a plan file was provided.
			if len(args) == 1 {
				runStandaloneMode(args[0])
				return nil
			}
			// Hook mode: stdin must be a pipe (called by Claude Code).
			runHookMode()
			return nil
		},
		SilenceUsage: true,
	}

	reviewCmd.Flags().StringVar(&handoffIn, "in", "", "")
	reviewCmd.Flags().StringVar(&handoffOut, "out", "", "")
	_ = reviewCmd.Flags().MarkHidden("in")
	_ = reviewCmd.Flags().MarkHidden("out")

	listCmd := &cobra.Command{
		Use:   "list",
		Short: "Browse and open saved plan files interactively",
		Long:  `Interactively browse all plan files saved in ~/.claude/plans/ and open one for review.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			runListMode()
			return nil
		},
		SilenceUsage: true,
	}

	rootCmd.AddCommand(reviewCmd, listCmd)

	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}

func runStandaloneMode(planPath string) {
	data, err := os.ReadFile(planPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "pluto: %v\n", err)
		os.Exit(1)
	}

	tty, err := os.OpenFile("/dev/tty", os.O_RDWR, 0)
	if err != nil {
		fmt.Fprintln(os.Stderr, "pluto: cannot open /dev/tty")
		os.Exit(1)
	}
	defer tty.Close()

	m := tui.NewModel(string(data), "", filepath.Base(planPath))
	p := tea.NewProgram(m,
		tea.WithInput(tty),
		tea.WithOutput(tty),
		tea.WithAltScreen(),
	)

	finalModel, err := p.Run()
	if err != nil {
		os.Exit(1)
	}

	result := finalModel.(tui.Model).Result()
	if result.Decision == tui.Reject {
		if len(result.Annotations) > 0 {
			fmt.Print(annotation.Format(result.Annotations))
		}
		os.Exit(1)
	}
}

func runListMode() {
	home, err := os.UserHomeDir()
	if err != nil {
		fmt.Fprintln(os.Stderr, "pluto: cannot determine home directory")
		os.Exit(1)
	}
	historyDir := filepath.Join(home, ".pluto", "history")

	sessions, err := os.ReadDir(historyDir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "pluto: cannot read history directory: %v\n", err)
		os.Exit(1)
	}

	var files []tui.PlanFile
	for _, session := range sessions {
		if !session.IsDir() {
			continue
		}
		sessionPath := filepath.Join(historyDir, session.Name())
		revisions, err := os.ReadDir(sessionPath)
		if err != nil {
			continue
		}
		// Find the last two non-dir entries (latest and previous revision).
		var prevEntry, latestEntry os.DirEntry
		for _, rev := range revisions {
			if !rev.IsDir() {
				prevEntry = latestEntry
				latestEntry = rev
			}
		}
		if latestEntry == nil {
			continue
		}
		info, err := latestEntry.Info()
		if err != nil {
			continue
		}
		prevPath := ""
		if prevEntry != nil {
			prevPath = filepath.Join(sessionPath, prevEntry.Name())
		}
		latestPath := filepath.Join(sessionPath, latestEntry.Name())
		title := planTitle(latestPath)
		files = append(files, tui.PlanFile{
			Name:     session.Name(),
			Title:    title,
			Path:     latestPath,
			PrevPath: prevPath,
			ModTime:  info.ModTime(),
		})
	}

	sort.Slice(files, func(i, j int) bool {
		return files[i].ModTime.After(files[j].ModTime)
	})

	tty, err := os.OpenFile("/dev/tty", os.O_RDWR, 0)
	if err != nil {
		fmt.Fprintln(os.Stderr, "pluto: cannot open /dev/tty")
		os.Exit(1)
	}

	lm := tui.NewListModel(files)
	p := tea.NewProgram(lm,
		tea.WithInput(tty),
		tea.WithOutput(tty),
		tea.WithAltScreen(),
	)

	finalModel, err := p.Run()
	tty.Close()
	if err != nil {
		os.Exit(1)
	}

	selected := finalModel.(tui.ListModel).Selected()
	if selected.Path == "" {
		return
	}

	planData, err := os.ReadFile(selected.Path)
	if err != nil {
		fmt.Fprintf(os.Stderr, "pluto: %v\n", err)
		os.Exit(1)
	}
	diffText := ""
	if selected.PrevPath != "" {
		prevData, err := os.ReadFile(selected.PrevPath)
		if err == nil {
			diffText = diff.Compute(string(prevData), string(planData))
		}
	}

	tty2, err := os.OpenFile("/dev/tty", os.O_RDWR, 0)
	if err != nil {
		fmt.Fprintln(os.Stderr, "pluto: cannot open /dev/tty")
		os.Exit(1)
	}
	defer tty2.Close()

	m := tui.NewModel(string(planData), diffText, selected.Name)
	p2 := tea.NewProgram(m,
		tea.WithInput(tty2),
		tea.WithOutput(tty2),
		tea.WithAltScreen(),
	)
	p2.Run()
}

// planTitle reads the first non-empty line of a plan file for display in the list.
func planTitle(path string) string {
	data, err := os.ReadFile(path)
	if err != nil {
		return ""
	}
	for _, line := range strings.SplitN(string(data), "\n", 20) {
		line = strings.TrimSpace(line)
		if line != "" {
			return strings.TrimLeft(line, "# ")
		}
	}
	return ""
}

func runHookMode() {
	// Bail out early if stdin is a TTY (i.e. run directly, not as a hook).
	if fi, err := os.Stdin.Stat(); err == nil && (fi.Mode()&os.ModeCharDevice) != 0 {
		fmt.Fprintln(os.Stderr, "pluto: must be invoked as a Claude Code PreToolUse hook, not run directly")
		fmt.Fprintln(os.Stderr, "  Add to ~/.claude/settings.json:")
		fmt.Fprintln(os.Stderr, `  {"hooks":{"PreToolUse":[{"matcher":"ExitPlanMode","hooks":[{"type":"command","command":"pluto review","timeout":300}]}]}}`)
		os.Exit(1)
	}

	// Read the PermissionRequest JSON from stdin (pipe from Claude Code).
	var input hook.PermissionRequest
	if err := json.NewDecoder(os.Stdin).Decode(&input); err != nil {
		fmt.Print(hook.AllowOutput())
		return
	}

	if input.ToolName != "ExitPlanMode" {
		fmt.Print(hook.AllowOutput())
		return
	}

	planText := input.ToolInput.Plan
	sessionID := input.SessionID
	if sessionID == "" {
		sessionID = "default"
	}

	// Persist before TUI so history survives force-quit.
	store := history.New()
	prevPlan, _ := store.LoadLatest(sessionID)
	if err := store.SavePlan(sessionID, planText); err != nil {
		log.Printf("pluto: failed to save plan history: %v", err)
	}

	diffText := ""
	if prevPlan != "" {
		diffText = diff.Compute(prevPlan, planText)
	}

	// Write handoff payload to a temp file.
	inFile, err := os.CreateTemp("", "pluto-in-*.json")
	if err != nil {
		fmt.Print(hook.AllowOutput())
		return
	}
	inFile.Close()
	inPath := inFile.Name()
	defer os.Remove(inPath)

	if err := handoff.WritePayload(inPath, handoff.Payload{
		Plan:      planText,
		Diff:      diffText,
		SessionID: sessionID,
	}); err != nil {
		fmt.Print(hook.AllowOutput())
		return
	}

	// Create the outFile path (doesn't exist yet; review mode will create it).
	outFile, err := os.CreateTemp("", "pluto-out-*.json")
	if err != nil {
		fmt.Print(hook.AllowOutput())
		return
	}
	outFile.Close()
	outPath := outFile.Name()
	os.Remove(outPath) // remove so we can poll for its creation
	defer os.Remove(outPath)

	// Get the absolute path of the current pluto binary.
	plutoPath, err := os.Executable()
	if err != nil {
		fmt.Print(hook.AllowOutput())
		return
	}

	// Open a new terminal window running pluto in review mode.
	if err := spawn.OpenReviewWindow(plutoPath, inPath, outPath); err != nil {
		log.Printf("pluto: failed to open review window: %v", err)
		fmt.Print(hook.AllowOutput())
		return
	}

	// Poll outFile every 200 ms; allow after 290 s timeout.
	deadline := time.Now().Add(290 * time.Second)
	for time.Now().Before(deadline) {
		time.Sleep(200 * time.Millisecond)
		if _, err := os.Stat(outPath); err == nil {
			break
		}
	}

	data, err := os.ReadFile(outPath)
	if err != nil {
		// Timeout or read failure — allow.
		fmt.Print(hook.AllowOutput())
		return
	}

	// outFile contains the final hook JSON; print it directly.
	fmt.Print(string(data))
}

func runReviewMode(inPath, outPath string) {
	payload, err := handoff.ReadPayload(inPath)
	if err != nil {
		os.Exit(1)
	}

	tty, err := os.OpenFile("/dev/tty", os.O_RDWR, 0)
	if err != nil {
		os.Exit(1)
	}
	defer tty.Close()

	m := tui.NewModel(payload.Plan, payload.Diff, payload.SessionID)
	p := tea.NewProgram(m,
		tea.WithInput(tty),
		tea.WithOutput(tty),
		tea.WithAltScreen(),
	)

	finalModel, err := p.Run()
	if err != nil {
		os.Exit(1)
	}

	result := finalModel.(tui.Model).Result()
	var output string
	switch result.Decision {
	case tui.Reject:
		msg := annotation.Format(result.Annotations)
		output = hook.DenyOutput(msg)
	default:
		output = hook.AllowOutput()
	}

	// Write result atomically: write to tmp, then rename.
	tmpPath := outPath + ".tmp"
	if err := os.WriteFile(tmpPath, []byte(output), 0600); err != nil {
		os.Exit(1)
	}
	if err := os.Rename(tmpPath, outPath); err != nil {
		os.Exit(1)
	}
}
