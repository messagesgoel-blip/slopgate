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
//   - Java:  throw new UnsupportedOperationException("not implemented")
//   - Rust:  todo!("..."), unimplemented!("..."), panic!("TODO")
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

// extractStringLiteralFrom extracts the content of a Go-style or JS-style
// interpreted string literal from s, searching for the opening quote at or
// after the start offset. It returns ("", false) if no string literal is found
// after start. Only handles basic "..." and '...' literals — raw strings and
// template literals are out of scope for the stub pattern.
func extractStringLiteralFrom(s string, start int) (string, bool) {
	if start >= len(s) {
		return "", false
	}
	// Look for the first opening quote at or after start.
	i := strings.IndexAny(s[start:], "\"'")
	if i < 0 {
		return "", false
	}
	i += start // absolute position in s
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
var slp006GoPanic = regexp.MustCompile(`\bpanic\s*\(`)

// slp006JSThrow matches JS throw new Error calls: throw new Error(
var slp006JSThrow = regexp.MustCompile(`\bthrow\s+new\s+Error\s*\(`)

// slp006PyRaise matches Python raise NotImplementedError
var slp006PyRaise = regexp.MustCompile(`\braise\s+NotImplementedError\b`)

// slp006JavaThrow matches Java throw new UnsupportedOperationException(
var slp006JavaThrow = regexp.MustCompile(`\bthrow\s+new\s+UnsupportedOperationException\s*\(`)

// slp006RustMacro matches Rust todo!(), unimplemented!(), panic!()
var slp006RustMacro = regexp.MustCompile(`\b(todo|unimplemented|panic)!\s*\(`)

func (r SLP006) Check(d *diff.Diff) []Finding {
	var out []Finding
	for _, f := range d.Files {
		if f.IsDelete {
			continue
		}
		isGo := isGoFile(f.Path)
		isJS := isJSOrTSFile(f.Path)
		isPy := isPythonFile(f.Path)
		isJava := isJavaFile(f.Path)
		isRust := isRustFile(f.Path)
		for _, ln := range f.AddedLines() {
			content := ln.Content

			// Go: panic("...stub keyword...")
			// Run regex on content so byte offsets are consistent with
			// extractStringLiteralFrom.
			if isGo {
				loc := slp006GoPanic.FindStringIndex(content)
				if loc != nil {
					lit, ok := extractStringLiteralFrom(content, loc[0])
					if ok && containsStubKeyword(lit) {
						out = append(out, Finding{
							RuleID:   r.ID(),
							Severity: r.DefaultSeverity(),
							File:     f.Path,
							Line:     ln.NewLineNo,
							Message:  fmt.Sprintf("Go panic with stub keyword %q -- implement or remove", lit),
							Snippet:  strings.TrimSpace(content),
						})
						continue
					}
				}
			}

			// JS/TS: throw new Error("...stub keyword...")
			if isJS {
				loc := slp006JSThrow.FindStringIndex(content)
				if loc != nil {
					lit, ok := extractStringLiteralFrom(content, loc[0])
					if ok && containsStubKeyword(lit) {
						out = append(out, Finding{
							RuleID:   r.ID(),
							Severity: r.DefaultSeverity(),
							File:     f.Path,
							Line:     ln.NewLineNo,
							Message:  fmt.Sprintf("JS throw with stub keyword %q -- implement or remove", lit),
							Snippet:  strings.TrimSpace(content),
						})
						continue
					}
				}
			}

			// Python: raise NotImplementedError
			if isPy {
				loc := slp006PyRaise.FindStringIndex(content)
				if loc != nil {
					msg := "Python raise NotImplementedError -- implement or remove"
					lit, ok := extractStringLiteralFrom(content, loc[1])
					if ok {
						msg = fmt.Sprintf("Python raise NotImplementedError(%q) -- implement or remove", lit)
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

			// Java: throw new UnsupportedOperationException("stub keyword...")
			if isJava {
				loc := slp006JavaThrow.FindStringIndex(content)
				if loc != nil {
					lit, ok := extractStringLiteralFrom(content, loc[0])
					if ok && containsStubKeyword(lit) {
						out = append(out, Finding{
							RuleID:   r.ID(),
							Severity: r.DefaultSeverity(),
							File:     f.Path,
							Line:     ln.NewLineNo,
							Message:  fmt.Sprintf("Java throw UnsupportedOperationException with stub keyword %q -- implement or remove", lit),
							Snippet:  strings.TrimSpace(content),
						})
						continue
					}
					// Bare throw new UnsupportedOperationException() without a stub
					// keyword is also slop -- the exception type itself signals
					// unimplemented code.
					out = append(out, Finding{
						RuleID:   r.ID(),
						Severity: r.DefaultSeverity(),
						File:     f.Path,
						Line:     ln.NewLineNo,
						Message:  "Java throw UnsupportedOperationException -- implement or remove",
						Snippet:  strings.TrimSpace(content),
					})
					continue
				}
			}

			// Rust: todo!("..."), unimplemented!("..."), panic!("stub keyword...")
			if isRust {
				loc := slp006RustMacro.FindStringIndex(content)
				if loc != nil {
					macro := slp006RustMacro.FindStringSubmatch(content)[1]
					// todo!() and unimplemented!() are always stubs.
					if macro == "todo" || macro == "unimplemented" {
						msg := fmt.Sprintf("Rust %s!() -- implement or remove", macro)
						lit, ok := extractStringLiteralFrom(content, loc[1])
						if ok {
							msg = fmt.Sprintf("Rust %s!(%q) -- implement or remove", macro, lit)
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
					// panic!("...stub keyword...") -- only flag with stub keyword.
					if macro == "panic" {
						lit, ok := extractStringLiteralFrom(content, loc[0])
						if ok && containsStubKeyword(lit) {
							out = append(out, Finding{
								RuleID:   r.ID(),
								Severity: r.DefaultSeverity(),
								File:     f.Path,
								Line:     ln.NewLineNo,
								Message:  fmt.Sprintf("Rust panic! with stub keyword %q -- implement or remove", lit),
								Snippet:  strings.TrimSpace(content),
							})
							continue
						}
					}
				}
			}
		}
	}
	return out
}
