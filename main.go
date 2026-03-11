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
	"pluto/internal/annotation"
	"pluto/internal/diff"
	"pluto/internal/handoff"
	"pluto/internal/history"
	"pluto/internal/hook"
	"pluto/internal/spawn"
	"pluto/internal/tui"
)

func main() {
	// Internal: review mode spawned by hook mode in a separate terminal.
	if len(os.Args) == 4 && os.Args[1] == "--review" {
		runReviewMode(os.Args[2], os.Args[3])
		return
	}
	// pluto list — interactive plan picker.
	if len(os.Args) == 2 && os.Args[1] == "list" {
		runListMode()
		return
	}
	// pluto <file> — standalone review of a plan file.
	if len(os.Args) == 2 && !strings.HasPrefix(os.Args[1], "-") {
		runStandaloneMode(os.Args[1])
		return
	}
	// Default: hook mode (called by Claude Code).
	runHookMode()
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
	plansDir := filepath.Join(home, ".claude", "plans")

	entries, err := os.ReadDir(plansDir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "pluto: cannot read plans directory: %v\n", err)
		os.Exit(1)
	}

	var files []tui.PlanFile
	for _, e := range entries {
		if !e.IsDir() && strings.HasSuffix(e.Name(), ".md") {
			info, _ := e.Info()
			files = append(files, tui.PlanFile{
				Name:    e.Name(),
				Path:    filepath.Join(plansDir, e.Name()),
				ModTime: info.ModTime(),
			})
		}
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
	if selected != "" {
		runStandaloneMode(selected)
	}
}

func runHookMode() {
	// Bail out early if stdin is a TTY (i.e. run directly, not as a hook).
	if fi, err := os.Stdin.Stat(); err == nil && (fi.Mode()&os.ModeCharDevice) != 0 {
		fmt.Fprintln(os.Stderr, "pluto: must be invoked as a Claude Code PreToolUse hook, not run directly")
		fmt.Fprintln(os.Stderr, "  Add to ~/.claude/settings.json:")
		fmt.Fprintln(os.Stderr, `  {"hooks":{"PreToolUse":[{"matcher":"ExitPlanMode","hooks":[{"type":"command","command":"pluto","timeout":300}]}]}}`)
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
