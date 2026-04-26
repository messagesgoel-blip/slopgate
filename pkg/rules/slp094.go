package rules

import (
	"regexp"
	"strings"

	"github.com/messagesgoel-blip/slopgate/pkg/diff"
)

// SLP094 flags shell commands that suppress failures with || true or || :
// This is a common anti-pattern where AI agents silence errors instead of
// handling them, leading to builds that appear green but are actually broken.
type SLP094 struct{}

func (SLP094) ID() string                { return "SLP094" }
func (SLP094) DefaultSeverity() Severity { return SeverityBlock }
func (SLP094) Description() string {
	return "shell command suppresses failure with || true — handle the error instead"
}

var slp094SilentFail = regexp.MustCompile(`\|\|\s*(?:true|:\s*)(?:\s|$)`)

func (r SLP094) Check(d *diff.Diff) []Finding {
	var out []Finding
	for _, f := range d.Files {
		if f.IsDelete || isDocFile(f.Path) {
			continue
		}
		if !isShellLikeFile(f.Path) {
			continue
		}
		for _, ln := range f.AddedLines() {
			if slp094SilentFail.MatchString(ln.Content) {
				out = append(out, Finding{
					RuleID:   r.ID(),
					Severity: r.DefaultSeverity(),
					File:     f.Path,
					Line:     ln.NewLineNo,
					Message:  "|| true suppresses command failure — handle the error or explicitly comment why it's safe",
					Snippet:  strings.TrimSpace(ln.Content),
				})
			}
		}
	}
	return out
}

func isShellLikeFile(path string) bool {
	lower := strings.ToLower(path)
	if strings.HasSuffix(lower, ".sh") || strings.HasSuffix(lower, ".bash") {
		return true
	}
	if strings.Contains(lower, "makefile") || strings.HasSuffix(lower, ".mk") {
		return true
	}
	if strings.HasSuffix(lower, ".yml") || strings.HasSuffix(lower, ".yaml") {
		return strings.Contains(lower, "ci") || strings.Contains(lower, "workflow")
	}
	return false
}
