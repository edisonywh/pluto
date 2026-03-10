package tui

import "github.com/charmbracelet/lipgloss"

var (
	// Palette
	colorDim        = lipgloss.Color("240")
	colorCursorBg   = lipgloss.Color("237")
	colorVisualBg   = lipgloss.Color("54")
	colorHeader     = lipgloss.Color("69")
	colorBorder     = lipgloss.Color("240")
	colorAdd        = lipgloss.Color("40")
	colorRemove     = lipgloss.Color("196")
	colorHunk       = lipgloss.Color("33")
	colorAnnotation = lipgloss.Color("214")
	colorReject     = lipgloss.Color("196")

	// Header bar across the full width
	headerStyle = lipgloss.NewStyle().
			Background(lipgloss.Color("235")).
			Foreground(lipgloss.Color("252")).
			Bold(true)

	// Status bar at the bottom
	statusBarStyle = lipgloss.NewStyle().
			Background(lipgloss.Color("235")).
			Foreground(lipgloss.Color("246"))

	// Annotation input bar shown in ModeAnnotate
	annotateBarStyle = lipgloss.NewStyle().
				Background(lipgloss.Color("57")).
				Foreground(lipgloss.Color("255")).
				Bold(true)

	// Line number gutter
	lineNumStyle = lipgloss.NewStyle().
			Foreground(colorDim).
			Width(4).
			Align(lipgloss.Right)

	// Cursor line highlight
	cursorLineStyle = lipgloss.NewStyle().
			Background(colorCursorBg).
			Bold(true)

	// Visual selection highlight
	visualLineStyle = lipgloss.NewStyle().
			Background(colorVisualBg)

	// Section header inside a panel
	panelTitleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(colorHeader)

	// Dim helper text
	dimStyle = lipgloss.NewStyle().
			Foreground(colorDim)

	// Diff line styles
	diffAddStyle = lipgloss.NewStyle().
			Foreground(colorAdd)

	diffRemoveStyle = lipgloss.NewStyle().
			Foreground(colorRemove)

	diffHunkStyle = lipgloss.NewStyle().
			Foreground(colorHunk).
			Bold(true)

	diffHeaderStyle = lipgloss.NewStyle().
			Foreground(colorDim).
			Italic(true)

	// Annotation list item styles
	annotationCommentStyle = lipgloss.NewStyle().
				Foreground(colorAnnotation)

	annotationDeleteStyle = lipgloss.NewStyle().
				Foreground(colorReject)

	annotationReplaceStyle = lipgloss.NewStyle().
				Foreground(colorHunk)

	// Help text
	helpKeyStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("252")).
			Bold(true)

	helpDescStyle = lipgloss.NewStyle().
			Foreground(colorDim)
)
