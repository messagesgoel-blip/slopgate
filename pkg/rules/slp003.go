package rules

import (
	"regexp"
	"strings"

	"github.com/messagesgoel-blip/slopgate/pkg/diff"
)

// SLP003 flags catch/except blocks that swallow errors silently —
// empty error handlers where the error is neither logged, wrapped, nor
// re-raised.
//
// Languages: Go, JS/TS, Python.
//
// Rationale: AI agents often write `if err != nil { return nil }` or
// `except: pass` to satisfy the type checker without actually handling
// the error. Real error handlers should at minimum log, wrap, or
// propagate the error.
type SLP003 struct{}

func (SLP003) ID() string                { return "SLP003" }
func (SLP003) DefaultSeverity() Severity { return SeverityWarn }
func (SLP003) Description() string {
	return "error handler swallows the error without acting on it"
}

// --- Go patterns ---

// slp003GoErrCheck matches `if err != nil {` on an added line.
var slp003GoErrCheck = regexp.MustCompile(`^\s*if\s+err\s*!=\s*nil\s*\{\s*$`)

// slp003LogTokens are substrings that indicate the error is being
// logged rather than silently swallowed.
var slp003LogTokens = []string{
	"log.", "slog.", "fmt.", "logger.",
}

// slp003WrapTokens are substrings that indicate the error is being
// wrapped and returned rather than silently swallowed.
var slp003WrapTokens = []string{
	"fmt.Errorf", "errors.Wrap", "errors.Wrapf",
}

// slp003GoHasHandling reports whether a Go error-handler block body
// contains logging, error wrapping, or re-panic.
func slp003GoHasHandling(content string) bool {
	for _, tok := range slp003LogTokens {
		if strings.Contains(content, tok) {
			return true
		}
	}
	for _, tok := range slp003WrapTokens {
		if strings.Contains(content, tok) {
			return true
		}
	}
	if strings.Contains(content, "panic(") {
		return true
	}
	return false
}

// --- JS/TS patterns ---

// slp003JSCatch matches `catch (e) {` or `catch {` on an added line.
// Also matches single-line `catch (e) {}` and `catch (e) { ... }` blocks.
// Allows catch to appear mid-line (e.g., `try { } catch (e) {}`).
var slp003JSCatch = regexp.MustCompile(`(?:^|\s)catch\s*(\(\s*\w+\s*\))?\s*\{`)

// slp003JSLogTokens are substrings that indicate the error is being
// logged in a JS/TS catch block.
var slp003JSLogTokens = []string{
	"logger.", "console.", "log.",
}

// slp003JSBailTokens are lines that are purely structural — they
// don't handle the error, just exit the block.
var slp003JSBailTokens = []string{
	"return;", "return undefined;", "return undefined", "return;",
}

// slp003JSHasHandling reports whether a JS/TS catch block body
// contains logging or error re-throwing.
func slp003JSHasHandling(content string) bool {
	for _, tok := range slp003JSLogTokens {
		if strings.Contains(content, tok) {
			return true
		}
	}
	if strings.Contains(content, "throw") {
		return true
	}
	return false
}

// slp003JSIsBailLine reports whether a trimmed line in a catch block
// is a bare structural exit (return; / return undefined;) that does
// nothing with the error.
func slp003JSIsBailLine(content string) bool {
	trimmed := strings.TrimSpace(content)
	for _, bail := range slp003JSBailTokens {
		if trimmed == bail {
			return true
		}
	}
	return false
}

// --- Python patterns ---

// slp003PythonExcept matches `except:` or `except Exception:` or
// `except Exception as e:` on an added line.
var slp003PythonExcept = regexp.MustCompile(`^\s*except\s*(?:\w+(?:\s+as\s+\w+)?)?\s*:\s*$`)

// slp003PythonBareExcept matches `except:` (no exception type).
var slp003PythonBareExcept = regexp.MustCompile(`^\s*except\s*:\s*$`)

// slp003PythonLogTokens are substrings that indicate the error is
// being logged in a Python except block.
var slp003PythonLogTokens = []string{
	"logging.", "logger.", "log.",
}

// slp003PythonHasHandling reports whether a Python except block body
// contains logging, error re-raising, or error wrapping.
func slp003PythonHasHandling(content string) bool {
	for _, tok := range slp003PythonLogTokens {
		if strings.Contains(content, tok) {
			return true
		}
	}
	if strings.Contains(content, "raise") {
		return true
	}
	return false
}

// slp003PythonIsBailLine reports whether a trimmed line in an except
// block is `pass` or `return` / `return None` — no error handling.
func slp003PythonIsBailLine(content string) bool {
	trimmed := strings.TrimSpace(content)
	return trimmed == "pass" || trimmed == "return" || trimmed == "return None"
}

// --- Check ---

