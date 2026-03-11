package tui

import (
	"fmt"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
)

// PlanFile holds metadata for a plan file in the ~/.claude/plans directory.
type PlanFile struct {
	Name    string
	Path    string
	ModTime time.Time
}

// ListModel is a bubbletea model for interactively selecting a plan file to review.
type ListModel struct {
	files        []PlanFile
	cursor       int
	selected     string
	windowWidth  int
	windowHeight int
}

// NewListModel returns a ListModel for the given plan files.
func NewListModel(files []PlanFile) ListModel {
	return ListModel{files: files}
}

// Selected returns the path chosen by the user, or "" if the user quit without selecting.
func (m ListModel) Selected() string {
	return m.selected
}

// Init implements tea.Model.
func (m ListModel) Init() tea.Cmd {
	return nil
}

// Update implements tea.Model.
func (m ListModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.windowWidth = msg.Width
		m.windowHeight = msg.Height
	case tea.KeyMsg:
		switch msg.String() {
		case "q", "esc", "ctrl+c":
			return m, tea.Quit
		case "j", "down":
			if m.cursor < len(m.files)-1 {
				m.cursor++
			}
		case "k", "up":
			if m.cursor > 0 {
				m.cursor--
			}
		case "enter":
			if len(m.files) > 0 {
				m.selected = m.files[m.cursor].Path
				return m, tea.Quit
			}
		}
	}
	return m, nil
}

// View implements tea.Model.
func (m ListModel) View() string {
	if m.windowWidth == 0 {
		return ""
	}

	var sb strings.Builder

	header := fmt.Sprintf("  pluto — Plans  [%d files]", len(m.files))
	sb.WriteString(headerStyle.Width(m.windowWidth).Render(header) + "\n")

	if len(m.files) == 0 {
		sb.WriteString("\n")
		sb.WriteString(dimStyle.Render("  No plan files found.\n"))
	} else {
		sb.WriteString("\n")

		// Layout: "  ▶  <name>  <age>"
		const ageWidth = 14
		const prefixWidth = 5 // "  ▶  " or "     "
		const gapWidth = 2
		nameWidth := m.windowWidth - prefixWidth - gapWidth - ageWidth
		if nameWidth < 10 {
			nameWidth = 10
		}

		for i, f := range m.files {
			name := strings.TrimSuffix(f.Name, ".md")
			if len(name) > nameWidth {
				name = name[:nameWidth-1] + "…"
			}
			age := fmt.Sprintf("%-*s", ageWidth, relativeTime(f.ModTime))

			if i == m.cursor {
				line := fmt.Sprintf("  ▶  %-*s  %s", nameWidth, name, age)
				sb.WriteString(cursorLineStyle.Width(m.windowWidth).Render(line) + "\n")
			} else {
				line := fmt.Sprintf("     %-*s  ", nameWidth, name)
				sb.WriteString(line + dimStyle.Render(age) + "\n")
			}
		}

		sb.WriteString("\n")
	}

	sb.WriteString(statusBarStyle.Width(m.windowWidth).Render("  j/k:navigate  enter:open  q:quit"))
	return sb.String()
}

// relativeTime returns a human-readable relative time string for the given time.
func relativeTime(t time.Time) string {
	d := time.Since(t)
	switch {
	case d < time.Minute:
		return "just now"
	case d < time.Hour:
		n := int(d.Minutes())
		if n == 1 {
			return "1 minute ago"
		}
		return fmt.Sprintf("%d minutes ago", n)
	case d < 24*time.Hour:
		n := int(d.Hours())
		if n == 1 {
			return "1 hour ago"
		}
		return fmt.Sprintf("%d hours ago", n)
	case d < 7*24*time.Hour:
		n := int(d.Hours() / 24)
		if n == 1 {
			return "1 day ago"
		}
		return fmt.Sprintf("%d days ago", n)
	case d < 30*24*time.Hour:
		n := int(d.Hours() / 24 / 7)
		if n == 1 {
			return "1 week ago"
		}
		return fmt.Sprintf("%d weeks ago", n)
	default:
		n := int(d.Hours() / 24 / 30)
		if n < 1 {
			n = 1
		}
		if n == 1 {
			return "1 month ago"
		}
		return fmt.Sprintf("%d months ago", n)
	}
}
