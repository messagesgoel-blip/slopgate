package rules

import (
	"regexp"
	"strings"

	"github.com/messagesgoel-blip/slopgate/pkg/diff"
)

// SLP101 flags feature-flag conditionals and empty alternate branches.
// This is a common AI slop pattern: scaffolding a gated branch without
// clarifying whether the divergence is intentional.
type SLP101 struct{}

func (SLP101) ID() string                { return "SLP101" }
func (SLP101) DefaultSeverity() Severity { return SeverityWarn }
func (SLP101) Description() string {
	return "feature-flag conditional or empty else branch — review for dead code or unintended scaffolding"
}

var slp101FlagCheck = regexp.MustCompile(`(?i)(?:if|when)\s*\(?\s*(?:!\s*)?(?:featureFlag|isEnabled|useFeature|feature\(|toggle\(|ldClient|growthbook|unleash\.|flags\[)\w*`)

var slp101EmptyBranch = regexp.MustCompile(`(?i)else\s*\{\s*\}`)

func (r SLP101) Check(d *diff.Diff) []Finding {
	var out []Finding
	for _, f := range d.Files {
		if f.IsDelete || isDocFile(f.Path) {
			continue
		}
		if isTestFile(f.Path) {
			continue
		}
		if !isJSOrTSFile(f.Path) && !isGoFile(f.Path) && !isJavaFile(f.Path) {
			continue
		}

		for _, ln := range f.AddedLines() {
			content := strings.TrimSpace(ln.Content)
			if strings.HasPrefix(content, "//") ||
				strings.HasPrefix(content, "/*") ||
				strings.HasPrefix(content, "*") {
				continue
			}

			if slp101FlagCheck.MatchString(content) {
				out = append(out, Finding{
					RuleID:   r.ID(),
					Severity: r.DefaultSeverity(),
					File:     f.Path,
					Line:     ln.NewLineNo,
					Message:  "feature-flag conditional detected — review for intended divergence or dead code",
					Snippet:  ln.Content,
				})
			}

			if slp101EmptyBranch.MatchString(content) {
				out = append(out, Finding{
					RuleID:   r.ID(),
					Severity: r.DefaultSeverity(),
					File:     f.Path,
					Line:     ln.NewLineNo,
					Message:  "empty else branch detected — remove the dead branch or implement the alternate path",
					Snippet:  ln.Content,
				})
			}
		}
	}
	return out
}
