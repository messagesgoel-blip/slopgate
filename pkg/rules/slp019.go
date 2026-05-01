package rules

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/messagesgoel-blip/slopgate/pkg/diff"
)

// SLP019 flags unreachable code — lines that appear immediately
// after a terminator (return, throw, panic, break, continue) at the
// same or deeper indentation level within the same hunk. AI agents
// frequently generate dead code after terminators.
//
// Exempt: closing braces/parens; blank lines; lines at shallower
// indentation (new scope); test files; doc files.
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

var slp019ContinuationSuffix = regexp.MustCompile(`(?:[([{.,:+\-*/%&|^!?]|\b(?:and|or)\b)\s*$`)

func slp019IsTerminator(content string) bool {
	raw := strings.TrimSpace(content)
	trimmed := strings.TrimSpace(stripCommentAndStrings(content))
	if trimmed == "" {
		return false
	}
	fields := strings.Fields(trimmed)
	if len(fields) == 0 {
		return false
	}
	word := fields[0]
	// Check compound terminators first (e.g. "sys.exit(1)" → "sys.exit").
	for key := range slp019Terminators {
		if strings.Contains(key, ".") && strings.HasPrefix(word, key) {
			// Ensure the match isn't a longer identifier (e.g. "sys.exit_code").
			rest := word[len(key):]
			if len(rest) == 0 || rest[0] == '(' || (!isAlphaNum(rest[0]) && rest[0] != '_') {
				return true
			}
		}
	}
	// Strip everything from first non-alpha rune onward (e.g. "panic(" → "panic").
	var b strings.Builder
	for _, r := range word {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || r == '.' {
			b.WriteRune(r)
		} else {
			break
		}
	}
	keyword := strings.ToLower(b.String())
	if !slp019Terminators[keyword] {
		return false
	}
	return !slp019StatementContinues(raw, trimmed, keyword)
}

func slp019StatementContinues(raw, cleaned, keyword string) bool {
	remainder := strings.TrimSpace(strings.TrimPrefix(cleaned, keyword))
	if remainder == "" {
		return false
	}

	var parens, brackets, braces int
	for _, r := range remainder {
		switch r {
		case '(':
			parens++
		case ')':
			if parens > 0 {
				parens--
			}
		case '[':
			brackets++
		case ']':
			if brackets > 0 {
				brackets--
			}
		case '{':
			braces++
		case '}':
			if braces > 0 {
				braces--
			}
		}
	}
	if parens > 0 || brackets > 0 || braces > 0 {
		return true
	}

	trimmedRaw := strings.TrimSpace(raw)
	if slp019ContinuationSuffix.MatchString(trimmedRaw) {
		return true
	}
	return strings.HasSuffix(trimmedRaw, "=>")
}

func isAlphaNum(b byte) bool {
	return (b >= 'a' && b <= 'z') || (b >= 'A' && b <= 'Z') || (b >= '0' && b <= '9')
}

func slp019IsClosingBrace(s string) bool {
	// Strip inline comments and trailing semicolons, then trim.
	clean := stripCommentAndStrings(s)
	clean = strings.TrimRight(clean, ";")
	clean = strings.TrimSpace(clean)
	return clean == "}" || clean == ")" || clean == "]" || clean == "})" || clean == "])"
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
							break
						}
						continue
					}
					trimmed := strings.TrimSpace(next.Content)
					if trimmed == "" {
						continue
					}
					// Skip comment-only lines.
					if strings.TrimSpace(stripCommentAndStrings(next.Content)) == "" {
						continue
					}
					if slp019IsClosingBrace(next.Content) {
						continue
					}
					indentNext := leadingSpaces(next.Content)
					if indentNext < indentTerm {
						break
					}
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
	return out
}
