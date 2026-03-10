package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
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
	if len(os.Args) == 4 && os.Args[1] == "--review" {
		runReviewMode(os.Args[2], os.Args[3])
		return
	}
	runHookMode()
}

func runHookMode() {
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
