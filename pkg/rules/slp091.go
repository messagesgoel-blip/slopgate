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

var slp091SQLDate = regexp.MustCompile(`(?i)(expires?_?at|valid_until|not_after|expiry_date)\s*[:=]\s*\d`)

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
		case "test", "tests", "testdata":
			return true
		}
	}
	return false
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

		if isDocFile(f.Path) {
			continue
		}
		inBlockComment := false
		for _, ln := range f.AddedLines() {
			content := ln.Content
			trimmed := strings.TrimSpace(content)
			if strings.HasPrefix(trimmed, "/*") {
				inBlockComment = true
			}
			if inBlockComment {
				if strings.Contains(trimmed, "*/") {
					inBlockComment = false
				}
				continue
			}
			if strings.HasPrefix(trimmed, "//") ||
				strings.HasPrefix(trimmed, "#") ||
				strings.HasPrefix(trimmed, "--") {
				continue
			}

			var msg string
			switch {
			case slp091JSDate.MatchString(content):
				msg = "hardcoded JS Date with string literal in test — use relative date instead"
			case slp091SQLDate.MatchString(content):
				msg = "hardcoded expiry date in test fixture — will expire and break CI"
			case slp091Timestamp.MatchString(content):
				msg = "hardcoded timestamp in test — use relative time or mock"
			case slp091ISODate.MatchString(content):
				if match := slp091ISODate.FindStringSubmatch(content); len(match) > 1 {
					year := match[1]
					if strings.HasPrefix(year, "202") || strings.HasPrefix(year, "203") {
						msg = "hardcoded date literal in test — consider using a relative date expression"
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
				Snippet:  strings.TrimSpace(ln.Content),
			})
		}
	}
	return out
}
