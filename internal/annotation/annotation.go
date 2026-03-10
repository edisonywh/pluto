package annotation

import "fmt"

// AnnotationType classifies what kind of feedback an annotation represents.
type AnnotationType int

const (
	Comment AnnotationType = iota
	Delete
	Replace
)

func (t AnnotationType) String() string {
	switch t {
	case Comment:
		return "COMMENT"
	case Delete:
		return "DELETE"
	case Replace:
		return "REPLACE"
	default:
		return "UNKNOWN"
	}
}

// LineRange is a 1-indexed inclusive range of plan lines.
type LineRange struct {
	Start int
	End   int
}

func (r LineRange) String() string {
	if r.Start == r.End {
		return fmt.Sprintf("L%d", r.Start)
	}
	return fmt.Sprintf("L%d-%d", r.Start, r.End)
}

// Annotation attaches a review comment to a range of plan lines.
type Annotation struct {
	Type    AnnotationType
	Range   LineRange
	Text    string // non-empty only for char-level selections
	Message string
}
