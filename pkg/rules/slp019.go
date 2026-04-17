package rules

import (
	"fmt"
	"strings"

	"github.com/messagesgoel-blip/slopgate/pkg/diff"
)

// SLP019 flags unreachable code — lines that appear immediately
// after a terminator (return, throw, panic, break, continue) at the
// same or deeper indentation level within the same hunk. AI agents
// frequently generate dead code after terminators.
//
// Exempt: closing braces/parens; blank lines; Go defer lines; lines
// at shallower indentation (new scope); test files; doc files.
type SLP019 struct{}

func (SLP019) ID() string                { return "SLP019" }
func (SLP019) DefaultSeverity() Severity { return SeverityWarn }
func (SLP019) Description() string {
	return "unreachable code after return/throw/panic/break/continue"
}

// slp019Terminators maps lowercased keywords that end execution in a scope.
var slp019Terminators = map[string]bool{
	"return":   true,
	"throw":    true,
	"panic":    true,
	"break":    true,
	"continue": true,
	"raise":    true, // Python
	"sys.exit": true, // Python
}

func slp019IsTerminator(content string) bool {
	trimmed := strings.TrimSpace(stripCommentAndStrings(content))
	if trimmed == "" {
		return false
	}
	// Get first word.
	fields := strings.Fields(trimmed)
	if len(fields) == 0 {
		return false
	}
	word := fields[0]
	// Strip everything from first non-alpha rune onward (e.g. "panic(" → "panic").
	var clean string
	for _, r := range word {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') {
			clean += string(r)
		} else {
			break
		}
	}
	return slp019Terminators[strings.ToLower(clean)]
}

func slp019IsClosingBrace(s string) bool {
	trimmed := strings.TrimSpace(s)
	return trimmed == "}" || trimmed == ")" || trimmed == "];" || trimmed == "]" || trimmed == "});"
}

func slp019IsDefer(s string) bool {
	trimmed := strings.TrimSpace(s)
	return strings.HasPrefix(trimmed, "defer ")
}

func (r SLP019) Check(d *diff.Diff) []Finding {
	var out []Finding
	for _, f := range d.Files {
		if f.IsDelete || isDocFile(f.Path) {
			continue
		}
		isTest := isGoTestFile(f.Path) || isJavaTestFile(f.Path) ||
			isPythonTestFile(f.Path) || isJSTestFile(f.Path) || isRustTestFile(f.Path)
		if isTest {
			continue
		}
		for _, h := range f.Hunks {
			lines := h.Lines
			for i := 0; i < len(lines)-1; i++ {
				ln := lines[i]
				if ln.Kind != diff.LineAdd {
					continue
				}
				if !slp019IsTerminator(ln.Content) {
					continue
				}
				// Look at subsequent lines for unreachable added lines.
				indentTerm := leadingSpaces(ln.Content)
				for j := i + 1; j < len(lines); j++ {
					next := lines[j]
					if next.Kind != diff.LineAdd {
						if next.Kind == diff.LineContext {
							// Context line = scope transition; stop.
							break
						}
						continue
					}
					trimmed := strings.TrimSpace(next.Content)
					if trimmed == "" {
						continue
					}
					if slp019IsClosingBrace(next.Content) {
						continue
					}
					if slp019IsDefer(next.Content) {
						continue
					}
					indentNext := leadingSpaces(next.Content)
					if indentNext < indentTerm {
						// Shallower indentation = new scope; stop.
						break
					}
					if indentNext >= indentTerm {
						out = append(out, Finding{
							RuleID:   r.ID(),
							Severity: r.DefaultSeverity(),
							File:     f.Path,
							Line:     next.NewLineNo,
							Message:  fmt.Sprintf("unreachable code after %s", strings.TrimSpace(stripCommentAndStrings(ln.Content))),
							Snippet:  strings.TrimSpace(next.Content),
						})
						break
					}
				}
			}
		}
	}
	return out
}
