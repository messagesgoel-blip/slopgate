package rules

import (
	"strings"

	"github.com/messagesgoel-blip/slopgate/pkg/diff"
)

// SLP096 flags new shell scripts that don't contain set -e, set -o pipefail,
// or set -o errexit in the first 20 lines. Shell scripts without error
// propagation continue execution after command failures, leading to masked
// errors and corrupted state.
type SLP096 struct{}

func (SLP096) ID() string                { return "SLP096" }
func (SLP096) DefaultSeverity() Severity { return SeverityWarn }
func (SLP096) Description() string {
	return "shell script missing set -e — add error propagation to prevent masked failures"
}

func (r SLP096) Check(d *diff.Diff) []Finding {
	var out []Finding
	for _, f := range d.Files {
		if f.IsDelete || isDocFile(f.Path) {
			continue
		}
		if !f.IsNew {
			continue
		}
		ext := strings.ToLower(f.Path)
		if !strings.HasSuffix(ext, ".sh") && !strings.HasSuffix(ext, ".bash") {
			continue
		}
		if !strings.HasPrefix(f.Path, "a/") && strings.Contains(f.Path, "source") {
			continue
		}

		hasSetE := false
		lineCount := 0
		for _, ln := range f.AddedLines() {
			if lineCount >= 20 {
				break
			}
			lineCount++
			content := strings.TrimSpace(ln.Content)
			if strings.Contains(content, "set -e") || strings.Contains(content, "set -o pipefail") || strings.Contains(content, "set -o errexit") {
				hasSetE = true
				break
			}
		}

		if !hasSetE && len(f.AddedLines()) > 0 {
			out = append(out, Finding{
				RuleID:   r.ID(),
				Severity: r.DefaultSeverity(),
				File:     f.Path,
				Line:     0,
				Message:  "shell script missing set -e, set -o pipefail, or set -o errexit — add to prevent masked failures",
				Snippet:  f.Path,
			})
		}
	}
	return out
}
