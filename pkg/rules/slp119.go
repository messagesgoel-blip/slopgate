package rules

import (
	"regexp"
	"strings"

	"github.com/messagesgoel-blip/slopgate/pkg/diff"
)

type SLP119 struct{}

func (SLP119) ID() string                { return "SLP119" }
func (SLP119) DefaultSeverity() Severity { return SeverityWarn }
func (SLP119) Description() string {
	return "TrimSuffix/TrimPrefix result used without checking if the suffix/prefix was present"
}

var slp119EmptyStrRe = regexp.MustCompile(`(?:==|!=)\s*""|(?:==|!=)\s*''`)
var slp119SafetyCallRe = regexp.MustCompile(`\b(?:HasSuffix|HasPrefix|hasSuffix|hasPrefix)\s*\(`)
var slp119TrimCallRe = regexp.MustCompile(`\b(?:TrimSuffix|TrimPrefix|trimSuffix|trimPrefix)\s*\(`)

func slp119HasSafetyCheck(text string) bool {
	if slp119SafetyCallRe.MatchString(text) {
		return true
	}
	return slp119EmptyStrRe.MatchString(text)
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

				if slp119TrimCallRe.MatchString(content) {

					window := stripCommentAndStrings(ln.Content)
					for j := 1; j <= 2; j++ {
						if i-j >= 0 && lines[i-j].Kind != diff.LineDelete {
							window += " " + stripCommentAndStrings(lines[i-j].Content)
						}
						if i+j < len(lines) && lines[i+j].Kind != diff.LineDelete {
							window += " " + stripCommentAndStrings(lines[i+j].Content)
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
