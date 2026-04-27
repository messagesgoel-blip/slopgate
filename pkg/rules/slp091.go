package rules

import (
	"regexp"
	"strings"

	"github.com/messagesgoel-blip/slopgate/pkg/diff"
)

// SLP091 flags hardcoded date/time literals in test fixtures that will
// predictably expire and break CI in the future.
//
// Rationale: AI agents frequently generate fixtures with literal dates
// (new Date("2025-01-01"), expires_at: 2026-06-01). These become time-
// bombed tests that fail months later with opaque errors.
type SLP091 struct{}

func (SLP091) ID() string                { return "SLP091" }
func (SLP091) DefaultSeverity() Severity { return SeverityBlock }
func (SLP091) Description() string {
	return "hardcoded date in test fixture — will expire and break CI"
}

var slp091ISODate = regexp.MustCompile(`\b(20[1-3]\d)[-/](0[1-9]|1[0-2])[-/](0[1-9]|[12]\d|3[01])\b`)

var slp091JSDate = regexp.MustCompile(`new\s+Date\s*\(\s*["'\x60]`)

var slp091SQLDate = regexp.MustCompile(`(?i)(expires?_?at|valid_until|not_after|expiry_date)\s*[:=]\s*\d{4}[-/]\d{1,2}[-/]\d{1,2}`)

var slp091Timestamp = regexp.MustCompile(`(?i)"(?:expires?_?(?:at|in)|ttl|deadline)"\s*[:=]\s*\d{10,13}\b|(?i)(expires?_?(?:at|in)|ttl|deadline)\s*[:=]\s*\d{10,13}\b`)

var testFileSuffixes = []string{
	"_test.go", ".test.go", "_test.py", ".test.js", ".test.ts", ".test.tsx", ".test.jsx",
	".spec.js", ".spec.ts", ".spec.tsx", ".spec.jsx",
	"_test.java", ".test.java", "_tests.java", ".tests.java", "_test.rs",
}

func isTestFile(path string) bool {
	lower := strings.ToLower(path)
	for _, s := range testFileSuffixes {
		if strings.HasSuffix(lower, s) {
			return true
		}
	}

	for _, segment := range strings.Split(lower, "/") {
		switch segment {
		case "test", "tests", "testdata", "fixtures", "__fixtures__", "fixture":
			return true
		}
	}
	return false
}

// indexOutsideQuotes returns the index of substr in s, but only if substr
// is found outside of any quoted string (single, double, or backtick).
// Returns -1 if not found or if substr is inside quotes.
func indexOutsideQuotes(s, substr string) int {
	inSingle := false
	inDouble := false
	inBacktick := false
	escaped := false
	for i := 0; i < len(s); i++ {
		ch := s[i]
		if escaped {
			escaped = false
			continue
		}
		if (inSingle || inDouble) && ch == '\\' {
			escaped = true
			continue
		}
		switch {
		case !inDouble && !inBacktick && ch == '\'':
			inSingle = !inSingle
		case !inSingle && !inBacktick && ch == '"':
			inDouble = !inDouble
		case !inSingle && !inDouble && ch == '`':
			inBacktick = !inBacktick
		}
		if inSingle || inDouble || inBacktick {
			continue
		}
		if i+len(substr) <= len(s) && s[i:i+len(substr)] == substr {
			return i
		}
	}
	return -1
}

