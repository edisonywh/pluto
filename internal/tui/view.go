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
	right := fmt.Sprintf("[%s | %d lines]  ", m.mode.String(), len(m.planLines))

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

	if m.showHelp {
		return helpStyle(m).Width(m.windowWidth).Render(m.renderHelp())
	}
	return m.renderStatusBar()
}

func helpStyle(m Model) lipgloss.Style {
	_ = m
	return lipgloss.NewStyle().
		Background(lipgloss.Color("235")).
		Foreground(colorDim).
		Padding(0, 1)
}

func (m Model) renderStatusBar() string {
	var bindings string
	switch m.mode {
	case ModeNormal:
		if m.pendingOp != "" {
			bindings = fmt.Sprintf("  [%s%s_]  j/k/w/b/G:range  esc:cancel", m.pendingOp, m.countStr)
		} else {
			bindings = "  k/j:scroll  w/b:word  {/}:para  v:char-select  V:line-select  d/c/r:op  D:diff  A:approve  R:reject  ?:help"
		}
	case ModeVisual:
		bindings = "  k/j/w/b/{/}:extend  c:comment  x:delete  r:replace  esc:cancel"
	case ModeVisualChar:
		bindings = "  h/l:char  w/b:word  c:comment  x:delete  r:replace  j/k:→line  esc:cancel"
	case ModeDiff:
		bindings = "  k/j:scroll  D/esc:back to plan"
	default:
		bindings = ""
	}
	return statusBarStyle.Width(m.windowWidth).Render(bindings)
}

func (m Model) renderHelp() string {
	var rows []string
	for _, group := range m.keyMap.FullHelp() {
		var parts []string
		for _, b := range group {
			h := b.Help()
			parts = append(parts, helpKeyStyle.Render(h.Key)+helpDescStyle.Render(":"+h.Desc))
		}
		rows = append(rows, strings.Join(parts, "  "))
	}
	return strings.Join(rows, "\n")
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