func (r SLP003) Check(d *diff.Diff) []Finding {
	var out []Finding
	for _, f := range d.Files {
		if f.IsDelete {
			continue
		}
		switch {
		case isGoFile(f.Path):
			out = append(out, r.checkGo(f)...)
		case isJSOrTSFile(f.Path):
			out = append(out, r.checkJS(f)...)
		case isPythonFile(f.Path):
			out = append(out, r.checkPython(f)...)
		}
	}
	return out
}

// checkGo scans Go files for `if err != nil { ... }` blocks that
// silently swallow the error.
func (r SLP003) checkGo(f diff.File) []Finding {
	var out []Finding
	for _, h := range f.Hunks {
		lines := h.Lines
		i := 0
		for i < len(lines) {
			ln := lines[i]
			if ln.Kind != diff.LineAdd || !slp003GoErrCheck.MatchString(ln.Content) {
				i++
				continue
			}
			startLine := ln.NewLineNo
			// Track brace depth from the `if` line.
			depth := strings.Count(ln.Content, "{") - strings.Count(ln.Content, "}")
			bodyAllAdded := true
			var bodyContent strings.Builder

			j := i + 1
			for j < len(lines) && depth > 0 {
				bl := lines[j]
				if bl.Kind != diff.LineAdd {
					bodyAllAdded = false
					break
				}
				bodyContent.WriteString(bl.Content)
				bodyContent.WriteByte('\n')
				depth += strings.Count(bl.Content, "{") - strings.Count(bl.Content, "}")
				j++
			}

			if bodyAllAdded && depth == 0 {
				body := bodyContent.String()
				// Only the closing brace means truly empty block.
				trimmed := strings.TrimSpace(body)
				if trimmed == "}" || trimmed == "" {
					out = append(out, Finding{
						RuleID:   r.ID(),
						Severity: r.DefaultSeverity(),
						File:     f.Path,
						Line:     startLine,
						Message:  "if err != nil block is empty — log, wrap, or return the error",
						Snippet:  strings.TrimSpace(ln.Content),
					})
				} else if isGoBailOnly(body) && !slp003GoHasHandling(body) {
					out = append(out, Finding{
						RuleID:   r.ID(),
						Severity: r.DefaultSeverity(),
						File:     f.Path,
						Line:     startLine,
						Message:  "if err != nil block swallows the error — log, wrap, or return the error",
						Snippet:  strings.TrimSpace(ln.Content),
					})
				}
			}

			if j > i {
				i = j
			} else {
				i++
			}
		}
	}
	return out
}

// isGoBailOnly reports whether the body, after stripping the closing
// brace, contains only bare return/return nil statements — no logging,
// no error wrapping.
func isGoBailOnly(body string) bool {
	lines := strings.Split(body, "\n")
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" || trimmed == "}" {
			continue
		}
		if trimmed == "return nil" || trimmed == "return" {
			continue
		}
		// Any other non-trivial statement means the block does something.
		return false
	}
	return true
}

// checkJS scans JS/TS files for `catch (e) { ... }` blocks that
// silently swallow the error.
func (r SLP003) checkJS(f diff.File) []Finding {
	var out []Finding
	for _, h := range f.Hunks {
		lines := h.Lines
		i := 0
		for i < len(lines) {
			ln := lines[i]
			if ln.Kind != diff.LineAdd || !slp003JSCatch.MatchString(ln.Content) {
				i++
				continue
			}
			startLine := ln.NewLineNo

			// Check if this is a single-line catch block: catch (e) {} or catch (e) { return; }
			if strings.Contains(ln.Content, "catch") && strings.Contains(ln.Content, "{") {
				// Extract content between braces for single-line blocks.
				if singleLineBody, ok := extractSingleLineCatchBody(ln.Content); ok {
					if singleLineBody == "" {
						// Empty catch block like catch (e) {}
						out = append(out, Finding{
							RuleID:   r.ID(),
							Severity: r.DefaultSeverity(),
							File:     f.Path,
							Line:     startLine,
							Message:  "catch block is empty — log or re-throw the error",
							Snippet:  strings.TrimSpace(ln.Content),
						})
					} else if jsIsBailOnly(singleLineBody) && !slp003JSHasHandling(singleLineBody) {
						// Bail-only content like catch (e) { return; }
						out = append(out, Finding{
							RuleID:   r.ID(),
							Severity: r.DefaultSeverity(),
							File:     f.Path,
							Line:     startLine,
							Message:  "catch block swallows the error — log or re-throw the error",
							Snippet:  strings.TrimSpace(ln.Content),
						})
					}
					i++
					continue
				}
			}

			depth := strings.Count(ln.Content, "{") - strings.Count(ln.Content, "}")
			bodyAllAdded := true
			var bodyContent strings.Builder

			j := i + 1
			for j < len(lines) && depth > 0 {
				bl := lines[j]
				if bl.Kind != diff.LineAdd {
					bodyAllAdded = false
					break
				}
				bodyContent.WriteString(bl.Content)
				bodyContent.WriteByte('\n')
				depth += strings.Count(bl.Content, "{") - strings.Count(bl.Content, "}")
				j++
			}

			if bodyAllAdded && depth == 0 {
				body := bodyContent.String()
				if slp003JSHasHandling(body) {
					i = j
					continue
				}
				trimmed := strings.TrimSpace(body)
				if trimmed == "}" || trimmed == "" {
					out = append(out, Finding{
						RuleID:   r.ID(),
						Severity: r.DefaultSeverity(),
						File:     f.Path,
						Line:     startLine,
						Message:  "catch block is empty — log or re-throw the error",
						Snippet:  strings.TrimSpace(ln.Content),
					})
				} else if jsIsBailOnly(body) {
					out = append(out, Finding{
						RuleID:   r.ID(),
						Severity: r.DefaultSeverity(),
						File:     f.Path,
						Line:     startLine,
						Message:  "catch block swallows the error — log or re-throw the error",
						Snippet:  strings.TrimSpace(ln.Content),
					})
				}
			}

			if j > i {
				i = j
			} else {
				i++
			}
		}
	}
	return out
}

