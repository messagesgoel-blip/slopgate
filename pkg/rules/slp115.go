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

func slp115IsExtBorder(b byte) bool {
	return !((b >= 'a' && b <= 'z') || (b >= 'A' && b <= 'Z') || (b >= '0' && b <= '9') || b == '_')
}

func slp115ContainsExtToken(s string, ext string) bool {
	idx := strings.Index(s, ext)
	for idx >= 0 {
		beforeOK := idx == 0 || slp115IsExtBorder(s[idx-1])
		afterIdx := idx + len(ext)
		afterOK := afterIdx >= len(s) || slp115IsExtBorder(s[afterIdx])
		if beforeOK && afterOK {
			return true
		}
		remaining := s[idx+1:]
		next := strings.Index(remaining, ext)
		if next < 0 {
			break
		}
		idx = idx + 1 + next
	}
	return false
}

func slp115AdditionalExts(group struct {
	narrow  string
	broader []string
}) []string {
	var additional []string
	for _, ext := range group.broader {
		if ext != group.narrow {
			additional = append(additional, ext)
		}
	}
	return additional
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
				contentLower := strings.ToLower(content)
				rawLower := strings.ToLower(raw)

				for _, group := range slp115ExtensionGroups {
					groupContent := contentLower
					if !slp115ContainsExtToken(groupContent, group.narrow) {
						if !slp115ContainsExtToken(rawLower, group.narrow) {
							continue
						}
						groupContent = rawLower
					}

					hasNarrow := false
					for _, ext := range group.broader {
						if slp115ContainsExtToken(groupContent, ext) && ext == group.narrow {
							hasNarrow = true
							break
						}
					}

					hasAnyBroader := false
					for _, ext := range group.broader {
						if ext != group.narrow && slp115ContainsExtToken(groupContent, ext) {
							hasAnyBroader = true
							break
						}
					}

					if hasNarrow && !hasAnyBroader {
						narrowExt := strings.TrimPrefix(group.narrow, ".")
						additional := slp115AdditionalExts(group)
						out = append(out, Finding{
							RuleID:   r.ID(),
							Severity: r.DefaultSeverity(),
							File:     f.Path,
							Line:     ln.NewLineNo,
							Message:  "narrow extension check for ." + narrowExt + " — consider including " + strings.Join(additional, ", "),
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
