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
	return "shell command suppresses failure with || true or || : — handle the error instead"
}

var slp094SilentFail = regexp.MustCompile(`\|\|\s*(?:true|:)\s*;?\s*(?:\s|$)`)

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
			if slp094IsCommentOnlyLine(ln.Content) {
				continue
			}
			if slp094SilentFail.MatchString(ln.Content) {
				out = append(out, Finding{
					RuleID:   r.ID(),
					Severity: r.DefaultSeverity(),
					File:     f.Path,
					Line:     ln.NewLineNo,
					Message:  "|| true or || : suppresses command failure — handle the error or explicitly comment why it's safe",
					Snippet:  ln.Content,
				})
			}
		}
	}
	return out
}

func slp094IsCommentOnlyLine(content string) bool {
	trim := strings.TrimSpace(content)
	return trim == "" ||
		strings.HasPrefix(trim, "#") ||
		strings.HasPrefix(trim, "//") ||
		strings.HasPrefix(trim, "/*") ||
		strings.HasPrefix(trim, "*/")
}

func isShellLikeFile(path string) bool {
	lower := strings.ToLower(path)
	if strings.HasSuffix(lower, ".sh") || strings.HasSuffix(lower, ".bash") {
		return true
	}
	base := lower
	if i := strings.LastIndex(lower, "/"); i >= 0 {
		base = lower[i+1:]
	}
	if base == "makefile" || strings.HasSuffix(base, ".mk") {
		return true
	}
	if strings.HasSuffix(lower, ".yml") || strings.HasSuffix(lower, ".yaml") {
		// CI token as standalone segment
		isCI := base == "ci.yml" || base == "ci.yaml" ||
			strings.HasPrefix(base, "ci.") || strings.HasPrefix(base, "ci-") ||
			strings.HasSuffix(base, "-ci.yml") || strings.HasSuffix(base, "-ci.yaml") ||
			strings.HasSuffix(base, ".ci.yml") || strings.HasSuffix(base, ".ci.yaml") ||
			base == "ci"

		inGitHubWorkflows := strings.HasPrefix(lower, ".github/workflows/") || strings.Contains(lower, "/.github/workflows/")
		isWorkflow := base == "workflow" ||
			strings.HasPrefix(base, "workflow.") ||
			strings.HasPrefix(base, "workflow-")

		return inGitHubWorkflows || isCI || isWorkflow
	}
	return false
}
