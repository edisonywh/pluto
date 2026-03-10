package tui

import (
	"strconv"
	"strings"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"pluto/internal/annotation"
)

// Update implements tea.Model.
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.windowWidth = msg.Width
		m.windowHeight = msg.Height
		m.ready = true
		return m, nil

	case tea.KeyMsg:
		if m.mode == ModeAnnotate {
			return m.handleAnnotateKey(msg)
		}
		return m.handleModeKey(msg)

	default:
		// Delegate non-key messages to textinput when in annotate mode
		// (needed for cursor blink animation)
		if m.mode == ModeAnnotate {
			var cmd tea.Cmd
			m.textInput, cmd = m.textInput.Update(msg)
			return m, cmd
		}
	}

	return m, nil
}

func (m Model) handleModeKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch m.mode {
	case ModeNormal:
		return m.handleNormalKey(msg)
	case ModeVisual:
		return m.handleVisualKey(msg)
	case ModeVisualChar:
		return m.handleVisualCharKey(msg)
	case ModeDiff:
		return m.handleDiffKey(msg)
	}
	return m, nil
}

func (m Model) handleNormalKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	contentH := m.contentHeight()
	lastLine := len(m.planLines) - 1
	keyStr := msg.String()

	// Operator-pending mode: route to motion handler.
	if m.pendingOp != "" {
		return m.handleOperatorMotion(msg, keyStr, contentH, lastLine)
	}

	// Digit accumulation (count prefix for motions/operators).
	if keyStr >= "1" && keyStr <= "9" || (keyStr == "0" && m.countStr != "") {
		m.countStr += keyStr
		return m, nil
	}

	// Consume any accumulated count for this key.
	count := 1
	if m.countStr != "" {
		if n, err := strconv.Atoi(m.countStr); err == nil && n > 0 {
			count = n
		}
		m.countStr = ""
	}

	switch {
	case key.Matches(msg, m.keyMap.Down):
		m.cursor = min(lastLine, m.cursor+count)
		m.scrollOffset = scrollToCursor(m.scrollOffset, m.cursor, contentH)

	case key.Matches(msg, m.keyMap.Up):
		m.cursor = max(0, m.cursor-count)
		m.scrollOffset = scrollToCursor(m.scrollOffset, m.cursor, contentH)

	case key.Matches(msg, m.keyMap.Top):
		m.cursor = 0
		m.scrollOffset = 0

	case key.Matches(msg, m.keyMap.Bottom):
		m.cursor = lastLine
		m.scrollOffset = max(0, lastLine-contentH+1)

	case key.Matches(msg, m.keyMap.HalfDown):
		half := contentH / 2
		m.cursor = min(lastLine, m.cursor+half)
		m.scrollOffset = scrollToCursor(m.scrollOffset, m.cursor, contentH)

	case key.Matches(msg, m.keyMap.HalfUp):
		half := contentH / 2
		m.cursor = max(0, m.cursor-half)
		m.scrollOffset = scrollToCursor(m.scrollOffset, m.cursor, contentH)

	case key.Matches(msg, m.keyMap.WordForward):
		m.cursor = nextNonBlank(m.planLines, m.cursor)
		m.scrollOffset = scrollToCursor(m.scrollOffset, m.cursor, contentH)

	case key.Matches(msg, m.keyMap.WordBackward):
		m.cursor = prevNonBlank(m.planLines, m.cursor)
		m.scrollOffset = scrollToCursor(m.scrollOffset, m.cursor, contentH)

	case key.Matches(msg, m.keyMap.ParaDown):
		m.cursor = nextParaBoundary(m.planLines, m.cursor)
		m.scrollOffset = scrollToCursor(m.scrollOffset, m.cursor, contentH)

	case key.Matches(msg, m.keyMap.ParaUp):
		m.cursor = prevParaBoundary(m.planLines, m.cursor)
		m.scrollOffset = scrollToCursor(m.scrollOffset, m.cursor, contentH)

	case key.Matches(msg, m.keyMap.Visual):
		m.mode = ModeVisualChar
		m.charAnchorCol = 0
		m.charCursorCol = 0

	case key.Matches(msg, m.keyMap.VisualLine):
		m.mode = ModeVisual
		m.visualStart = m.cursor

	// Operator-pending: enter pending mode, anchor visualStart.
	case keyStr == "d":
		m.pendingOp = "d"
		m.visualStart = m.cursor

	case keyStr == "c":
		m.pendingOp = "c"
		m.visualStart = m.cursor

	case keyStr == "r":
		m.pendingOp = "r"
		m.visualStart = m.cursor

	case key.Matches(msg, m.keyMap.Diff):
		if m.diffText != "" {
			m.mode = ModeDiff
			m.diffScrollOffset = 0
		}

	case key.Matches(msg, m.keyMap.Approve):
		m.result = Result{Decision: Approve}
		return m, tea.Quit

	case key.Matches(msg, m.keyMap.Reject):
		m.result = Result{Decision: Reject, Annotations: m.annotations}
		return m, tea.Quit

	case key.Matches(msg, m.keyMap.Help):
		m.showHelp = !m.showHelp
	}

	return m, nil
}

