package rules

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/messagesgoel-blip/slopgate/pkg/diff"
)

// SLP202 flags accesses that may dereference a nil/null pointer.
//
// Primary pattern (high signal): a nil/sentinel check for variable X is
// removed in the diff while X is still dereferenced in newly added code
// at the same or shallower indentation level (i.e. outside the old guard).
//
// Languages: Go, JS/TS, Python, Java, Rust.
//
// Scope: diff only — looks at added/deleted lines within the same file hunk.
type SLP202 struct{}

// ID returns the rule identifier: "SLP202".
func (SLP202) ID() string { return "SLP202" }

// DefaultSeverity returns this rule's default severity.
func (SLP202) DefaultSeverity() Severity { return SeverityBlock }

// Description returns a short description of the SLP202 rule.
func (SLP202) Description() string {
	return "possible nil/null dereference — missing guard before property/method access"
}

// ---------------------------------------------------------------------------
// Regex library
// ---------------------------------------------------------------------------

// requireNonNullPattern extracts the variable name from
// Objects.requireNonNull(varName) calls.
var requireNonNullPattern = regexp.MustCompile(`Objects\.requireNonNull\(([a-zA-Z_]\w*)`)

// nilCheckPatterns match common nil/sentinel guard lines.
var nilCheckPatterns = []*regexp.Regexp{
	// Go
	regexp.MustCompile(`\bif\s+\w+\s*!=\s*nil\b`),
	regexp.MustCompile(`\bif\s+\w+\s*==\s*nil\b`),
	regexp.MustCompile(`\bif\s+nil\s*!=\s*\w+`),
	regexp.MustCompile(`\bif\s+nil\s*==\s*\w+`),
	// JS/TS
	regexp.MustCompile(`\bif\s*\(?\w+\s*[!=]==?\s*(null|undefined)\)?`),
	// Tightened to require property access after && to avoid matching
	// bare boolean checks in Go/Java (e.g. "if user && condition").
	regexp.MustCompile(`\bif\s*\(?\w+\s*&&\s*\w+\.\w+`),
	regexp.MustCompile(`\bif\s*\(?\s*!\s*\w+\s*\)?`),
	regexp.MustCompile(`\bif\s*\(?!(null|undefined)\s*\w+\)?`),
	// Python
	regexp.MustCompile(`\bif\s+\w+\s+is\s+not\s+None`),
	regexp.MustCompile(`\bif\s+\w+\s+is\s+None`),
	regexp.MustCompile(`\bif\s+\w+:`),
	// Java
	regexp.MustCompile(`\bif\s*\(\s*\w+\s*!=\s*null\s*\)`),
	regexp.MustCompile(`\bif\s*\(\s*\w+\s*==\s*null\s*\)`),
	regexp.MustCompile(`Objects\.requireNonNull`),
	// Rust
	regexp.MustCompile(`\bif\s+(\w+)\.is_some\(\)`),
	regexp.MustCompile(`\bif\s+(\w+)\.is_none\(\)`),
	regexp.MustCompile(`\bif\s+let\s+Some\((\w+)\)\s*=\s*(\w+)`),
}

// derefPatterns match lines that access a variable's field, method, or index.
// The first capture group is the variable name being dereferenced.
var derefPatterns = []*regexp.Regexp{
	regexp.MustCompile(`\b(\w+)\.(\w+)\b`),
	regexp.MustCompile(`\b(\w+)\[`),
	regexp.MustCompile(`\b(\w+)\[["']`),
}

// skipLinePatterns are lines we should never flag.
var skipLinePatterns = []*regexp.Regexp{
	regexp.MustCompile(`^\s*//`),   // Go/JS/TS/Java comment
	regexp.MustCompile(`^\s*#`),    // Python comment
	regexp.MustCompile(`^\s*/\*`),  // block comment start
	regexp.MustCompile(`^\s*\*/`),  // block comment end
	regexp.MustCompile(`^\s*\*.+`), // doc comment line
	regexp.MustCompile(`^\s*$`),    // blank/whitespace-only line
}

// ---------------------------------------------------------------------------
// Check
// ---------------------------------------------------------------------------

