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

var slp095SilentReturn = regexp.MustCompile(`(?i)\breturn\s+(?:null|0|false|\[\]|\{\}|""|''|undefined|none)(?:\s*(?:;|\}|//|#)|\s*$)`)
var slp095CatchRE = regexp.MustCompile(`\bcatch\b`)
var slp095ExceptRE = regexp.MustCompile(`\bexcept\b`)

var slp095RaiseRE = regexp.MustCompile(`\braise\b`)

func indentationOf(line string) int {
	width := 0
	for _, r := range line {
		if r == '\t' {
			width += 4
		} else if r == ' ' {
			width++
		} else {
			break
		}
	}
	return width
}

func hasErrorHandling(cLower string) bool {
	return strings.Contains(cLower, "throw ") ||
		slp095RaiseRE.MatchString(cLower) ||
		strings.Contains(cLower, "reject(") ||
		strings.Contains(cLower, "console.error") ||
		strings.Contains(cLower, "console.warn") ||
		strings.Contains(cLower, "log.error") ||
		strings.Contains(cLower, "log.warn") ||
		strings.Contains(cLower, "logger.") ||
		strings.Contains(cLower, "sentry.") ||
		strings.Contains(cLower, "logging.error") ||
		strings.Contains(cLower, "logging.warning") ||
		strings.Contains(cLower, "logging.exception")
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
		isPython := isPythonFile(f.Path)
		isCatchLang := isJSOrTSFile(f.Path) || isJavaFile(f.Path)

		for _, h := range f.Hunks {
			inCatch := false
			catchBraceDepth := 0
			handling := false
			var silentLine *diff.Line
			var exceptIndent int

			for i := range h.Lines {
				ln := &h.Lines[i]
				if ln.Kind == diff.LineDelete {
					continue
				}
				content := strings.TrimSpace(ln.Content)
				cLower := strings.ToLower(content)

				if !inCatch {
					if (isCatchLang && slp095CatchRE.MatchString(cLower)) ||
						(isPython && slp095ExceptRE.MatchString(cLower)) {
						inCatch = true
						catchBraceDepth = 0
						handling = false
						silentLine = nil
						if isPython {
							exceptIndent = indentationOf(ln.Content)
						}
						catchSub := ln.Content
						if isCatchLang {
							if ci := strings.Index(strings.ToLower(ln.Content), "catch"); ci >= 0 {
								catchSub = ln.Content[ci:]
							}
						}
						catchBraceDepth += strings.Count(catchSub, "{")
						catchBraceDepth -= strings.Count(catchSub, "}")
						if hasErrorHandling(cLower) {
							handling = true
						}
						if ln.Kind == diff.LineAdd && slp095SilentReturn.MatchString(content) {
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

				// Count closing braces that appear before the first opening brace (source order)
				// so that transition lines like "} catch (e) {" close the current block first.
				closesBeforeOpen := strings.Count(ln.Content, "}")
				if firstOpen := strings.Index(ln.Content, "{"); firstOpen >= 0 {
					closesBeforeOpen = strings.Count(ln.Content[:firstOpen], "}")
				}
				catchBraceDepth -= closesBeforeOpen

				blockEnded := false
				if isPython {
					if content != "" && indentationOf(ln.Content) <= exceptIndent {
						blockEnded = true
					}
				} else {
					if catchBraceDepth <= 0 && closesBeforeOpen > 0 {
						blockEnded = true
					}
				}

				// Only adjust for braces after the first open if the block hasn't ended.
				if !blockEnded {
					if firstOpen := strings.Index(ln.Content, "{"); firstOpen >= 0 {
						catchBraceDepth += strings.Count(ln.Content[firstOpen:], "{")
						catchBraceDepth -= strings.Count(ln.Content[firstOpen:], "}")
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
					handling = false
					silentLine = nil
					// Re-check if the current line also starts a new (chained) catch.
					if (isCatchLang && slp095CatchRE.MatchString(cLower)) ||
						(isPython && slp095ExceptRE.MatchString(cLower)) {
						inCatch = true
						catchBraceDepth = 0
						catchSub2 := ln.Content
						if isCatchLang {
							if ci2 := strings.Index(strings.ToLower(ln.Content), "catch"); ci2 >= 0 {
								catchSub2 = ln.Content[ci2:]
							}
						}
						catchBraceDepth += strings.Count(catchSub2, "{")
						catchBraceDepth -= strings.Count(catchSub2, "}")
					}
				}

				if hasErrorHandling(cLower) {
					handling = true
				}
				if ln.Kind == diff.LineAdd && slp095SilentReturn.MatchString(content) {
					silentLine = ln
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
