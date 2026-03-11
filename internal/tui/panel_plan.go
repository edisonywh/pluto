package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// renderPlanPanel renders the left panel content showing the plan with line numbers,
// cursor, and visual selection highlights.
func renderPlanPanel(m Model, width, height int) string {
	lines := m.planLines
	visibleStart := m.scrollOffset
	visibleEnd := min(visibleStart+height-2, len(lines)) // -1 title, -1 indicator

	// Compute visual selection bounds (0-indexed, inclusive)
	selStart, selEnd := -1, -1
	if m.mode == ModeVisual || (m.mode == ModeAnnotate && m.pendingCharText == "") {
		r := visualRange(m.visualStart, m.cursor)
		selStart = r.Start - 1
		selEnd = r.End - 1
	}

	// Char-visual selection state
	charSelActive := m.mode == ModeVisualChar
	charSelLo := min(m.charAnchorCol, m.charCursorCol)
	charSelHi := max(m.charAnchorCol, m.charCursorCol)

	var sb strings.Builder

	// Panel title
	title := panelTitleStyle.Render("  Plan")
	sb.WriteString(title + "\n")

	// Content width available after the 4-char line-number prefix.
	contentWidth := width - 4

	for i := visibleStart; i < visibleEnd; i++ {
		lineContent := lines[i]
		subLines := wrapLine(lineContent, contentWidth)

		for j, sub := range subLines {
			var prefix string
			if j == 0 {
				prefix = fmt.Sprintf("%3d ", i+1)
			} else {
				prefix = "    "
			}

			var rendered string
			switch {
			case i == m.cursor && charSelActive && j == 0:
				rendered = renderCharVisualLine(prefix, sub, charSelLo, charSelHi, width)

			case i == m.cursor:
				rendered = cursorLineStyle.Width(width).Render(prefix + sub)

			case selStart >= 0 && i >= selStart && i <= selEnd:
				rendered = visualLineStyle.Width(width).Render(prefix + sub)

			default:
				numPart := lineNumStyle.Render(prefix)
				contentPart := styledPlanLine(sub)
				rendered = numPart + contentPart
			}

			sb.WriteString(rendered + "\n")
		}
	}

	// Scroll percentage indicator
	if visibleEnd < len(lines) || visibleStart > 0 {
		pct := 0
		denom := len(lines) - height
		if denom > 0 {
			pct = m.scrollOffset * 100 / denom
		}
		indicator := dimStyle.Render(fmt.Sprintf("  [%d%%]", pct))
		sb.WriteString(indicator + "\n")
	}

	return sb.String()
}

// wrapLine splits line into sub-lines that each fit within width visible characters,
// breaking on word boundaries. Continuation lines preserve the original leading indentation.
func wrapLine(line string, width int) []string {
	if width <= 0 || lipgloss.Width(line) <= width {
		return []string{line}
	}

	// Detect leading whitespace so continuation lines stay aligned.
	trimmed := strings.TrimLeft(line, " \t")
	indentLen := len(line) - len(trimmed)
	indent := line[:indentLen]

	words := strings.Fields(line)
	if len(words) == 0 {
		return []string{line}
	}

	var result []string
	current := ""

	for _, word := range words {
		var candidate string
		if current == "" {
			candidate = indent + word
		} else {
			candidate = current + " " + word
		}

		if lipgloss.Width(candidate) > width && current != "" {
			result = append(result, current)
			current = indent + word
		} else {
			current = candidate
		}
	}
	if current != "" {
		result = append(result, current)
	}
	if len(result) == 0 {
		return []string{line}
	}
	return result
}

// renderCharVisualLine renders a line with a character-level selection highlight.
// The prefix (line number) is rendered in cursorLineStyle, and the selected
// [lo:hi+1] byte range within the line content is highlighted with visualLineStyle+Bold.
func renderCharVisualLine(prefix, line string, lo, hi, totalWidth int) string {
	charStyle := lipgloss.NewStyle().Background(colorVisualBg).Bold(true)
	baseStyle := cursorLineStyle

	n := len(line)
	if lo > n {
		lo = n
	}
	if hi >= n {
		hi = n - 1
	}

	if lo > hi || n == 0 {
		return baseStyle.Width(totalWidth).Render(prefix + line)
	}

	before := line[:lo]
	selected := line[lo : hi+1]
	after := ""
	if hi+1 < n {
		after = line[hi+1:]
	}

	renderedPrefix := baseStyle.Render(prefix)
	renderedBefore := baseStyle.Render(before)
	renderedSel := charStyle.Render(selected)
	renderedAfter := baseStyle.Render(after)

	combined := renderedPrefix + renderedBefore + renderedSel + renderedAfter
	// Pad to totalWidth.
	visible := lipgloss.Width(combined)
	if visible < totalWidth {
		combined += baseStyle.Render(strings.Repeat(" ", totalWidth-visible))
	}
	return combined
}

// styledPlanLine applies basic lipgloss styling to markdown-like plan lines.
func styledPlanLine(line string) string {
	trimmed := strings.TrimSpace(line)
	switch {
	case strings.HasPrefix(trimmed, "# "):
		return lipgloss.NewStyle().Bold(true).Foreground(colorHeader).Render(line)
	case strings.HasPrefix(trimmed, "## "):
		return lipgloss.NewStyle().Bold(true).Foreground(colorHeader).Render(line)
	case strings.HasPrefix(trimmed, "### "):
		return lipgloss.NewStyle().Bold(true).Render(line)
	default:
		return line
	}
}