// Check implements the diff-aware SLP202 rule for nil-deref guard detection.
func (r SLP202) Check(d *diff.Diff) []Finding {
	var out []Finding

	for _, f := range d.Files {
		if f.IsDelete {
			continue
		}

		idx := strings.LastIndex(f.Path, ".")
		if idx < 0 {
			continue
		}
		ext := strings.ToLower(f.Path[idx:])
		if !supportedExt(ext) {
			continue
		}

		if isTestFile(f.Path) {
			continue
		}

		for _, h := range f.Hunks {
			// Pass 1: find variables whose nil guards were REMOVED (deleted lines).
			// Map: variable name -> shallowest guard indentation.
			removedGuards := map[string]int{}
			for _, ln := range h.Lines {
				if ln.Kind != diff.LineDelete {
					continue
				}
				trimmed := strings.TrimSpace(ln.Content)
				// Skip blank lines and comment-only lines so deleted
				// comments like "// if user != nil" aren't treated as
				// removed guard blocks.
				if trimmed == "" || strings.HasPrefix(trimmed, "//") ||
					strings.HasPrefix(trimmed, "#") ||
					strings.HasPrefix(trimmed, "/*") ||
					strings.HasPrefix(trimmed, "*") {
					continue
				}
				for _, pat := range nilCheckPatterns {
					sm := pat.FindStringSubmatch(ln.Content)
					if sm != nil {
						vars := map[string]bool{}
						if len(sm) >= 2 {
							for _, cap := range sm[1:] {
								if cap != "" {
									vars[cap] = true
								}
							}
						}
						// Fall back to token-based extraction for non-capture
						// patterns (Go/JS/TS/Python/Java) or as a safety net.
						for v := range extractGuardVars(ln.Content) {
							vars[v] = true
						}
						indent := slp202LeadingSpaces(ln.Content)
						for v := range vars {
							if cur, ok := removedGuards[v]; !ok || indent < cur {
								removedGuards[v] = indent
							}
						}
						break
					}
				}
			}

			if len(removedGuards) == 0 {
				continue
			}

			// Pass 2: flag added lines that dereference a variable whose
			// guard was removed, and the access is at or above the guard's
			// indentation (meaning it's outside the old guard block).
			for _, ln := range h.Lines {
				if ln.Kind != diff.LineAdd {
					continue
				}
				content := strings.TrimSpace(ln.Content)

				if isSkippableLine(content) {
					continue
				}

				indent := slp202LeadingSpaces(ln.Content)

				found := false
				for _, pat := range derefPatterns {
					submatches := pat.FindAllStringSubmatch(content, -1)
					for _, sm := range submatches {
						if len(sm) >= 2 {
							varName := sm[1]
							if guardIndent, ok := removedGuards[varName]; ok && indent <= guardIndent {
								// Skip if this specific variable is
								// protected by an inline guard.
								if hasInlineNilGuard(varName, content) {
									continue
								}
								out = append(out, Finding{
									RuleID:   r.ID(),
									Severity: r.DefaultSeverity(),
									File:     f.Path,
									Line:     ln.NewLineNo,
									Message:  fmt.Sprintf("possible nil dereference — guard for %q was removed but the variable is still used here", varName),
									Snippet:  content,
								})
								found = true
								break
							}
						}
					}
					if found {
						break
					}
				}
			}
		}
	}

	return out
}

// supportedExt returns true for languages SLP202 covers.
func supportedExt(ext string) bool {
	switch ext {
	case ".go", ".js", ".ts", ".tsx", ".jsx", ".py", ".java", ".kt", ".rs":
		return true
	}
	return false
}

// extractGuardVars returns a map of variable names mentioned in a nil-check
// expression.  It is intentionally conservative — prefers false positives
// over missed detections.
func extractGuardVars(line string) map[string]bool {
	out := map[string]bool{}

	// Handle Objects.requireNonNull(varName) — single-token call that
	// strings.Fields would not split further.
	if m := requireNonNullPattern.FindStringSubmatch(line); len(m) > 1 {
		out[m[1]] = true
	}

	fields := strings.Fields(line)
	for i, tok := range fields {
		tok = strings.Trim(tok, "(,:);[]{}")
		// Strip leading pointer and negation (!, *) so *err and !user
		// both resolve to the variable name.
		tok = strings.TrimLeft(tok, "!*")
		if isLikelyVariable(tok) {
			out[tok] = true
		}
		// Comparison operators bracket the variable name.
		if isComparison(tok) {
			if i > 0 {
				prev := strings.Trim(fields[i-1], "(,:);[]{}")
				prev = strings.TrimLeft(prev, "!*")
				if isLikelyVariable(prev) {
					out[prev] = true
				}
			}
			if i+1 < len(fields) {
				next := strings.Trim(strings.TrimRight(fields[i+1], ",):;[]{}"), "(,:);[]{}")
				next = strings.TrimLeft(next, "!*")
				if isLikelyVariable(next) {
					out[next] = true
				}
			}
		}
	}
	return out
}

