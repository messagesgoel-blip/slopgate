package rules

import (
	"regexp"
	"strings"

	"github.com/messagesgoel-blip/slopgate/pkg/diff"
)

// SLP053 flags numeric config values (timeouts, limits, etc.) added
// without an explanatory comment on the same or immediately preceding
// added line.
type SLP053 struct{}

func (SLP053) ID() string                { return "SLP053" }
func (SLP053) DefaultSeverity() Severity { return SeverityInfo }
func (SLP053) Description() string {
	return "config value lacks rationale comment — explain why this value was chosen"
}

// configKeyPattern matches lines that look like config key = value.
var configKeyPattern = regexp.MustCompile(`(?i)(timeout|limit|max|min|retry|delay|wait|ttl|expire|batch)\s*[:=]\s*(\d+)`)

func (r SLP053) Check(d *diff.Diff) []Finding {
	var out []Finding
	for _, f := range d.Files {
		if f.IsDelete {
			continue
		}
		for _, h := range f.Hunks {
			prevAddedComment := false
			for _, ln := range h.Lines {
				if ln.Kind != diff.LineAdd {
					continue
				}
				content := strings.TrimSpace(ln.Content)
				isComment := strings.HasPrefix(content, "//") || strings.HasPrefix(content, "#") ||
					strings.HasPrefix(content, ";") || strings.HasPrefix(content, "--")
				if isComment {
					prevAddedComment = true
					continue
				}
				m := configKeyPattern.FindStringSubmatch(content)
				if m == nil {
					prevAddedComment = false
					continue
				}
				key, val := m[1], m[2]
				if !prevAddedComment {
					out = append(out, Finding{
						RuleID:   r.ID(),
						Severity: r.DefaultSeverity(),
						File:     f.Path,
						Line:     ln.NewLineNo,
						Message:  "config " + key + " = " + val + " lacks rationale comment — explain why this value was chosen",
						Snippet:  content,
					})
				}
				prevAddedComment = false
			}
		}
	}
	return out
}
