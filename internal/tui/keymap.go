package tui

import "github.com/charmbracelet/bubbles/key"

// KeyMap holds all key bindings for pluto.
type KeyMap struct {
	Up               key.Binding
	Down             key.Binding
	Top              key.Binding
	Bottom           key.Binding
	HalfUp           key.Binding
	HalfDown         key.Binding
	WordForward      key.Binding
	WordBackward     key.Binding
	ParaDown         key.Binding
	ParaUp           key.Binding
	Visual           key.Binding
	VisualLine       key.Binding
	CharLeft         key.Binding
	CharRight        key.Binding
	Diff             key.Binding
	Approve          key.Binding
	Reject           key.Binding
	Help             key.Binding
	Comment          key.Binding
	Delete           key.Binding
	Replace          key.Binding
	Cancel           key.Binding
	Confirm          key.Binding
	FocusAnnotations key.Binding
}

// DefaultKeyMap returns the default vim-style key bindings.
func DefaultKeyMap() KeyMap {
	return KeyMap{
		Up: key.NewBinding(
			key.WithKeys("k", "up"),
			key.WithHelp("k/↑", "up"),
		),
		Down: key.NewBinding(
			key.WithKeys("j", "down"),
			key.WithHelp("j/↓", "down"),
		),
		Top: key.NewBinding(
			key.WithKeys("g"),
			key.WithHelp("g", "top"),
		),
		Bottom: key.NewBinding(
			key.WithKeys("G"),
			key.WithHelp("G", "bottom"),
		),
		HalfUp: key.NewBinding(
			key.WithKeys("ctrl+u"),
			key.WithHelp("ctrl+u", "half up"),
		),
		HalfDown: key.NewBinding(
			key.WithKeys("ctrl+d"),
			key.WithHelp("ctrl+d", "half down"),
		),
		WordForward: key.NewBinding(
			key.WithKeys("w"),
			key.WithHelp("w", "next non-blank"),
		),
		WordBackward: key.NewBinding(
			key.WithKeys("b"),
			key.WithHelp("b", "prev non-blank"),
		),
		ParaDown: key.NewBinding(
			key.WithKeys("}"),
			key.WithHelp("}", "next paragraph"),
		),
		ParaUp: key.NewBinding(
			key.WithKeys("{"),
			key.WithHelp("{", "prev paragraph"),
		),
		Visual: key.NewBinding(
			key.WithKeys("v"),
			key.WithHelp("v", "visual char select"),
		),
		VisualLine: key.NewBinding(
			key.WithKeys("V"),
			key.WithHelp("V", "visual line select"),
		),
		CharLeft: key.NewBinding(
			key.WithKeys("h"),
			key.WithHelp("h", "char left"),
		),
		CharRight: key.NewBinding(
			key.WithKeys("l"),
			key.WithHelp("l", "char right"),
		),
		Diff: key.NewBinding(
			key.WithKeys("D"),
			key.WithHelp("D", "toggle diff"),
		),
		Approve: key.NewBinding(
			key.WithKeys("A"),
			key.WithHelp("A", "approve"),
		),
		Reject: key.NewBinding(
			key.WithKeys("R"),
			key.WithHelp("R", "reject"),
		),
		Help: key.NewBinding(
			key.WithKeys("?"),
			key.WithHelp("?", "help"),
		),
		Comment: key.NewBinding(
			key.WithKeys("c"),
			key.WithHelp("c", "comment"),
		),
		Delete: key.NewBinding(
			key.WithKeys("x"),
			key.WithHelp("x", "delete"),
		),
		Replace: key.NewBinding(
			key.WithKeys("r"),
			key.WithHelp("r", "replace"),
		),
		Cancel: key.NewBinding(
			key.WithKeys("esc"),
			key.WithHelp("esc", "cancel"),
		),
		Confirm: key.NewBinding(
			key.WithKeys("enter"),
			key.WithHelp("enter", "confirm"),
		),
		FocusAnnotations: key.NewBinding(
			key.WithKeys("tab"),
			key.WithHelp("tab", "focus annotations"),
		),
	}
}

// ShortHelp returns the abbreviated help binding list.
func (k KeyMap) ShortHelp() []key.Binding {
	return []key.Binding{k.Up, k.Down, k.Visual, k.Diff, k.Approve, k.Reject, k.FocusAnnotations, k.Help}
}

// FullHelp returns the grouped full help binding list.
func (k KeyMap) FullHelp() [][]key.Binding {
	return [][]key.Binding{
		{k.Up, k.Down, k.Top, k.Bottom, k.HalfUp, k.HalfDown},
		{k.WordForward, k.WordBackward, k.ParaDown, k.ParaUp},
		{k.Visual, k.VisualLine, k.CharLeft, k.CharRight, k.Comment, k.Delete, k.Replace},
		{k.Diff, k.Approve, k.Reject, k.FocusAnnotations, k.Help, k.Cancel, k.Confirm},
	}
}
