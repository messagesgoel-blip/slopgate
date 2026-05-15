package rules

import (
	"regexp"
	"strings"

	"github.com/messagesgoel-blip/slopgate/pkg/diff"
)

// SLP116 checks for regex patterns with nested quantifiers that could cause ReDoS.
type SLP116 struct{}

func (SLP116) ID() string                { return "SLP116" }
func (SLP116) DefaultSeverity() Severity { return SeverityWarn }
func (SLP116) Description() string {
	return "regex contains nested quantifiers — potential ReDoS vulnerability"
}

var slp116NestedQuantifier = regexp.MustCompile(`\(\.[*+?]\)[*+?]|\[\^?[^\]]+\][*+?][*+?]`)

var slp116ComplexNested = regexp.MustCompile(`(?:\(\.[*+?]\)|[+*?]\s*[+*?]|\.\*\.\*)`)

func (r SLP116) Check(d *diff.Diff) []Finding {
	var out []Finding
	for _, f := range d.Files {
		if f.IsDelete || isDocFile(f.Path) {
			continue
		}
		if !isGoFile(f.Path) && !isJSOrTSFile(f.Path) && !isPythonFile(f.Path) {
			continue
		}

		for _, h := range f.Hunks {
			for _, ln := range h.Lines {
				if ln.Kind != diff.LineAdd {
					continue
				}
				raw := strings.TrimSpace(ln.Content)
				cleaned := stripCommentAndStrings(ln.Content)
				cleaned = strings.TrimSpace(cleaned)

				if cleaned == "" || strings.HasPrefix(raw, "//") || strings.HasPrefix(raw, "/*") || strings.HasPrefix(raw, "#") {
					continue
				}
				content := raw

				if !strings.Contains(content, "/") && !strings.Contains(content, "regex") &&
					!strings.Contains(content, "Regex") && !strings.Contains(content, "re.") &&
					!strings.Contains(content, "pattern") && !strings.Contains(content, "Pattern") {
					if !strings.Contains(content, `\`) {
						continue
					}
				}

				if strings.Contains(content, "regex") || strings.Contains(content, "Regex") ||
					strings.Contains(content, "re.") || strings.Contains(content, "pattern") ||
					strings.Contains(content, "Pattern") || strings.Contains(content, `\.`) {
					if slp116NestedQuantifier.MatchString(content) ||
						slp116ComplexNested.MatchString(content) {
						out = append(out, Finding{
							RuleID:   r.ID(),
							Severity: r.DefaultSeverity(),
							File:     f.Path,
							Line:     ln.NewLineNo,
							Message:  "regex with nested quantifiers detected — potential ReDoS vulnerability",
							Snippet:  ln.Content,
						})
					}
				}
			}
		}
	}
	return out
}