// stripInlineCommentOutsideQuotes removes an inline comment suffix from s
// while respecting single-quoted, double-quoted, and backtick-quoted strings.
// "//" is always treated as a comment start outside quotes; " #" and " --"
// require a preceding space so that "http://" and SQL dates are not truncated.
func stripInlineCommentOutsideQuotes(s string) string {
	inSingle := false
	inDouble := false
	inBacktick := false
	escaped := false
	for i := 0; i < len(s); i++ {
		ch := s[i]
		if escaped {
			escaped = false
			continue
		}
		if (inSingle || inDouble) && ch == '\\' {
			escaped = true
			continue
		}
		switch {
		case !inDouble && !inBacktick && ch == '\'':
			inSingle = !inSingle
		case !inSingle && !inBacktick && ch == '"':
			inDouble = !inDouble
		case !inSingle && !inDouble && ch == '`':
			inBacktick = !inBacktick
		}
		if inSingle || inDouble || inBacktick {
			continue
		}
		// "//" — always a line comment when outside quotes.
		if ch == '/' && i+1 < len(s) && s[i+1] == '/' {
			return strings.TrimSpace(s[:i])
		}
		// " #" — hash preceded by whitespace (shell/Python inline comment).
		if ch == '#' && i > 0 && (s[i-1] == ' ' || s[i-1] == '\t') {
			return strings.TrimSpace(s[:i-1])
		}
		// " --" — double-dash preceded by whitespace (SQL inline comment).
		if ch == '-' && i > 0 && (s[i-1] == ' ' || s[i-1] == '\t') && i+1 < len(s) && s[i+1] == '-' {
			return strings.TrimSpace(s[:i-1])
		}
	}
	return strings.TrimSpace(s)
}

func (r SLP091) Check(d *diff.Diff) []Finding {
	var out []Finding
	for _, f := range d.Files {
		if f.IsDelete {
			continue
		}
		if !isTestFile(f.Path) {
			continue
		}

		// Do not skip doc files for SLP091 — test fixtures like testdata/*.txt
		// are legitimate locations for hardcoded dates.
		inBlockComment := false
		for _, ln := range f.AddedLines() {
			content := ln.Content
			trimmed := strings.TrimSpace(content)
			// If we're inside a block comment, skip until */ is found.
			if inBlockComment {
				if closeIdx := indexOutsideQuotes(trimmed, "*/"); closeIdx >= 0 {
					trimmed = strings.TrimSpace(trimmed[closeIdx+2:])
					content = trimmed
					inBlockComment = false
					if trimmed == "" {
						continue
					}
					// Fall through to strip any further /* */ spans on this line.
				} else {
					continue
				}
			}
			// Repeatedly strip all /* */ comment spans from the line.
			for {
				openIdx := indexOutsideQuotes(trimmed, "/*")
				if openIdx < 0 {
					break
				}
				closeOff := indexOutsideQuotes(trimmed[openIdx+2:], "*/")
				if closeOff < 0 {
					// Unclosed /* — truncate at the comment open and set inBlockComment.
					trimmed = strings.TrimSpace(trimmed[:openIdx])
					content = trimmed
					inBlockComment = true
					break
				}
				absClose := openIdx + 2 + closeOff + 2
				trimmed = strings.TrimSpace(trimmed[:openIdx] + trimmed[absClose:])
				content = trimmed
			}
			if trimmed == "" {
				continue
			}
			if strings.HasPrefix(trimmed, "//") ||
				strings.HasPrefix(trimmed, "#") ||
				strings.HasPrefix(trimmed, "--") {
				continue
			}

			// Strip inline comment suffixes (quote-aware) before regex matching.
			contentForMatch := stripInlineCommentOutsideQuotes(content)
			if contentForMatch == "" {
				continue
			}
			var msg string
			switch {
			case slp091JSDate.MatchString(contentForMatch):
				msg = "hardcoded JS Date with string literal in test — use relative date instead"
			case slp091SQLDate.MatchString(contentForMatch):
				msg = "hardcoded expiry date in test fixture — will expire and break CI"
			case slp091Timestamp.MatchString(contentForMatch):
				msg = "hardcoded timestamp in test — use relative time or mock"
			case slp091ISODate.MatchString(contentForMatch):
				for _, match := range slp091ISODate.FindAllStringSubmatch(contentForMatch, -1) {
					if len(match) > 1 {
						year := match[1]
						if strings.HasPrefix(year, "202") || strings.HasPrefix(year, "203") {
							msg = "hardcoded date literal in test — consider using a relative date expression"
							break
						}
					}
				}
			}
			if msg == "" {
				continue
			}
			out = append(out, Finding{
				RuleID:   r.ID(),
				Severity: r.DefaultSeverity(),
				File:     f.Path,
				Line:     ln.NewLineNo,
				Message:  msg,
				Snippet:  ln.Content,
			})
		}
	}
	return out
}
