package rules

import (
	"strings"

	"github.com/messagesgoel-blip/slopgate/pkg/diff"
)

type SLP117 struct{}

func (SLP117) ID() string                { return "SLP117" }
func (SLP117) DefaultSeverity() Severity { return SeverityInfo }
func (SLP117) Description() string {
	return "unanchored regex — add ^, $, or \\b anchor to prevent unintended substring matches"
}

func (r SLP117) Check(d *diff.Diff) []Finding {
	var out []Finding
	for _, f := range d.Files {
		if f.IsDelete || isDocFile(f.Path) {
			continue
		}
		if !isGoFile(f.Path) && !isJSOrTSFile(f.Path) {
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
					continue
				}

				if !strings.Contains(content, "regex") && !strings.Contains(content, "Regex") &&
					!strings.Contains(content, "re.") && !strings.Contains(content, "pattern") &&
					!strings.Contains(content, "Pattern") && !strings.Contains(content, `\d`) &&
					!strings.Contains(content, `\w`) && !strings.Contains(content, `\s`) {
					continue
				}

				if strings.Contains(content, `^`) || strings.Contains(content, `$`) ||
					strings.Contains(content, `\b`) || strings.Contains(content, `\A`) ||
					strings.Contains(content, `\z`) || strings.Contains(content, `\Z`) {
					continue
				}

				out = append(out, Finding{
					RuleID:   r.ID(),
					Severity: r.DefaultSeverity(),
					File:     f.Path,
					Line:     ln.NewLineNo,
					Message:  "unanchored regex pattern — add ^, $, or \\b anchors to prevent unintended substring matches",
					Snippet:  ln.Content,
				})
			}
		}
	}
	return out
}
