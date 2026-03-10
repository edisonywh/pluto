package annotation

import (
	"fmt"
	"sort"
	"strings"
)

// Format converts a slice of annotations into a structured deny message string.
func Format(annotations []Annotation) string {
	if len(annotations) == 0 {
		return ""
	}

	sorted := make([]Annotation, len(annotations))
	copy(sorted, annotations)
	sort.Slice(sorted, func(i, j int) bool {
		return sorted[i].Range.Start < sorted[j].Range.Start
	})

	var sb strings.Builder
	sb.WriteString("# Plan Feedback\n\n")
	for _, a := range sorted {
		ref := ""
		if a.Text != "" {
			ref = fmt.Sprintf(" (re: %q)", a.Text)
		}
		switch a.Type {
		case Delete:
			fmt.Fprintf(&sb, "[%s] DELETE%s\n", a.Range.String(), ref)
		case Comment:
			fmt.Fprintf(&sb, "[%s] COMMENT%s: %s\n", a.Range.String(), ref, a.Message)
		case Replace:
			fmt.Fprintf(&sb, "[%s] REPLACE%s: %s\n", a.Range.String(), ref, a.Message)
		}
	}
	return sb.String()
}
