package rules

import (
	"fmt"
	"strings"

	"github.com/messagesgoel-blip/slopgate/pkg/diff"
)

// SLP070 flags diffs that touch too many top-level directories.
type SLP070 struct{}

func (SLP070) ID() string                { return "SLP070" }
func (SLP070) DefaultSeverity() Severity { return SeverityInfo }
func (SLP070) Description() string {
	return "diff touches too many top-level directories"
}

// rootBucket is the sentinel returned for paths with no directory separator.
const rootBucket = "."

func topLevelDir(path string) string {
	before, _, found := strings.Cut(path, "/")
	if !found {
		return rootBucket
	}
	return before
}

func (r SLP070) Check(d *diff.Diff) []Finding {
	var out []Finding
	unique := make(map[string]bool)
	var modified []diff.File
	for _, f := range d.Files {
		if f.IsDelete {
			continue
		}
		if len(f.AddedLines()) == 0 {
			continue
		}
		modified = append(modified, f)
		unique[topLevelDir(f.Path)] = true
	}
	if len(unique) < 6 {
		return out
	}
	msg := fmt.Sprintf("diff touches %d top-level directories — consider splitting into focused PRs", len(unique))
	for _, f := range modified {
		out = append(out, Finding{
			RuleID:   r.ID(),
			Severity: r.DefaultSeverity(),
			File:     f.Path,
			Line:     0,
			Message:  msg,
			Snippet:  "",
		})
	}
	return out
}
