package diff

import (
	"github.com/aymanbagabas/go-udiff"
)

// Compute returns a unified diff string between prev and curr.
// Returns "" if the content is identical or if diff fails.
func Compute(prev, curr string) string {
	edits := udiff.Strings(prev, curr)
	if len(edits) == 0 {
		return ""
	}
	unified, err := udiff.ToUnified("previous", "current", prev, edits, 3)
	if err != nil {
		return ""
	}
	return unified
}
