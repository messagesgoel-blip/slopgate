package rules

import (
	"regexp"
	"strings"

	"github.com/messagesgoel-blip/slopgate/pkg/diff"
)

// SLP095 flags try/catch/except blocks where the catch handler returns
// a sentinel value (null, 0, false, empty collection) without re-throwing
// or logging. This is the "silent failure" pattern.
type SLP095 struct{}

func (SLP095) ID() string                { return "SLP095" }
func (SLP095) DefaultSeverity() Severity { return SeverityBlock }
func (SLP095) Description() string {
	return "catch block returns silently without handling the error — use throw/reject or log then rethrow"
}

var slp095SilentReturn = regexp.MustCompile(`(?i)\breturn\s+(?:null|0|false|\[\]|\{\}|""|''|undefined|none)(?:\s|;|$|})`)
var slp095CatchRE = regexp.MustCompile(`\bcatch\b`)
var slp095ExceptRE = regexp.MustCompile(`\bexcept\b`)

func indentationOf(line string) string {
	for i, r := range line {
		if r != ' ' && r != '\t' {
			return line[:i]
		}
	}
	return line
}

func hasErrorHandling(cLower string) bool {
	return strings.Contains(cLower, "throw ") ||
		strings.Contains(cLower, "reject(") ||
		strings.Contains(cLower, "console.error") ||
		strings.Contains(cLower, "console.warn") ||
		strings.Contains(cLower, "log.error") ||
		strings.Contains(cLower, "log.warn") ||
		strings.Contains(cLower, "logger.") ||
		strings.Contains(cLower, "sentry.")
}

func (r SLP095) Check(d *diff.Diff) []Finding {
	var out []Finding
	for _, f := range d.Files {
		if f.IsDelete || isDocFile(f.Path) {
			continue
		}
		if !isJSOrTSFile(f.Path) && !isPythonFile(f.Path) && !isJavaFile(f.Path) {
			continue
		}
		if isTestFile(f.Path) {
			continue
		}

		for _, h := range f.Hunks {
			inCatch := false
			catchBraceDepth := 0
			handling := false
			var silentLine *diff.Line
			var exceptIndent string

			for i := range h.Lines {
				ln := &h.Lines[i]
				if ln.Kind != diff.LineAdd {
					continue
				}
				content := strings.TrimSpace(ln.Content)
				cLower := strings.ToLower(content)

				if !inCatch {
					if slp095CatchRE.MatchString(cLower) || slp095ExceptRE.MatchString(cLower) {
						inCatch = true
						catchBraceDepth = 0
						handling = false
						silentLine = nil
						if isPythonFile(f.Path) {
							exceptIndent = indentationOf(ln.Content)
						}
						catchBraceDepth += strings.Count(ln.Content, "{")
						catchBraceDepth -= strings.Count(ln.Content, "}")
						if hasErrorHandling(cLower) {
							handling = true
						}
						if slp095SilentReturn.MatchString(content) {
							silentLine = ln
						}
						if catchBraceDepth <= 0 && strings.Contains(ln.Content, "}") {
							if !handling && silentLine != nil {
								out = append(out, Finding{
									RuleID:   r.ID(),
									Severity: r.DefaultSeverity(),
									File:     f.Path,
									Line:     silentLine.NewLineNo,
									Message:  "catch/except block returns silently — rethrow or handle with explicit logging",
									Snippet:  strings.TrimSpace(silentLine.Content),
								})
							}
							inCatch = false
						}
					}
					continue
				}

				catchBraceDepth += strings.Count(ln.Content, "{")
				catchBraceDepth -= strings.Count(ln.Content, "}")

				if hasErrorHandling(cLower) {
					handling = true
				}
				if slp095SilentReturn.MatchString(content) {
					silentLine = ln
				}

				blockEnded := false
				if isPythonFile(f.Path) {
					if content != "" && indentationOf(ln.Content) <= exceptIndent {
						blockEnded = true
					}
				} else {
					if catchBraceDepth <= 0 && strings.Contains(ln.Content, "}") {
						blockEnded = true
					}
				}

				if blockEnded {
					if !handling && silentLine != nil {
						out = append(out, Finding{
							RuleID:   r.ID(),
							Severity: r.DefaultSeverity(),
							File:     f.Path,
							Line:     silentLine.NewLineNo,
							Message:  "catch/except block returns silently — rethrow or handle with explicit logging",
							Snippet:  strings.TrimSpace(silentLine.Content),
						})
					}
					inCatch = false
				}
			}

			// Handle single-line catch/except (last line in hunk).
			if inCatch && !handling && silentLine != nil {
				out = append(out, Finding{
					RuleID:   r.ID(),
					Severity: r.DefaultSeverity(),
					File:     f.Path,
					Line:     silentLine.NewLineNo,
					Message:  "catch/except block returns silently — rethrow or handle with explicit logging",
					Snippet:  strings.TrimSpace(silentLine.Content),
				})
			}
		}
	}
	return out
}
