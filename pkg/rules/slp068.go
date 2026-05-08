package rules

import (
	"strings"

	"github.com/messagesgoel-blip/slopgate/pkg/diff"
)

// SLP068 flags duplicate 8-line code blocks within the same file.
type SLP068 struct{}

func (SLP068) ID() string                { return "SLP068" }
func (SLP068) DefaultSeverity() Severity { return SeverityWarn }
func (SLP068) Description() string {
	return "duplicate logic block within the same file"
}

const slp068Window = 8

func windowKey(lines []diff.Line, start int) string {
	var b strings.Builder
	for i := start; i < start+slp068Window && i < len(lines); i++ {
		if i > start {
			b.WriteByte('\n')
		}
		b.WriteString(lines[i].Content)
	}
	return b.String()
}

func (r SLP068) Check(d *diff.Diff) []Finding {
	var out []Finding
	for _, f := range d.Files {
		if f.IsDelete || isDocFile(f.Path) || !isSourceLikeFile(f.Path) {
			continue
		}
		if isTestFile(f.Path) {
			continue
		}
		added := f.AddedLines()
		if len(added) < slp068Window {
			continue
		}
		seen := make(map[string]bool)
		flagged := make(map[int]bool)
		lastFlaggedLineByKey := make(map[string]int)
		for i := 0; i <= len(added)-slp068Window; i++ {
			key := windowKey(added, i)
			if len(strings.TrimSpace(key)) < 20 {
				continue
			}
			if seen[key] {
				lineNo := added[i].NewLineNo
				last, ok := lastFlaggedLineByKey[key]
				if !flagged[lineNo] && (!ok || lineNo-last >= slp068Window) {
					flagged[lineNo] = true
					lastFlaggedLineByKey[key] = lineNo
					out = append(out, Finding{
						RuleID:   r.ID(),
						Severity: r.DefaultSeverity(),
						File:     f.Path,
						Line:     lineNo,
						Message:  "duplicate code block within the same file — extract to helper function",
						Snippet:  strings.TrimSpace(added[i].Content),
					})
				}
			} else {
				seen[key] = true
			}
		}
	}
	return out
}
