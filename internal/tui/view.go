package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// View implements tea.Model.
func (m Model) View() string {
	if !m.ready {
		return "Loading pluto...\n"
	}

	if m.showHelp {
		return lipgloss.Place(m.windowWidth, m.windowHeight,
			lipgloss.Center, lipgloss.Center,
			helpOverlayView())
	}

	header := m.renderHeader()
	footer := m.renderFooter()

	headerH := lipgloss.Height(header)
	footerH := lipgloss.Height(footer)
	contentH := m.windowHeight - headerH - footerH
	if contentH < 1 {
		contentH = 1
	}

	// Left panel gets 60% of width; right gets the rest.
	// The separator "│" is 1 character, so account for it.
	leftWidth := m.windowWidth * 3 / 5
	rightWidth := m.windowWidth - leftWidth - 1

	var leftContent string
	if m.mode == ModeDiff {
		leftContent = renderDiffPanel(m.diffLines, m.diffScrollOffset, leftWidth, contentH)
	} else {
		leftContent = renderPlanPanel(m, leftWidth, contentH)
	}
	rightContent := renderAnnotationsPanel(m, rightWidth, contentH)

	// Pad each panel to contentH lines so JoinHorizontal aligns them.
	leftLines := ensureHeight(leftContent, contentH)
	rightLines := ensureHeight(rightContent, contentH)

	// Build two-column layout with a │ separator.
	var content strings.Builder
	for i := 0; i < contentH; i++ {
		left := ""
		right := ""
		if i < len(leftLines) {
			left = leftLines[i]
		}
		if i < len(rightLines) {
			right = rightLines[i]
		}
		// Pad left column to leftWidth
		left = padToWidth(left, leftWidth)
		content.WriteString(left + dimStyle.Render("│") + right + "\n")
	}

	return lipgloss.JoinVertical(lipgloss.Left, header, content.String(), footer)
}

// renderHeader renders the top bar with session and mode info.
func (m Model) renderHeader() string {
	left := fmt.Sprintf("  pluto — Plan Review  [session: %s]", m.sessionID)
	var right string
	if len(m.annotations) > 0 {
		right = fmt.Sprintf("[%s | %d lines | %d ann]  ", m.mode.String(), len(m.planLines), len(m.annotations))
	} else {
		right = fmt.Sprintf("[%s | %d lines]  ", m.mode.String(), len(m.planLines))
	}

	gap := m.windowWidth - len(left) - len(right)
	if gap < 0 {
		gap = 0
	}
	content := left + strings.Repeat(" ", gap) + right
	return headerStyle.Width(m.windowWidth).Render(content)
}

// renderFooter renders either the annotation input bar, help text, or status bar.
func (m Model) renderFooter() string {
	switch m.mode {
	case ModeAnnotate:
		var label string
		if m.pendingCharText != "" {
			label = fmt.Sprintf("  [%s L%d (re: %q)] > ", m.annotationType.String(), m.cursor+1, m.pendingCharText)
		} else {
			r := visualRange(m.visualStart, m.cursor)
			label = fmt.Sprintf("  [%s %s] > ", m.annotationType.String(), r.String())
		}
		return annotateBarStyle.Width(m.windowWidth).Render(label + m.textInput.View())
	}

	return m.renderStatusBar()
}

func helpOverlayView() string {
	title := lipgloss.NewStyle().Bold(true).Foreground(colorHeader).Render("Key Bindings") +
		dimStyle.Render("  (? / q / esc to close)")

	section := func(s string) string {
		return lipgloss.NewStyle().Bold(true).Foreground(colorAnnotation).Render(s)
	}
	entry := func(keys, desc string) string {
		k := lipgloss.NewStyle().Foreground(lipgloss.Color("255")).Width(20).Render(keys)
		return k + dimStyle.Render(desc)
	}

	left := strings.Join([]string{
		section("Navigation"),
		entry("k / ↑", "up"),
		entry("j / ↓", "down"),
		entry("g", "top"),
		entry("G", "bottom"),
		entry("ctrl+u / ctrl+d", "half page up/down"),
		entry("w / b", "next/prev non-blank"),
		entry("{ / }", "paragraph boundary"),
		"",
		section("Review"),
		entry("D", "toggle diff"),
		entry("A", "approve plan"),
		entry("R", "reject with annotations"),
		entry("?", "toggle this help"),
	}, "\n")

	right := strings.Join([]string{
		section("Line Select  (V)"),
		entry("j / k  w / b  { / }", "extend selection"),
		entry("c", "comment on lines"),
		entry("x", "delete lines"),
		entry("r", "replace lines"),
		entry("esc", "cancel"),
		"",
		section("Char Select  (v)"),
		entry("h / l  w / b", "extend selection"),
		entry("c / r", "comment / replace"),
		entry("x", "delete text"),
		"",
		section("Operators  (normal mode)"),
		entry("d / c / r + motion", "delete / comment / replace"),
		entry("dd / cc / rr", "current line"),
		entry("daw", "delete word"),
		"",
		section("Annotations Pane  (tab)"),
		entry("j / k", "navigate"),
		entry("x / dd", "delete annotation"),
		entry("tab / esc", "back to plan"),
	}, "\n")

	leftCol := lipgloss.NewStyle().Width(44).Render(left)
	rightCol := lipgloss.NewStyle().Width(44).Render(right)
	body := lipgloss.JoinHorizontal(lipgloss.Top, leftCol, rightCol)

	return lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(colorHeader).
		Padding(1, 2).
		Render(title + "\n\n" + body)
}

func (m Model) renderStatusBar() string {
	var bindings string
	switch m.mode {
	case ModeNormal:
		if m.pendingOp != "" {
			bindings = fmt.Sprintf("  [%s%s_]  j/k/w/b/G:range  esc:cancel", m.pendingOp, m.countStr)
		} else {
			bindings = "  k/j:scroll  w/b:word  {/}:para  v:char-select  V:line-select  d/c/r:op  D:diff  u:undo  A:approve  R:reject  ?:help"
		}
	case ModeVisual:
		bindings = "  k/j/w/b/{/}:extend  c:comment  x/d:delete  r:replace  esc:cancel"
	case ModeVisualChar:
		bindings = "  h/l:char  w/b:word  c:comment  x:delete  r:replace  j/k:→line  esc:cancel"
	case ModeDiff:
		bindings = "  k/j:scroll  D/esc:back to plan"
	case ModeAnnotations:
		bindings = "  k/j:move  x/dd:delete  tab/esc:back"
	default:
		bindings = ""
	}
	return statusBarStyle.Width(m.windowWidth).Render(bindings)
}


// ensureHeight splits content into lines and pads/trims to exactly height lines.
func ensureHeight(content string, height int) []string {
	lines := strings.Split(strings.TrimRight(content, "\n"), "\n")
	for len(lines) < height {
		lines = append(lines, "")
	}
	return lines[:height]
}

// padToWidth pads s with spaces on the right to reach exactly w visible characters.
// It accounts for ANSI escape codes by using lipgloss.Width.
func padToWidth(s string, w int) string {
	visible := lipgloss.Width(s)
	if visible >= w {
		return s
	}
	return s + strings.Repeat(" ", w-visible)
}
