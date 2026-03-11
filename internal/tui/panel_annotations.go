package tui

import (
	"fmt"
	"strings"

	"pluto/internal/annotation"
)

// renderAnnotationsPanel renders the right panel listing all current annotations.
func renderAnnotationsPanel(m Model, width, height int) string {
	_ = height // reserved for future scrolling
	count := len(m.annotations)

	var sb strings.Builder
	sb.WriteString(panelTitleStyle.Render(fmt.Sprintf("  Annotations (%d)", count)) + "\n")
	sb.WriteString("\n")

	if count == 0 {
		sb.WriteString(dimStyle.Render("  No annotations yet.\n"))
		sb.WriteString(dimStyle.Render("  v → select lines\n"))
		sb.WriteString(dimStyle.Render("  c/x/r → annotate\n"))
		return sb.String()
	}

	for i, a := range m.annotations {
		isSelected := m.mode == ModeAnnotations && i == m.annotationCursor
		rangeStr := a.Range.String()

		var item string
		switch a.Type {
		case annotation.Delete:
			item = fmt.Sprintf("  [%s] DELETE", rangeStr)
			if isSelected {
				sb.WriteString(cursorLineStyle.Width(width).Render(item) + "\n\n")
			} else {
				sb.WriteString(annotationDeleteStyle.Render(item) + "\n\n")
			}

		case annotation.Comment:
			item = fmt.Sprintf("  [%s] COMMENT", rangeStr)
			if isSelected {
				sb.WriteString(cursorLineStyle.Width(width).Render(item) + "\n")
			} else {
				sb.WriteString(annotationCommentStyle.Render(item) + "\n")
			}
			if a.Message != "" {
				msg := fmt.Sprintf("  %q", a.Message)
				sb.WriteString(dimStyle.Width(width).Render(msg) + "\n")
			}
			sb.WriteString("\n")

		case annotation.Replace:
			item = fmt.Sprintf("  [%s] REPLACE", rangeStr)
			if isSelected {
				sb.WriteString(cursorLineStyle.Width(width).Render(item) + "\n")
			} else {
				sb.WriteString(annotationReplaceStyle.Render(item) + "\n")
			}
			if a.Message != "" {
				msg := fmt.Sprintf("  %q", a.Message)
				sb.WriteString(dimStyle.Width(width).Render(msg) + "\n")
			}
			sb.WriteString("\n")
		}
	}

	return sb.String()
}
