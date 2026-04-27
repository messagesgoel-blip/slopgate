package rules

import (
	"strings"

	"github.com/messagesgoel-blip/slopgate/pkg/diff"
)

type SLP119 struct{}

func (SLP119) ID() string                { return "SLP119" }
func (SLP119) DefaultSeverity() Severity { return SeverityWarn }
func (SLP119) Description() string {
	return "TrimSuffix/TrimPrefix result used without checking if the suffix/prefix was present"
}

func slp119HasSafetyCheck(text string) bool {
	return strings.Contains(text, "HasSuffix") || strings.Contains(text, "HasPrefix") ||
		strings.Contains(text, "hasSuffix") || strings.Contains(text, "hasPrefix") ||
		strings.Contains(text, `== ""`) || strings.Contains(text, `!= ""`) ||
		strings.Contains(text, `==''`) || strings.Contains(text, `!=''`)
}

func (r SLP119) Check(d *diff.Diff) []Finding {
	var out []Finding
	for _, f := range d.Files {
		if f.IsDelete || isDocFile(f.Path) {
			continue
		}
		if !isGoFile(f.Path) && !isJSOrTSFile(f.Path) && !isPythonFile(f.Path) {
			continue
		}

		for _, h := range f.Hunks {
			lines := h.Lines
			for i, ln := range lines {
				if ln.Kind != diff.LineAdd {
					continue
				}
				content := stripCommentAndStrings(ln.Content)
				content = strings.TrimSpace(content)
				if content == "" {
					continue
				}

				if strings.HasPrefix(content, "//") || strings.HasPrefix(content, "/*") {
					continue
				}

				if strings.Contains(content, "TrimSuffix") || strings.Contains(content, "TrimPrefix") ||
					strings.Contains(content, "trimSuffix") || strings.Contains(content, "trimPrefix") {

					window := ln.Content
					for j := 1; j <= 2; j++ {
						if i-j >= 0 {
							window += " " + lines[i-j].Content
						}
						if i+j < len(lines) {
							window += " " + lines[i+j].Content
						}
					}
					if slp119HasSafetyCheck(window) {
						continue
					}
					out = append(out, Finding{
						RuleID:   r.ID(),
						Severity: r.DefaultSeverity(),
						File:     f.Path,
						Line:     ln.NewLineNo,
						Message:  "TrimSuffix/TrimPrefix result used without checking if the suffix/prefix was present — consider checking with strings.HasSuffix/HasPrefix first",
						Snippet:  ln.Content,
					})
				}
			}
		}
	}
	return out
}