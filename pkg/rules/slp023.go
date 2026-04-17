package rules

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/messagesgoel-blip/slopgate/pkg/diff"
)

// SLP023 flags Go bare type assertions without the comma-ok guard.
// AI agents frequently write
//
//	s := v.(string)
//
// which panics if the assertion fails, instead of the safe form:
//
//	s, ok := v.(string)
//
// Exempt: comma-ok assignments; type switches (v.(type)); test files.
type SLP023 struct{}

func (SLP023) ID() string                { return "SLP023" }
func (SLP023) DefaultSeverity() Severity { return SeverityWarn }
func (SLP023) Description() string {
	return "bare type assertion without comma-ok guard panics on mismatch"
}

// slp023TypeAssert matches a type assertion pattern: expr.(TypeName).
// Supports qualified types like pkg.Type and *pkg.Type.
var slp023TypeAssert = regexp.MustCompile(`\w[\w.*]*\s*\.\s*\(\s*\*?\s*\w+(?:\.\w+)*\s*\)`)

// slp023CommaOk matches the comma-ok guard: `, ok :=` or `, ok =` (with or without space).
var slp023CommaOk = regexp.MustCompile(`,\s*ok\s+:?=`)

// slp023TypeSwitch matches type switch constructs: `.(\s*type\s*)`.
var slp023TypeSwitch = regexp.MustCompile(`\.\s*\(\s*type\s*\)`)

func (r SLP023) Check(d *diff.Diff) []Finding {
	var out []Finding
	for _, f := range d.Files {
		if f.IsDelete || !isGoFile(f.Path) || isGoTestFile(f.Path) {
			continue
		}
		for _, ln := range f.AddedLines() {
			content := ln.Content
			trimmed := strings.TrimLeft(content, " \t")
			if strings.HasPrefix(trimmed, "//") {
				continue
			}
			// Skip type switches.
			if slp023TypeSwitch.MatchString(content) {
				continue
			}
			// Skip comma-ok assertions.
			if slp023CommaOk.MatchString(content) {
				continue
			}
			// Check for bare type assertions.
			if !slp023TypeAssert.MatchString(content) {
				continue
			}
			// Extract the assertion for the message.
			m := slp023TypeAssert.FindString(content)
			out = append(out, Finding{
				RuleID:   r.ID(),
				Severity: r.DefaultSeverity(),
				File:     f.Path,
				Line:     ln.NewLineNo,
				Message:  fmt.Sprintf("bare type assertion %s — add comma-ok guard to prevent panic on type mismatch", m),
				Snippet:  strings.TrimSpace(content),
			})
		}
	}
	return out
}
