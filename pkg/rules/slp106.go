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
var slp106Release = regexp.MustCompile(`(?i)\b(?:Close|Release|Disconnect|Shutdown)\(`)
var slp106DeferClose = regexp.MustCompile(`(?i)\bdefer\s+`)
var slp106VarAssign = regexp.MustCompile(`(?i)(\w+)\s*(?:,\s*\w+\s*)?:=\s*.*?(?:os\.Open|OpenFile|sql\.Open|Connect|Acquire|Dial|Listen|NewClient|NewConsumer|NewProducer)\(`)
var slp106VarCall = regexp.MustCompile(`(?i)\b(\w+)\.(?:Close|Release|Disconnect|Shutdown)\(`)

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
			// Track acquired resources by variable name
			acquired := make(map[string]diff.Line) // varName -> line

			for _, ln := range h.Lines {
				if ln.Kind != diff.LineAdd {
					continue
				}
				clean := stripCommentAndStrings(ln.Content)
				content := strings.TrimSpace(ln.Content)

				// Check for acquisition with variable assignment
				if slp106Acquire.MatchString(clean) {
					// Try to extract the variable being assigned
					if match := slp106VarAssign.FindStringSubmatch(clean); match != nil && len(match) >= 2 {
						varName := match[1]
						acquired[varName] = ln
					} else {
						// Acquisition without clear variable — use line number as key
						varName := "__line__" + string(rune(ln.NewLineNo))
						acquired[varName] = ln
					}
				}

				// Check for release on a specific variable
				if slp106Release.MatchString(clean) || slp106DeferClose.MatchString(clean) {
					if match := slp106VarCall.FindStringSubmatch(clean); match != nil && len(match) >= 2 {
						varName := match[1]
						delete(acquired, varName)
					} else if slp106DeferClose.MatchString(content) {
						// Generic defer close without variable — try to match any single acquire
						// This handles cases like "defer conn.Close()" where conn is a parameter
						if len(acquired) == 1 {
							// Only one acquire, assume this closes it
							for k := range acquired {
								delete(acquired, k)
								break
							}
						}
					}
				}
			}

			// Emit findings for any remaining unmatched acquires.
			for _, ln := range acquired {
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
