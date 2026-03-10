package tui

import (
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"pluto/internal/annotation"
)

// Mode represents the current interaction mode of the TUI.
type Mode int

const (
	ModeNormal Mode = iota
	ModeVisual
	ModeVisualChar
	ModeAnnotate
	ModeDiff
)

func (m Mode) String() string {
	switch m {
	case ModeNormal:
		return "NORMAL"
	case ModeVisual:
		return "VISUAL"
	case ModeVisualChar:
		return "VISUAL CHAR"
	case ModeAnnotate:
		return "ANNOTATE"
	case ModeDiff:
		return "DIFF"
	default:
		return "UNKNOWN"
	}
}

// Decision is the final outcome chosen by the reviewer.
type Decision int

const (
	Pending Decision = iota
	Approve
	Reject
)

// Result holds the reviewer's final decision and any annotations.
type Result struct {
	Decision    Decision
	Annotations []annotation.Annotation
}

// Model is the bubbletea model for the plan reviewer TUI.
type Model struct {
	// Source data
	planText  string
	planLines []string
	diffText  string
	diffLines []string
	sessionID string

	// Key bindings
	keyMap KeyMap

	// Terminal size
	windowWidth  int
	windowHeight int

	// Plan viewport
	scrollOffset int
	cursor       int // 0-indexed current line
	visualStart  int // 0-indexed visual anchor

	// Annotations
	annotations    []annotation.Annotation
	annotationType annotation.AnnotationType

	// Current mode
	mode Mode

	// Text input for ModeAnnotate
	textInput textinput.Model

	// Diff viewport
	diffScrollOffset int

	// Final result (populated on A or R)
	result Result

	// Char-visual selection state
	charAnchorCol   int    // column where v was pressed (byte offset)
	charCursorCol   int    // moving column cursor (byte offset)
	pendingCharText string // captured text, set before ModeAnnotate, cleared on confirm/cancel

	// Operator-pending state
	pendingOp string // "", "d", "c", "r", "da"
	countStr  string // digit accumulator e.g. "2", "12"

	// UI state
	showHelp bool
	ready    bool
}

// NewModel constructs a Model with the given plan and diff texts.
func NewModel(planText, diffText, sessionID string) Model {
	ti := textinput.New()
	ti.Placeholder = "Enter annotation message..."
	ti.CharLimit = 500

	return Model{
		planText:  planText,
		planLines: strings.Split(planText, "\n"),
		diffText:  diffText,
		diffLines: strings.Split(diffText, "\n"),
		sessionID: sessionID,
		keyMap:    DefaultKeyMap(),
		textInput: ti,
		mode:      ModeNormal,
	}
}

// Result returns the final decision and annotations after the TUI exits.
func (m Model) Result() Result {
	return m.result
}

// Init implements tea.Model.
func (m Model) Init() tea.Cmd {
	return textinput.Blink
}

// visualRange returns the 1-indexed LineRange for the current visual selection.
func visualRange(visualStart, cursor int) annotation.LineRange {
	start := min(visualStart, cursor) + 1
	end := max(visualStart, cursor) + 1
	return annotation.LineRange{Start: start, End: end}
}

// charVisualText returns the selected text in ModeVisualChar.
// It slices the current cursor line between the anchor and cursor columns.
func (m Model) charVisualText() string {
	if m.cursor >= len(m.planLines) {
		return ""
	}
	line := m.planLines[m.cursor]
	lo := min(m.charAnchorCol, m.charCursorCol)
	hi := max(m.charAnchorCol, m.charCursorCol)
	if lo >= len(line) {
		return ""
	}
	if hi >= len(line) {
		hi = len(line) - 1
	}
	return line[lo : hi+1]
}

// scrollToCursor returns a scroll offset that keeps cursor in view.
func scrollToCursor(offset, cursor, height int) int {
	if cursor < offset {
		return cursor
	}
	if cursor >= offset+height {
		return cursor - height + 1
	}
	return offset
}