// handleOperatorMotion handles key presses while an operator (d/c/r) is pending.
func (m Model) handleOperatorMotion(msg tea.KeyMsg, keyStr string, contentH, lastLine int) (tea.Model, tea.Cmd) {
	// Handle daw sub-sequence.
	if m.pendingOp == "da" {
		if keyStr == "w" {
			start := m.visualStart
			m.pendingOp = ""
			m.countStr = ""
			return m.applyOperator("d", start, start, contentH, lastLine)
		}
		m.pendingOp = ""
		m.countStr = ""
		return m, nil
	}

	// Digit accumulation inside pending mode.
	if keyStr >= "1" && keyStr <= "9" || (keyStr == "0" && m.countStr != "") {
		m.countStr += keyStr
		return m, nil
	}

	count := 1
	if m.countStr != "" {
		if n, err := strconv.Atoi(m.countStr); err == nil && n > 0 {
			count = n
		}
	}

	switch {
	case keyStr == "esc":
		m.pendingOp = ""
		m.countStr = ""
		return m, nil

	// daw sub-sequence: d + a → wait for w.
	case m.pendingOp == "d" && keyStr == "a":
		m.pendingOp = "da"
		return m, nil

	// Repeated operator (dd/cc/rr) → current line only.
	case keyStr == m.pendingOp:
		op := m.pendingOp
		start := m.visualStart
		// Respect a count prefix: e.g. 3dd deletes count lines from cursor.
		end := min(lastLine, start+count-1)
		m.pendingOp = ""
		m.countStr = ""
		return m.applyOperator(op, start, end, contentH, lastLine)

	// j motion.
	case key.Matches(msg, m.keyMap.Down):
		newCursor := min(lastLine, m.cursor+count)
		m.cursor = newCursor
		m.scrollOffset = scrollToCursor(m.scrollOffset, m.cursor, contentH)
		op := m.pendingOp
		start := m.visualStart
		m.pendingOp = ""
		m.countStr = ""
		return m.applyOperator(op, start, m.cursor, contentH, lastLine)

	// k motion.
	case key.Matches(msg, m.keyMap.Up):
		newCursor := max(0, m.cursor-count)
		m.cursor = newCursor
		m.scrollOffset = scrollToCursor(m.scrollOffset, m.cursor, contentH)
		op := m.pendingOp
		start := m.visualStart
		m.pendingOp = ""
		m.countStr = ""
		return m.applyOperator(op, start, m.cursor, contentH, lastLine)

	// G motion — range from visualStart to last line.
	case key.Matches(msg, m.keyMap.Bottom):
		m.cursor = lastLine
		m.scrollOffset = max(0, lastLine-contentH+1)
		op := m.pendingOp
		start := m.visualStart
		m.pendingOp = ""
		m.countStr = ""
		return m.applyOperator(op, start, m.cursor, contentH, lastLine)

	// w motion.
	case key.Matches(msg, m.keyMap.WordForward):
		newCursor := m.cursor
		for i := 0; i < count; i++ {
			newCursor = nextNonBlank(m.planLines, newCursor)
		}
		m.cursor = newCursor
		m.scrollOffset = scrollToCursor(m.scrollOffset, m.cursor, contentH)
		op := m.pendingOp
		start := m.visualStart
		m.pendingOp = ""
		m.countStr = ""
		return m.applyOperator(op, start, m.cursor, contentH, lastLine)

	// b motion.
	case key.Matches(msg, m.keyMap.WordBackward):
		newCursor := m.cursor
		for i := 0; i < count; i++ {
			newCursor = prevNonBlank(m.planLines, newCursor)
		}
		m.cursor = newCursor
		m.scrollOffset = scrollToCursor(m.scrollOffset, m.cursor, contentH)
		op := m.pendingOp
		start := m.visualStart
		m.pendingOp = ""
		m.countStr = ""
		return m.applyOperator(op, start, m.cursor, contentH, lastLine)

	default:
		m.pendingOp = ""
		m.countStr = ""
		return m, nil
	}
}

