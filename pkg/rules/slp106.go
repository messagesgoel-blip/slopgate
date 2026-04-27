package rules

import (
	"regexp"
	"strings"

	"github.com/messagesgoel-blip/slopgate/pkg/diff"
)

// SLP106 flags resource acquisition functions (Open, Connect, Acquire, Listen,
// Dial) without a corresponding release/close/defer in the same hunk.
// alenAI slop pattern: agents open connections but forget cleanup.
type SLP106 struct{}

func (SLP106) ID() string                { return "SLP106" }
func (SLP106) DefaultSeverity() Severity { return SeverityWarn }
func (SLP106) Description() string {
	return "resource acquired without release/close in scope — add deferred cleanup"
}

var slp106Acquire = regexp.MustCompile(`(?i)\b(?:os\.Open|OpenFile|sql\.Open|Connect|Acquire|Dial|Listen|NewClient|NewConsumer|NewProducer)\(`)
var slp106Release = regexp.MustCompile(`(?i)\b(?:Close|Release|Disconnect|Shutdown|defer.*\bclose|defer.*\bcancel)\(`)

func (r SLP106) Check(d *diff.Diff) []Finding {
	var out []Finding
	for _, f := range d.Files {
		if f.IsDelete || isDocFile(f.Path) {
			continue
		}
		if !isGoFile(f.Path) && !isJSOrTSFile(f.Path) && !isPythonFile(f.Path) && !isJavaFile(f.Path) && !isRustFile(f.Path) {
			continue
		}

		for _, h := range f.Hunks {
			var acquireLines []diff.Line
			for _, ln := range h.Lines {
				if ln.Kind != diff.LineAdd {
					continue
				}
				clean := stripCommentAndStrings(ln.Content)
				if slp106Acquire.MatchString(clean) {
					acquireLines = append(acquireLines, ln)
				}
				// Pop the most recently unmatched acquire when a release is seen.
				if slp106Release.MatchString(clean) && len(acquireLines) > 0 {
					acquireLines = acquireLines[:len(acquireLines)-1]
				}
			}
			// Emit findings for any remaining unmatched acquires.
			for _, ln := range acquireLines {
				out = append(out, Finding{
					RuleID:   r.ID(),
					Severity: r.DefaultSeverity(),
					File:     f.Path,
					Line:     ln.NewLineNo,
					Message:  "resource acquired without visible release/close in this hunk — add deferred cleanup",
					Snippet:  strings.TrimSpace(ln.Content),
				})
			}
		}
	}
	return out
}
