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

var slp118IndexRe = regexp.MustCompile(`(?:[A-Za-z0-9_]|[\)\]\}])\s*\[\d+\]`)
var slp118GoGuardRe = regexp.MustCompile(`if len\((.+?)\)\s*>\s*(\d+)|if len\((.+?)\)\s*>=\s*(\d+)`)
var slp118JSGuardRe = regexp.MustCompile(`if\s*\(\s*(.+?)\.length\s*>\s*(\d+)\)|if\s*\(\s*(.+?)\.length\s*>=\s*(\d+)\)`)
var slp118PyGuardRe = regexp.MustCompile(`if len\((.+?)\)\s*>\s*(\d+)|if len\((.+?)\)\s*>=\s*(\d+)`)

type slp118Guard struct {
	collection string
	bound      int
	op         string
}

func slp118ExtractGoGuard(line string) *slp118Guard {
	m := slp118GoGuardRe.FindStringSubmatch(line)
	if m == nil {
		return nil
	}
	if m[1] != "" {
		return &slp118Guard{collection: m[1], op: ">"}
	}
	if m[3] != "" {
		return &slp118Guard{collection: m[3], op: ">="}
	}
	return nil
}

func slp118ExtractJSGuard(line string) *slp118Guard {
	m := slp118JSGuardRe.FindStringSubmatch(line)
	if m == nil {
		return nil
	}
	if m[1] != "" {
		return &slp118Guard{collection: m[1], op: ">"}
	}
	if m[3] != "" {
		return &slp118Guard{collection: m[3], op: ">="}
	}
	return nil
}

func slp118ExtractPyGuard(line string) *slp118Guard {
	m := slp118PyGuardRe.FindStringSubmatch(line)
	if m == nil {
		return nil
	}
	if m[1] != "" {
		return &slp118Guard{collection: m[1], op: ">"}
	}
	if m[3] != "" {
		return &slp118Guard{collection: m[3], op: ">="}
	}
	return nil
}

func slp118ExtractGuard(line string, filePath string) *slp118Guard {
	if isGoFile(filePath) {
		return slp118ExtractGoGuard(line)
	}
	if isJSOrTSFile(filePath) {
		return slp118ExtractJSGuard(line)
	}
	if isPythonFile(filePath) {
		return slp118ExtractPyGuard(line)
	}
	return nil
}

func slp118IsGuardedBy(guard *slp118Guard, content string) bool {
	if guard == nil {
		return false
	}
	return strings.Contains(content, guard.collection)
}

func slp118IsBlockEnd(content string) bool {
	trimmed := strings.TrimSpace(content)
	return trimmed == "}" || trimmed == "fi" || trimmed == "end"
}

func slp118IsIndexAccess(content string) bool {
	locs := slp118IndexRe.FindAllStringIndex(content, -1)
	for _, loc := range locs {
		end := loc[1]
		if end < len(content) && isAlpha(content[end]) {
			continue
		}
		return true
	}
	return false
}

func isAlpha(b byte) bool {
	return (b >= 'a' && b <= 'z') || (b >= 'A' && b <= 'Z')
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
			var currentGuard *slp118Guard
			for _, ln := range h.Lines {
				if ln.Kind != diff.LineAdd {
					continue
				}
				content := stripCommentAndStrings(ln.Content)
				content = strings.TrimSpace(content)
				if content == "" {
					continue
				}

				if slp118IsBlockEnd(content) {
					currentGuard = nil
					continue
				}

				guard := slp118ExtractGuard(content, f.Path)
				if guard != nil {
					currentGuard = guard
					continue
				}

				if strings.HasPrefix(content, "if ") || strings.HasPrefix(content, "for ") ||
					strings.HasPrefix(content, "while ") || strings.HasPrefix(content, "//") ||
					strings.HasPrefix(content, "/*") || strings.HasPrefix(content, "*") {
					continue
				}

				if slp118IsGuardedBy(currentGuard, content) {
					continue
				}

				if slp118IsIndexAccess(content) {
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