// applyOperator fires the accumulated operator over [rangeStart, rangeEnd].
func (m Model) applyOperator(op string, rangeStart, rangeEnd, contentH, lastLine int) (tea.Model, tea.Cmd) {
	m.visualStart = rangeStart
	m.cursor = rangeEnd
	m.scrollOffset = scrollToCursor(m.scrollOffset, m.cursor, contentH)
	r := visualRange(rangeStart, rangeEnd)

	switch op {
	case "d":
		m.annotations = append(m.annotations, annotation.Annotation{
			Type:  annotation.Delete,
			Range: r,
		})
		return m, nil

	case "c":
		m.mode = ModeAnnotate
		m.annotationType = annotation.Comment
		m.textInput.SetValue("")
		m.textInput.Focus()
		return m, textinput.Blink

	case "r":
		m.mode = ModeAnnotate
		m.annotationType = annotation.Replace
		m.textInput.SetValue("")
		m.textInput.Focus()
		return m, textinput.Blink
	}

	return m, nil
}

func (m Model) handleVisualKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	contentH := m.contentHeight()
	lastLine := len(m.planLines) - 1

	switch {
	case key.Matches(msg, m.keyMap.Down):
		if m.cursor < lastLine {
			m.cursor++
			m.scrollOffset = scrollToCursor(m.scrollOffset, m.cursor, contentH)
		}

	case key.Matches(msg, m.keyMap.Up):
		if m.cursor > 0 {
			m.cursor--
			m.scrollOffset = scrollToCursor(m.scrollOffset, m.cursor, contentH)
		}

	case key.Matches(msg, m.keyMap.WordForward):
		m.cursor = nextNonBlank(m.planLines, m.cursor)
		m.scrollOffset = scrollToCursor(m.scrollOffset, m.cursor, contentH)

	case key.Matches(msg, m.keyMap.WordBackward):
		m.cursor = prevNonBlank(m.planLines, m.cursor)
		m.scrollOffset = scrollToCursor(m.scrollOffset, m.cursor, contentH)

	case key.Matches(msg, m.keyMap.ParaDown):
		m.cursor = nextParaBoundary(m.planLines, m.cursor)
		m.scrollOffset = scrollToCursor(m.scrollOffset, m.cursor, contentH)

	case key.Matches(msg, m.keyMap.ParaUp):
		m.cursor = prevParaBoundary(m.planLines, m.cursor)
		m.scrollOffset = scrollToCursor(m.scrollOffset, m.cursor, contentH)

	case key.Matches(msg, m.keyMap.Bottom):
		m.cursor = lastLine
		m.scrollOffset = max(0, lastLine-contentH+1)

	case key.Matches(msg, m.keyMap.Comment):
		m.mode = ModeAnnotate
		m.annotationType = annotation.Comment
		m.textInput.SetValue("")
		m.textInput.Focus()
		return m, textinput.Blink

	case key.Matches(msg, m.keyMap.Delete):
		r := visualRange(m.visualStart, m.cursor)
		m.annotations = append(m.annotations, annotation.Annotation{
			Type:  annotation.Delete,
			Range: r,
		})
		m.mode = ModeNormal

	case key.Matches(msg, m.keyMap.Replace):
		m.mode = ModeAnnotate
		m.annotationType = annotation.Replace
		m.textInput.SetValue("")
		m.textInput.Focus()
		return m, textinput.Blink

	case key.Matches(msg, m.keyMap.Cancel):
		m.mode = ModeNormal
	}

	return m, nil
}

