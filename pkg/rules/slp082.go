package rules

import (
	"regexp"
	"strings"

	"github.com/messagesgoel-blip/slopgate/pkg/diff"
)

// SLP082 flags JSX array mappings that are missing the key prop.
// This causes React warnings and can lead to rendering issues.
type SLP082 struct{}

func (SLP082) ID() string                { return "SLP082" }
func (SLP082) DefaultSeverity() Severity { return SeverityWarn }
func (SLP082) Description() string {
	return "JSX list item missing key prop - add key to avoid React rendering issues"
}

var (
	// Matches .map() or .forEach() followed by JSX list item
	slp082MapPattern = regexp.MustCompile(`(?i)\.map.*<li|\.map.*<div|\.map.*<span|\.map.*<p|\.map.*<tr|\.map.*<td|\.map.*<th|\.map.*<option`)
	slp082ForEachPattern = regexp.MustCompile(`(?i)\.forEach.*<li|\.forEach.*<div|\.forEach.*<span|\.forEach.*<p|\.forEach.*<tr|\.forEach.*<td|\.forEach.*<th|\.forEach.*<option`)

	// Check for key prop
	slp082HasKeyPattern = regexp.MustCompile(`(?i)\skey\s*[=:]\s*`)
)

func (r SLP082) Check(d *diff.Diff) []Finding {
	var out []Finding
	for _, f := range d.Files {
		if f.IsDelete {
			continue
		}

		// Only check TSX files
		if !strings.HasSuffix(strings.ToLower(f.Path), ".tsx") {
			continue
		}

		for _, h := range f.Hunks {
			for _, ln := range h.Lines {
				if ln.Kind != diff.LineAdd {
					continue
				}

				content := strings.TrimSpace(ln.Content)

				// Check if this is a .map() or .forEach() call with JSX list item
				mapMatch := slp082MapPattern.MatchString(content)
				forEachMatch := slp082ForEachPattern.MatchString(content)

				if mapMatch || forEachMatch {
					// Check if key prop is present
					hasKey := slp082HasKeyPattern.MatchString(content)

					if !hasKey {
						out = append(out, Finding{
							RuleID:   r.ID(),
							Severity: r.DefaultSeverity(),
							File:     f.Path,
							Line:     ln.NewLineNo,
							Message:  "JSX list item created in .map() without key prop - add key={item.id} or similar",
							Snippet:  content,
						})
					}
				}
			}
		}
	}
	return out
}
