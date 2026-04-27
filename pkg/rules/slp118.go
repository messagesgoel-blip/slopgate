package rules

import (
	"regexp"
	"strings"

	"github.com/messagesgoel-blip/slopgate/pkg/diff"
)

type SLP118 struct{}

func (SLP118) ID() string                { return "SLP118" }
func (SLP118) DefaultSeverity() Severity { return SeverityBlock }
func (SLP118) Description() string {
	return "slice or index access without length guard — may panic on empty collection"
}

var slp118IndexRe = regexp.MustCompile(`\[\d+\]`)
var slp118GoGuardRe = regexp.MustCompile(`if len\(.+\)\s*>\s*\d+|if len\(.+\)\s*>=\s*\d+`)
var slp118JSGuardRe = regexp.MustCompile(`if\s*\(.+\.length\s*>\s*\d+\)|if\s*\(.+\.length\s*>=\s*\d+\)`)
var slp118PyGuardRe = regexp.MustCompile(`if len\(.+\)\s*>\s*\d+|if len\(.+\)\s*>=\s*\d+`)

func slp118IsGuarded(prevContent string, filePath string) bool {
	if prevContent == "" {
		return false
	}
	if isGoFile(filePath) && slp118GoGuardRe.MatchString(prevContent) {
		return true
	}
	if isJSOrTSFile(filePath) && slp118JSGuardRe.MatchString(prevContent) {
		return true
	}
	if isPythonFile(filePath) && slp118PyGuardRe.MatchString(prevContent) {
		return true
	}
	return false
}

func (r SLP118) Check(d *diff.Diff) []Finding {
	var out []Finding
	for _, f := range d.Files {
		if f.IsDelete || isDocFile(f.Path) {
			continue
		}
		if !isGoFile(f.Path) && !isJSOrTSFile(f.Path) && !isPythonFile(f.Path) {
			continue
		}

		for _, h := range f.Hunks {
			prevContent := ""
			for _, ln := range h.Lines {
				if ln.Kind != diff.LineAdd {
					continue
				}
				content := stripCommentAndStrings(ln.Content)
				content = strings.TrimSpace(content)
				if content == "" {
					continue
				}

				if strings.HasPrefix(content, "if ") || strings.HasPrefix(content, "for ") ||
					strings.HasPrefix(content, "while ") || strings.HasPrefix(content, "//") ||
					strings.HasPrefix(content, "/*") || strings.HasPrefix(content, "*") {
					prevContent = content
					continue
				}

				if slp118IsGuarded(prevContent, f.Path) {
					prevContent = content
					continue
				}

				if slp118IndexRe.MatchString(content) {
					out = append(out, Finding{
						RuleID:   r.ID(),
						Severity: r.DefaultSeverity(),
						File:     f.Path,
						Line:     ln.NewLineNo,
						Message:  "direct index access without length guard — may panic on empty collection",
						Snippet:  ln.Content,
					})
				}
			}
		}
	}
	return out
}