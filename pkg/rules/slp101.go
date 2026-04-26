package rules

import (
	"regexp"
	"strings"

	"github.com/messagesgoel-blip/slopgate/pkg/diff"
)

// SLP101 flags feature flag or toggle checks where both branches of the
// conditional are identical, or one branch is empty. This is a dead flag.
type SLP101 struct{}

func (SLP101) ID() string                { return "SLP101" }
func (SLP101) DefaultSeverity() Severity { return SeverityWarn }
func (SLP101) Description() string {
	return "dead feature flag — both branches are identical or one is empty"
}

var slp101FlagCheck = regexp.MustCompile(`(?i)(?:if|when)\s*\(?\s*(?:!\s*)?(?:featureFlag|isEnabled|useFeature|feature\(|toggle\(|ldClient|growthbook|unleash\.|flags\[)\w*`)

var slp101EmptyBranch = regexp.MustCompile(`(?i)else\s*\{\s*\}`)

func (r SLP101) Check(d *diff.Diff) []Finding {
	var out []Finding
	for _, f := range d.Files {
		if f.IsDelete || isDocFile(f.Path) {
			continue
		}
		if strings.Contains(strings.ToLower(f.Path), ".test.") ||
			strings.Contains(strings.ToLower(f.Path), ".spec.") {
			continue
		}
		if !isJSOrTSFile(f.Path) && !isGoFile(f.Path) && !isJavaFile(f.Path) {
			continue
		}

		for _, ln := range f.AddedLines() {
			content := strings.TrimSpace(ln.Content)

			if slp101FlagCheck.MatchString(content) {
				out = append(out, Finding{
					RuleID:   r.ID(),
					Severity: r.DefaultSeverity(),
					File:     f.Path,
					Line:     ln.NewLineNo,
					Message:  "feature-flag conditional detected — review for intended divergence or dead code",
					Snippet:  content,
				})
			}

			if slp101EmptyBranch.MatchString(content) {
				out = append(out, Finding{
					RuleID:   r.ID(),
					Severity: r.DefaultSeverity(),
					File:     f.Path,
					Line:     ln.NewLineNo,
					Message:  "empty else branch in feature-flagged code — remove the dead branch",
					Snippet:  content,
				})
			}
		}
	}
	return out
}
