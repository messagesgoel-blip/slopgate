package rules

import (
	"regexp"
	"strings"

	"github.com/messagesgoel-blip/slopgate/pkg/diff"
)

// SLP065 flags ignored error returns in Go. If a function call returns
// an error but the next added line does not check it, we flag.
//
// Heuristics:
//   - `_ = ...`, `_ := ...`, or `_, _ = ...` on the LHS of a function call.
//   - `err` assigned but not followed by `if err != nil` in the same hunk.
//   - Inline `if err := doSomething(); err != nil { ... }` is treated as handled.
//   - `_, err := doSomething()` where `err` is named on LHS is treated as handled.
//
// Scope: Go files only.
type SLP065 struct{}

func (SLP065) ID() string                { return "SLP065" }
func (SLP065) DefaultSeverity() Severity { return SeverityWarn }
func (SLP065) Description() string {
	return "returned error is ignored — handle or explicitly suppress with _"
}

var slp065ErrAssignLHS = regexp.MustCompile(`(^|[^a-zA-Z0-9_])err\s*(:=|=)`)
var slp065FuncCall = regexp.MustCompile(`\w+\s*\(`)
var slp065BlankLHS = regexp.MustCompile(`(^|[^a-zA-Z0-9_])_\s*(:=|=|,)`)
var slp065ErrCheck = regexp.MustCompile(`if\s+err\s*!=\s*nil`)
var slp065InlineErrInit = regexp.MustCompile(`if\s+[^;]*;\s*err\s*!=\s*nil`)
var slp065ExplicitSuppression = regexp.MustCompile(`^\s*_\s*=`)
var slp065NamedErrOnLHS = regexp.MustCompile(`_\s*,\s*err\s*:?=`)

func (r SLP065) Check(d *diff.Diff) []Finding {
	var out []Finding
	for _, f := range d.Files {
		if f.IsDelete || !isGoFile(f.Path) {
			continue
		}
		for _, h := range f.Hunks {
			lines := h.Lines
			for i, ln := range lines {
				if ln.Kind != diff.LineAdd {
					continue
				}
				trimmed := strings.TrimSpace(ln.Content)
				if trimmed == "" || strings.HasPrefix(trimmed, "//") {
					continue
				}

				// Inline `if err := ... ; err != nil { ... }` — always handled.
				if slp065InlineErrInit.MatchString(ln.Content) {
					continue
				}

				namedErrOnLHS := slp065NamedErrOnLHS.MatchString(ln.Content)

				// `_ = someFunc()` — explicit suppression, skip.
				// `_, _ = someFunc()` — blank tuple assign, flag unless explicit `_ =`.
				if !namedErrOnLHS && slp065BlankLHS.MatchString(ln.Content) && slp065FuncCall.MatchString(ln.Content) {
					if !slp065ExplicitSuppression.MatchString(ln.Content) {
						out = append(out, Finding{
							RuleID:   r.ID(),
							Severity: r.DefaultSeverity(),
							File:     f.Path,
							Line:     ln.NewLineNo,
							Message:  "error return ignored — handle or explicitly suppress with _",
							Snippet:  trimmed,
						})
					}
					continue
				}

				// `err` is assigned on this line but next added line is not `if err != nil`.
				if slp065ErrAssignLHS.MatchString(ln.Content) {
					if !slp065FuncCall.MatchString(ln.Content) {
						continue
					}
					if slp065ErrCheck.MatchString(ln.Content) {
						continue
					}
					if nextAdded := nextAddedLine(lines, i+1); nextAdded != nil {
						if slp065ErrCheck.MatchString(nextAdded.Content) {
							continue
						}
					}
					out = append(out, Finding{
						RuleID:   r.ID(),
						Severity: r.DefaultSeverity(),
						File:     f.Path,
						Line:     ln.NewLineNo,
						Message:  "error return ignored — handle or explicitly suppress with _",
						Snippet:  trimmed,
					})
				}
			}
		}
	}
	return out
}

func nextAddedLine(lines []diff.Line, idx int) *diff.Line {
	for i := idx; i < len(lines); i++ {
		if lines[i].Kind == diff.LineAdd {
			return &lines[i]
		}
	}
	return nil
}