func (m Model) handleAnnotateKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch {
	case key.Matches(msg, m.keyMap.Confirm):
		text := strings.TrimSpace(m.textInput.Value())
		var r annotation.LineRange
		if m.pendingCharText != "" {
			r = annotation.LineRange{Start: m.cursor + 1, End: m.cursor + 1}
		} else {
			r = visualRange(m.visualStart, m.cursor)
		}
		m.annotations = append(m.annotations, annotation.Annotation{
			Type:    m.annotationType,
			Range:   r,
			Text:    m.pendingCharText,
			Message: text,
		})
		m.pendingCharText = ""
		m.textInput.Blur()
		m.mode = ModeNormal
		return m, nil

	case key.Matches(msg, m.keyMap.Cancel):
		m.pendingCharText = ""
		m.textInput.Blur()
		m.mode = ModeNormal
		return m, nil

	default:
		var cmd tea.Cmd
		m.textInput, cmd = m.textInput.Update(msg)
		return m, cmd
	}
}

func (m Model) handleDiffKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	diffLineCount := len(m.diffLines)
	contentH := m.contentHeight()

	switch {
	case key.Matches(msg, m.keyMap.Down):
		maxOffset := max(0, diffLineCount-contentH)
		if m.diffScrollOffset < maxOffset {
			m.diffScrollOffset++
		}

	case key.Matches(msg, m.keyMap.Up):
		if m.diffScrollOffset > 0 {
			m.diffScrollOffset--
		}

	case key.Matches(msg, m.keyMap.Diff), key.Matches(msg, m.keyMap.Cancel):
		m.mode = ModeNormal
	}

	return m, nil
}

func (m Model) handleVisualCharKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	contentH := m.contentHeight()
	lastLine := len(m.planLines) - 1

	var line string
	if m.cursor < len(m.planLines) {
		line = m.planLines[m.cursor]
	}
	lastCol := 0
	if len(line) > 0 {
		lastCol = len(line) - 1
	}

	switch {
	case key.Matches(msg, m.keyMap.CharLeft):
		if m.charCursorCol > 0 {
			m.charCursorCol--
		}

	case key.Matches(msg, m.keyMap.CharRight):
		if m.charCursorCol < lastCol {
			m.charCursorCol++
		}

	case key.Matches(msg, m.keyMap.WordForward):
		m.charCursorCol = nextWordCol(line, m.charCursorCol)

	case key.Matches(msg, m.keyMap.WordBackward):
		m.charCursorCol = prevWordCol(line, m.charCursorCol)

	case key.Matches(msg, m.keyMap.Down):
		// Switch to line visual, extend down.
		m.mode = ModeVisual
		m.visualStart = m.cursor
		if m.cursor < lastLine {
			m.cursor++
			m.scrollOffset = scrollToCursor(m.scrollOffset, m.cursor, contentH)
		}

	case key.Matches(msg, m.keyMap.Up):
		// Switch to line visual, extend up.
		m.mode = ModeVisual
		m.visualStart = m.cursor
		if m.cursor > 0 {
			m.cursor--
			m.scrollOffset = scrollToCursor(m.scrollOffset, m.cursor, contentH)
		}

	case key.Matches(msg, m.keyMap.Comment):
		m.pendingCharText = m.charVisualText()
		m.mode = ModeAnnotate
		m.annotationType = annotation.Comment
		m.textInput.SetValue("")
		m.textInput.Focus()
		return m, textinput.Blink

	case key.Matches(msg, m.keyMap.Replace):
		m.pendingCharText = m.charVisualText()
		m.mode = ModeAnnotate
		m.annotationType = annotation.Replace
		m.textInput.SetValue("")
		m.textInput.Focus()
		return m, textinput.Blink

	case key.Matches(msg, m.keyMap.Delete):
		r := annotation.LineRange{Start: m.cursor + 1, End: m.cursor + 1}
		m.annotations = append(m.annotations, annotation.Annotation{
			Type:  annotation.Delete,
			Range: r,
			Text:  m.charVisualText(),
		})
		m.mode = ModeNormal

	case key.Matches(msg, m.keyMap.Cancel):
		m.mode = ModeNormal
	}

	return m, nil
}