// isComparison returns true for ==, !=, ===, !==, is, isnt operators.
func isComparison(tok string) bool {
	switch tok {
	case "!=", "==", "!==", "===", "is", "isnt":
		return true
	}
	return false
}

// isLikelyVariable returns true if s looks like a local variable name
// (not a keyword, type, or literal).
func isLikelyVariable(s string) bool {
	if len(s) > 0 {
		switch strings.ToLower(s) {
		case "if", "for", "return", "nil", "null", "none", "true", "false",
			"this", "self", "defer", "go", "func", "var",
			"let", "const", "with", "as", "of", "in", "and", "or", "not",
			"is", "instanceof", "typeof", "require":
			return false
		}
		c := s[0]
		return (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || c == '_'
	}
	return false
}

// isSkippableLine returns true for lines that should never be flagged.
func isSkippableLine(content string) bool {
	trimmed := strings.TrimLeft(content, " \t")
	for _, pat := range skipLinePatterns {
		if pat.MatchString(trimmed) {
			return true
		}
	}
	for _, safe := range []string{".is_some()", ".is_none()", ".unwrap()", ".expect("} {
		if strings.Contains(content, safe) {
			return true
		}
	}
	return false
}

// inlineNilGuardPatterns are regex patterns that match inline nil/null/undefined
// guards on a single line. These use proper word boundaries instead of substring
// checks to avoid false positives (e.g. "notify" being treated as "not"+"if").
// Patterns with a capture group extract the guarded variable name; patterns
// without one fall back to a proximity check against the matched text.
var inlineNilGuardPatterns = []*regexp.Regexp{
	// Go: if x != nil { / if x == nil {
	regexp.MustCompile(`\bif\s+\w+\s*[!=]==?\s*nil\b`),
	// JS/TS/Java: if (x !== null) / if (x != null) / if (x == null) / if (x === null)
	regexp.MustCompile(`\bif\s*\(?\w+\s*[!=]==?\s*null\s*\)?`),
	// JS/TS: if (typeof x !== 'undefined') / if (x !== undefined)
	regexp.MustCompile(`\bif\s*\(?\w+\s*[!=]==?\s*undefined\s*\)?`),
	// Python: if x is not None
	regexp.MustCompile(`\bif\s+\w+\s+is\s+not\s+None\b`),
	// Python: if x: (bare truthy guard on same line)
	regexp.MustCompile(`\bif\s+(\w+)\s*:`),
}

// hasInlineNilGuard returns true if the line contains a nil guard that
// protects the specific dereferenced variable. Previously this was a
// line-level check that suppressed ALL dereferences if ANY guard existed,
// which caused false negatives when the guard protected a different variable.
func hasInlineNilGuard(varName, content string) bool {
	// Optional chaining is variable-specific: "v?.prop" only guards v.
	if strings.Contains(content, varName+"?.") || strings.Contains(content, varName+"?!") {
		return true
	}
	// Nullish coalescing is variable-specific: "v ?? x" only guards v.
	if strings.Contains(content, varName+"??") || strings.Contains(content, varName+"?:") {
		return true
	}
	// Match standard nil-check patterns and verify the captured group
	// (the guarded variable) is the one that lost its guard.
	for _, pat := range inlineNilGuardPatterns {
		sm := pat.FindStringSubmatch(content)
		if sm != nil {
			// If the pattern has capture groups, check the captured
			// variable name for a match.
			captured := false
			for _, cap := range sm[1:] {
				if cap == varName {
					return true
				}
				captured = true
			}
			// Patterns without capture groups: fall back to a proximity
			// check — if varName appears in the full matched text, it is
			// likely guarded.
			if !captured && strings.Contains(sm[0], varName) {
				return true
			}
		}
	}
	// Python inline truthy check: "if x: x.prop"
	if strings.Contains(content, "is not None") || strings.Contains(content, "is not none") {
		if strings.Contains(content, varName) {
			return true
		}
	}
	return false
}

// slp202LeadingSpaces returns the number of leading space/tab characters.
func slp202LeadingSpaces(s string) int {
	n := 0
	for n < len(s) && (s[n] == ' ' || s[n] == '\t') {
		n++
	}
	return n
}
