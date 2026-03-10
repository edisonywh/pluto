package tui

import (
	"strings"
)

// renderDiffPanel renders the left panel in diff mode with colorized unified diff output.
func renderDiffPanel(diffLines []string, scrollOffset, width, height int) string {
	var sb strings.Builder

	// Panel title
	sb.WriteString(panelTitleStyle.Render("  Diff") + "\n")

	if len(diffLines) == 0 || (len(diffLines) == 1 && diffLines[0] == "") {
		sb.WriteString(dimStyle.Render("  No previous version to diff against.\n"))
		return sb.String()
	}

	visibleStart := scrollOffset
	visibleEnd := min(visibleStart+height-1, len(diffLines)) // -1 for title row

	for i := visibleStart; i < visibleEnd; i++ {
		line := diffLines[i]
		var rendered string
		switch {
		case strings.HasPrefix(line, "+++") || strings.HasPrefix(line, "---"):
			rendered = diffHeaderStyle.Render(line)
		case strings.HasPrefix(line, "+"):
			rendered = diffAddStyle.Render(line)
		case strings.HasPrefix(line, "-"):
			rendered = diffRemoveStyle.Render(line)
		case strings.HasPrefix(line, "@@"):
			rendered = diffHunkStyle.Render(line)
		default:
			rendered = line
		}
		sb.WriteString(rendered + "\n")
	}

	return sb.String()
}