// nextWordCol returns the byte offset of the start of the next word after col in line.
func nextWordCol(line string, col int) int {
	n := len(line)
	if col >= n-1 {
		return col
	}
	i := col
	// Skip current word characters.
	for i < n && line[i] != ' ' && line[i] != '\t' {
		i++
	}
	// Skip whitespace.
	for i < n && (line[i] == ' ' || line[i] == '\t') {
		i++
	}
	if i >= n {
		return col
	}
	return i
}

// prevWordCol returns the byte offset of the start of the previous word before col in line.
func prevWordCol(line string, col int) int {
	if col <= 0 {
		return 0
	}
	i := col - 1
	// Skip whitespace backwards.
	for i > 0 && (line[i] == ' ' || line[i] == '\t') {
		i--
	}
	// Skip word characters backwards.
	for i > 0 && line[i-1] != ' ' && line[i-1] != '\t' {
		i--
	}
	return i
}

// contentHeight returns the number of lines available for the plan/diff panel content.
func (m Model) contentHeight() int {
	// header(1) + footer(1) + some padding = 3 lines overhead
	// We also subtract 1 for the panel title row rendered inside the panel
	h := m.windowHeight - 4
	if h < 1 {
		return 10 // safe fallback before first WindowSizeMsg
	}
	return h
}

// nextNonBlank returns the index of the next non-blank line after cursor.
// Returns cursor if no non-blank line exists forward.
func nextNonBlank(lines []string, cursor int) int {
	for i := cursor + 1; i < len(lines); i++ {
		if strings.TrimSpace(lines[i]) != "" {
			return i
		}
	}
	return cursor
}

// prevNonBlank returns the index of the previous non-blank line before cursor.
// Returns cursor if no non-blank line exists backward.
func prevNonBlank(lines []string, cursor int) int {
	for i := cursor - 1; i >= 0; i-- {
		if strings.TrimSpace(lines[i]) != "" {
			return i
		}
	}
	return cursor
}

// nextParaBoundary returns the index of the next blank line after cursor
// (paragraph boundary). Returns last line if none found.
func nextParaBoundary(lines []string, cursor int) int {
	for i := cursor + 1; i < len(lines); i++ {
		if strings.TrimSpace(lines[i]) == "" {
			return i
		}
	}
	return len(lines) - 1
}

// prevParaBoundary returns the index of the previous blank line before cursor
// (paragraph boundary). Returns 0 if none found.
func prevParaBoundary(lines []string, cursor int) int {
	for i := cursor - 1; i >= 0; i-- {
		if strings.TrimSpace(lines[i]) == "" {
			return i
		}
	}
	return 0
}
