package rules

import (
	"regexp"
	"strings"

	"github.com/messagesgoel-blip/slopgate/pkg/diff"
)

// SLP056 flags hardcoded secrets in added lines across any file type.
type SLP056 struct{}

func (SLP056) ID() string                { return "SLP056" }
func (SLP056) DefaultSeverity() Severity { return SeverityBlock }
func (SLP056) Description() string {
	return "hardcoded secrets detected in added lines"
}

var secretPatterns = []*regexp.Regexp{
	regexp.MustCompile(`(?i)api[_-]?key\s*[:=]\s*["']\w+`),
	regexp.MustCompile(`(?i)password\s*[:=]\s*["'][^"']+`),
	regexp.MustCompile(`(?i)secret\s*[:=]\s*["']\w+`),
	regexp.MustCompile(`(?i)token\s*[:=]\s*["']\w+`),
	regexp.MustCompile(`(?i)bearer\s+\w+`),
	regexp.MustCompile(`(?i)aws_access_key_id\s*[:=]\s*\w+`),
	regexp.MustCompile(`(?i)private_key\s*[:=]`),
}

var skipWords = []string{"example", "sample", "dummy", "test", "placeholder", "fake", "mock", "todo", "fixme"}

func (r SLP056) Check(d *diff.Diff) []Finding {
	var out []Finding
	for _, f := range d.Files {
		if f.IsDelete {
			continue
		}
		for _, ln := range f.AddedLines() {
			lower := strings.ToLower(ln.Content)
			skip := false
			for _, w := range skipWords {
				if strings.Contains(lower, w) {
					skip = true
					break
				}
			}
			if skip {
				continue
			}
			for _, re := range secretPatterns {
				if re.MatchString(ln.Content) {
					out = append(out, Finding{
						RuleID:   r.ID(),
						Severity: r.DefaultSeverity(),
						File:     f.Path,
						Line:     ln.NewLineNo,
						Message:  "hardcoded secret pattern detected — use environment variables or a secret manager",
						Snippet:  "[REDACTED]",
					})
					break
				}
			}
		}
	}
	return out
}
