package spawn

import (
	"fmt"
	"os"
	"os/exec"
)

// OpenReviewWindow opens a new terminal window running pluto in review mode.
// plutoPath is the absolute path to the pluto binary.
// inFile  is the temp file containing the handoff payload.
// outFile is the temp file where the review result will be written.
func OpenReviewWindow(plutoPath, inFile, outFile string) error {
	switch detectTerminal() {
	case "ghostty":
		return openGhosttyWindow(plutoPath, inFile, outFile)
	case "iterm2":
		return openITermWindow(plutoPath, inFile, outFile)
	default:
		return openTerminalAppWindow(plutoPath, inFile, outFile)
	}
}

func detectTerminal() string {
	if _, err := os.Stat("/Applications/Ghostty.app"); err == nil {
		return "ghostty"
	}
	if _, err := os.Stat("/Applications/iTerm.app"); err == nil {
		return "iterm2"
	}
	return "terminal"
}

func openGhosttyWindow(plutoPath, inFile, outFile string) error {
	return exec.Command("open", "-na", "Ghostty.app", "--args",
		"-e", plutoPath, "review", "--in", inFile, "--out", outFile).Start()
}

func openITermWindow(plutoPath, inFile, outFile string) error {
	cmd := fmt.Sprintf("%s review --in %s --out %s", plutoPath, inFile, outFile)
	script := fmt.Sprintf(`tell application "iTerm2"
		create window with default profile command "%s"
	end tell`, escapeAppleScript(cmd))
	return exec.Command("osascript", "-e", script).Start()
}

func openTerminalAppWindow(plutoPath, inFile, outFile string) error {
	cmd := fmt.Sprintf(`%s review --in %s --out %s; exit`, plutoPath, inFile, outFile)
	script := fmt.Sprintf(
		`tell application "Terminal"
			activate
			do script "%s"
		end tell`,
		escapeAppleScript(cmd),
	)
	return exec.Command("osascript", "-e", script).Start()
}

// escapeAppleScript escapes a string for safe embedding inside AppleScript double quotes.
func escapeAppleScript(s string) string {
	out := make([]byte, 0, len(s))
	for i := 0; i < len(s); i++ {
		c := s[i]
		switch c {
		case '"':
			out = append(out, '\\', '"')
		case '\\':
			out = append(out, '\\', '\\')
		default:
			out = append(out, c)
		}
	}
	return string(out)
}
