package rules

import (
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

func slp095SilentReturn(cLower string) bool {
	return strings.Contains(cLower, "return null") ||
		strings.Contains(cLower, "return 0") ||
		strings.Contains(cLower, "return false") ||
		strings.Contains(cLower, "return []") ||
		strings.Contains(cLower, "return {}") ||
		strings.Contains(cLower, "return \"\"") ||
		strings.Contains(cLower, "return ''") ||
		strings.Contains(cLower, "return undefined") ||
		strings.Contains(cLower, "return none")
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
		if strings.Contains(strings.ToLower(f.Path), ".test.") ||
			strings.Contains(strings.ToLower(f.Path), ".spec.") {
			continue
		}
		if isJavaTestFile(f.Path) {
			continue
		}

		for _, h := range f.Hunks {
			inCatch := false
			catchBraceDepth := 0
			handling := false
			var silentLine *diff.Line

			for i := range h.Lines {
				ln := &h.Lines[i]
				if ln.Kind != diff.LineAdd {
					continue
				}
				content := strings.TrimSpace(ln.Content)
				cLower := strings.ToLower(content)

				if !inCatch {
					if strings.Contains(cLower, "catch") || strings.Contains(cLower, "except") {
						inCatch = true
						catchBraceDepth = 0
						handling = false
						silentLine = nil
						catchBraceDepth += strings.Count(ln.Content, "{")
						catchBraceDepth -= strings.Count(ln.Content, "}")
						if hasErrorHandling(cLower) {
							handling = true
						}
						if slp095SilentReturn(cLower) {
							silentLine = ln
						}
					}
					continue
				}

				catchBraceDepth += strings.Count(ln.Content, "{")
				catchBraceDepth -= strings.Count(ln.Content, "}")

				if hasErrorHandling(cLower) {
					handling = true
				}
				if slp095SilentReturn(cLower) {
					silentLine = ln
				}

				blockEnded := false
				if isPythonFile(f.Path) {
					if !strings.HasPrefix(ln.Content, " ") && !strings.HasPrefix(ln.Content, "\t") && content != "" {
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