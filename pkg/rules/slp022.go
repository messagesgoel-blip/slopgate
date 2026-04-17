package rules

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/messagesgoel-blip/slopgate/pkg/diff"
)

// SLP022 flags Go error wrapping that uses %v or %s instead of %w
// in fmt.Errorf calls. AI agents frequently write
//
//	fmt.Errorf("something failed: %v", err)
//
// instead of the correct
//
//	fmt.Errorf("something failed: %w", err)
//
// The %v form compiles and returns an error, but breaks error chain
// unwrapping with errors.Is/errors.As.
//
// Exempt: lines already using %w; errors.Wrap/errors.Wrapf; test files.
type SLP022 struct{}

func (SLP022) ID() string                { return "SLP022" }
func (SLP022) DefaultSeverity() Severity { return SeverityWarn }
func (SLP022) Description() string {
	return "fmt.Errorf uses %v/%s with error arg instead of %w for wrapping"
}

// slp022Errorf matches fmt.Errorf( calls on added lines.
var slp022Errorf = regexp.MustCompile(`fmt\.Errorf\s*\(`)

// slp022FormatStr extracts the double-quoted format string from a
// fmt.Errorf call. Group 1 contains the format string content.
var slp022FormatStr = regexp.MustCompile(`fmt\.Errorf\s*\(\s*"([^"]*)"`)

// slp022ErrArg matches an error variable in the args list (after the
// format string's closing quote), e.g. ", err" or ", myErr" or ", returnErr".
var slp022ErrArg = regexp.MustCompile(`"[^"]*"\s*,[^)]*[Ee]rr\b`)

func (r SLP022) Check(d *diff.Diff) []Finding {
	var out []Finding
	for _, f := range d.Files {
		if f.IsDelete || !isGoFile(f.Path) || isGoTestFile(f.Path) {
			continue
		}
		for _, ln := range f.AddedLines() {
			content := ln.Content
			// Skip comment-only lines.
			trimmed := strings.TrimLeft(content, " \t")
			if strings.HasPrefix(trimmed, "//") {
				continue
			}
			if !slp022Errorf.MatchString(content) {
				continue
			}
			m := slp022FormatStr.FindStringSubmatch(content)
			if m == nil {
				// Multi-line or backtick format string — skip.
				continue
			}
			formatStr := m[1]
			if strings.Contains(formatStr, "%w") {
				continue
			}
			if !strings.Contains(formatStr, "%v") && !strings.Contains(formatStr, "%s") {
				continue
			}
			// Check that an error variable appears in the args.
			if !slp022ErrArg.MatchString(content) {
				continue
			}
			out = append(out, Finding{
				RuleID:   r.ID(),
				Severity: r.DefaultSeverity(),
				File:     f.Path,
				Line:     ln.NewLineNo,
				Message:  fmt.Sprintf("fmt.Errorf uses %%v/%%s with error arg — use %%w to preserve error chain (%q)", formatStr),
				Snippet:  strings.TrimSpace(content),
			})
		}
	}
	return out
}
