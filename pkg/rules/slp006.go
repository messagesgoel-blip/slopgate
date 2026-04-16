package rules

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/messagesgoel-blip/slopgate/pkg/diff"
)

// SLP006 flags panic/throw/raise stub bodies that signal unimplemented
// code. These are common when an AI agent generates a skeleton and
// leaves the real logic unwritten — the panic or throw is a sentinel
// that will crash at runtime if the code path is ever hit.
//
// Detected patterns (on ADDED lines only):
//   - Go:    panic("not implemented"), panic("TODO"), panic(fmt.Sprintf("TODO: ..."))
//   - JS/TS: throw new Error("not implemented"), throw new Error("TODO")
//   - Python: raise NotImplementedError, raise NotImplementedError("msg")
//
// Non-stub panics like panic(err) or panic("buffer too small") are
// deliberately excluded — they don't contain a stub keyword.
type SLP006 struct{}

func (SLP006) ID() string                { return "SLP006" }
func (SLP006) DefaultSeverity() Severity { return SeverityBlock }
func (SLP006) Description() string {
	return "stub body signals unimplemented code"
}

// stubKeywords are the case-insensitive words that mark a panic/throw
// as a stub rather than a legitimate runtime error.
var stubKeywords = []string{
	"not implemented",
	"todo",
	"fixme",
	"unimplemented",
}

// containsStubKeyword reports whether s (case-insensitive) contains any
// of the stub keywords as a substring.
func containsStubKeyword(s string) bool {
	lower := strings.ToLower(s)
	for _, kw := range stubKeywords {
		if strings.Contains(lower, kw) {
			return true
		}
	}
	return false
}

// extractStringLiteral extracts the content of a Go-style or JS-style
// interpreted string literal from s. It returns ("", false) if no
// string literal is found. Only handles basic "..." and '...' literals
// — raw strings and template literals are out of scope for the stub
// pattern.
func extractStringLiteral(s string) (string, bool) {
	// Look for the first opening quote after the panic/throw call.
	i := strings.IndexAny(s, "\"'")
	if i < 0 {
		return "", false
	}
	quote := s[i]
	// Find the closing quote. Simple scan — does not handle escaped
	// quotes, which are vanishingly rare in stub messages.
	j := strings.IndexByte(s[i+1:], quote)
	if j < 0 {
		return "", false
	}
	return s[i+1 : i+1+j], true
}

// slp006GoPanic matches Go panic calls: panic(
var slp006GoPanic = regexp.MustCompile(`panic\s*\(`)

// slp006JSThrow matches JS throw new Error calls: throw new Error(
var slp006JSThrow = regexp.MustCompile(`throw\s+new\s+Error\s*\(`)

// slp006PyRaise matches Python raise NotImplementedError
var slp006PyRaise = regexp.MustCompile(`raise\s+NotImplementedError`)

func (r SLP006) Check(d *diff.Diff) []Finding {
	var out []Finding
	for _, f := range d.Files {
		if f.IsDelete {
			continue
		}
		isGo := isGoFile(f.Path)
		isJS := isJSOrTSFile(f.Path)
		isPy := isPythonFile(f.Path)
		for _, ln := range f.AddedLines() {
			content := ln.Content
			stripped := stripCommentAndStrings(content)

			// Go: panic("...stub keyword...")
			if isGo && slp006GoPanic.MatchString(stripped) {
				lit, ok := extractStringLiteral(content)
				if ok && containsStubKeyword(lit) {
					out = append(out, Finding{
						RuleID:   r.ID(),
						Severity: r.DefaultSeverity(),
						File:     f.Path,
						Line:     ln.NewLineNo,
						Message:  fmt.Sprintf("Go panic with stub keyword %q — implement or remove", lit),
						Snippet:  strings.TrimSpace(content),
					})
					continue
				}
			}

			// JS/TS: throw new Error("...stub keyword...")
			if isJS && slp006JSThrow.MatchString(stripped) {
				lit, ok := extractStringLiteral(content)
				if ok && containsStubKeyword(lit) {
					out = append(out, Finding{
						RuleID:   r.ID(),
						Severity: r.DefaultSeverity(),
						File:     f.Path,
						Line:     ln.NewLineNo,
						Message:  fmt.Sprintf("JS throw with stub keyword %q — implement or remove", lit),
						Snippet:  strings.TrimSpace(content),
					})
					continue
				}
			}

			// Python: raise NotImplementedError
			if isPy && slp006PyRaise.MatchString(content) {
				msg := "Python raise NotImplementedError — implement or remove"
				lit, ok := extractStringLiteral(content)
				if ok {
					msg = fmt.Sprintf("Python raise NotImplementedError(%q) — implement or remove", lit)
				}
				out = append(out, Finding{
					RuleID:   r.ID(),
					Severity: r.DefaultSeverity(),
					File:     f.Path,
					Line:     ln.NewLineNo,
					Message:  msg,
					Snippet:  strings.TrimSpace(content),
				})
				continue
			}
		}
	}
	return out
}