// extractSingleLineCatchBody extracts the body content from a single-line
// catch block like `catch (e) { return; }`. Returns ("", true) if empty braces,
// ("content", true) if has content, or ("", false) if not a single-line block.
func extractSingleLineCatchBody(content string) (string, bool) {
	// Find "catch" keyword first, then look for its braces.
	catchIdx := strings.Index(content, "catch")
	if catchIdx < 0 {
		return "", false
	}
	// Find the first `{` after "catch".
	start := strings.Index(content[catchIdx:], "{")
	if start < 0 {
		return "", false
	}
	start += catchIdx // adjust to absolute position
	end := strings.LastIndex(content, "}")
	if end < 0 || end < start+1 {
		return "", false
	}
	// Check if there's content between braces.
	body := content[start+1 : end]
	trimmed := strings.TrimSpace(body)
	// If the body contains a newline, it's not a single-line block.
	if strings.Contains(trimmed, "\n") {
		return "", false
	}
	return trimmed, true
}

// jsIsBailOnly reports whether the body, after stripping the closing
// brace, contains only bare return / return undefined statements.
func jsIsBailOnly(body string) bool {
	lines := strings.Split(body, "\n")
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" || trimmed == "}" {
			continue
		}
		if slp003JSIsBailLine(trimmed) {
			continue
		}
		return false
	}
	return true
}

// checkPython scans Python files for `except: ...` blocks that
// silently swallow the error.
func (r SLP003) checkPython(f diff.File) []Finding {
	var out []Finding
	for _, h := range f.Hunks {
		lines := h.Lines
		i := 0
		for i < len(lines) {
			ln := lines[i]
			if ln.Kind != diff.LineAdd || !slp003PythonExcept.MatchString(ln.Content) {
				i++
				continue
			}

			startLine := ln.NewLineNo
			exceptIndent := leadingSpaces(ln.Content)

			// Collect the body: all subsequent added lines that are
			// indented deeper than the except line, or blank lines.
			var bodyContent strings.Builder
			bodyAllAdded := true
			j := i + 1
			for j < len(lines) {
				bl := lines[j]
				if bl.Kind != diff.LineAdd {
					bodyAllAdded = false
					break
				}
				blTrimmed := strings.TrimSpace(bl.Content)
				if blTrimmed == "" {
					j++
					continue
				}
				blIndent := leadingSpaces(bl.Content)
				if blIndent <= exceptIndent {
					// De-dented line ends the block.
					break
				}
				bodyContent.WriteString(bl.Content)
				bodyContent.WriteByte('\n')
				j++
			}

			if bodyAllAdded {
				body := bodyContent.String()
				if slp003PythonHasHandling(body) {
					i = j
					continue
				}
				if pythonIsBailOnly(body) {
					msg := "except block swallows the error — log or re-raise the error"
					if slp003PythonBareExcept.MatchString(ln.Content) {
						msg = "bare except swallows the error — log or re-raise the error"
					}
					out = append(out, Finding{
						RuleID:   r.ID(),
						Severity: r.DefaultSeverity(),
						File:     f.Path,
						Line:     startLine,
						Message:  msg,
						Snippet:  strings.TrimSpace(ln.Content),
					})
				}
			}

			if j > i {
				i = j
			} else {
				i++
			}
		}
	}
	return out
}

// pythonIsBailOnly reports whether the body contains only pass /
// return / return None — no real error handling.
func pythonIsBailOnly(body string) bool {
	lines := strings.Split(body, "\n")
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			continue
		}
		if slp003PythonIsBailLine(trimmed) {
			continue
		}
		return false
	}
	return true
}

// leadingSpaces returns the number of leading spaces in s.
func leadingSpaces(s string) int {
	n := 0
	for _, r := range s {
		if r == ' ' {
			n++
		} else if r == '\t' {
			n += 4 // assume tab = 4 spaces
		} else {
			break
		}
	}
	return n
}
