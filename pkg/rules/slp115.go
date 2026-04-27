package rules

import (
	"strings"

	"github.com/messagesgoel-blip/slopgate/pkg/diff"
)

type SLP115 struct{}

func (SLP115) ID() string                { return "SLP115" }
func (SLP115) DefaultSeverity() Severity { return SeverityInfo }
func (SLP115) Description() string {
	return "narrow extension check — add broader extension coverage for related file types"
}

var slp115ExtensionGroups = []struct {
	narrow  string
	broader []string
}{
	{narrow: ".js", broader: []string{".js", ".mjs", ".cjs"}},
	{narrow: ".ts", broader: []string{".ts", ".mts", ".cts"}},
	{narrow: ".py", broader: []string{".py", ".pyi", ".pyw"}},
	{narrow: ".css", broader: []string{".css", ".scss", ".less", ".sass"}},
}

func (r SLP115) Check(d *diff.Diff) []Finding {
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
				content := cleaned
				if content == "" {
					content = raw
				}
				contentLower := strings.ToLower(content)
				rawLower := strings.ToLower(raw)

				for _, group := range slp115ExtensionGroups {
					groupContent := contentLower
					if !strings.Contains(groupContent, group.narrow) {
						if !strings.Contains(rawLower, group.narrow) {
							continue
						}
						groupContent = rawLower
					}

					hasNarrow := false
					for _, ext := range group.broader {
						if strings.Contains(groupContent, ext) && ext == group.narrow {
							hasNarrow = true
							break
						}
					}

					hasAnyBroader := false
					for _, ext := range group.broader {
						if ext != group.narrow && strings.Contains(groupContent, ext) {
							hasAnyBroader = true
							break
						}
					}

					if hasNarrow && !hasAnyBroader {
						narrowExt := strings.TrimPrefix(group.narrow, ".")
						out = append(out, Finding{
							RuleID:   r.ID(),
							Severity: r.DefaultSeverity(),
							File:     f.Path,
							Line:     ln.NewLineNo,
							Message:  "narrow extension check for ." + narrowExt + " — consider including " + strings.Join(group.broader, ", "),
							Snippet:  ln.Content,
						})
						break
					}
				}
			}
		}
	}
	return out
}