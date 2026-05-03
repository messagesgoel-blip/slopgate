package rules

import (
	"regexp"
	"strings"

	"github.com/messagesgoel-blip/slopgate/pkg/diff"
)

// SLP142 flags unsafe path construction where filepath.Join or path.Join
// is used to access files without subsequent symlink evaluation and
// containment checks. This catches potential path traversal and
// symlink escape vulnerabilities.
type SLP142 struct{}

func (SLP142) ID() string                { return "SLP142" }
func (SLP142) DefaultSeverity() Severity { return SeverityWarn }
func (SLP142) Description() string {
	return "unsafe path resolution — evaluate symlinks and check containment"
}

var (
	slp142PathJoin = regexp.MustCompile(`\b(?:filepath|path)\.Join\s*\(`)
	slp142FileOp   = regexp.MustCompile(`\b(?:os\.(?:Open|OpenFile|ReadFile|ReadDir|Remove|RemoveAll|Create|Stat|Lstat)|ioutil\.ReadFile|ioutil\.ReadDir)\s*\(`)
	slp142EvalSafe = regexp.MustCompile(`\bEvalSymlinks\b`)
)

func (r SLP142) Check(d *diff.Diff) []Finding {
	var out []Finding
	for _, f := range d.Files {
		if f.IsDelete || !isSourceLikeFile(f.Path) {
			continue
		}

		added := f.AddedLines()
		for i, ln := range added {
			clean := stripCommentAndStrings(ln.Content)
			if !slp142PathJoin.MatchString(clean) {
				continue
			}

			// Collect surrounding context (heuristic: current line + next 12)
			context := collectHunkBlock(added, i, 12, false)

			// If it contains a file operation using the constructed path
			if slp142FileOp.MatchString(context) {
				// But lacks safety checks — only EvalSymlinks counts as safe
				if !slp142EvalSafe.MatchString(context) {
					out = append(out, Finding{
						RuleID:   r.ID(),
						Severity: r.DefaultSeverity(),
						File:     f.Path,
						Line:     ln.NewLineNo,
						Message:  "path constructed via Join and used in file operation without EvalSymlinks containment check",
						Snippet:  strings.TrimSpace(ln.Content),
					})
				}
			}
		}
	}
	return out
}
