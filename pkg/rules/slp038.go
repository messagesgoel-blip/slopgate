package rules

import (
	"regexp"
	"strings"

	"github.com/messagesgoel-blip/slopgate/pkg/diff"
)

// SLP038 flags SQL queries using a pr_number/PR identifier parameter without
// also scoping by repo and branch.
//
// Rationale: PR numbers are not globally unique across repositories. A query
// that filters only by PR number can return data from the wrong repo if
// multiple repos have PRs with the same number. CodeRabbit often flags this
// as a cross-repo data leakage vulnerability.
//
// Languages: Go.
//
// Scope: only added lines in Go files.
type SLP038 struct{}

func (SLP038) ID() string                { return "SLP038" }
func (SLP038) DefaultSeverity() Severity { return SeverityWarn }
func (SLP038) Description() string {
	return "SQL query by PR number without repo/branch scoping may leak cross-repo data"
}

// prQueryRe matches SQL queries that filter by pr_number/PR without repo/branch.
// Looks for patterns like: WHERE ... pr_number = ? or WHERE ... PRNumber = ?
var prQueryRe = regexp.MustCompile(`(?i)WHERE.*pr[_-]?number.*=`)

// scopeByRepoOrBranchRe matches if the query scopes by repo and/or branch.
var scopeByRepoOrBranchRe = regexp.MustCompile(`(?i)(WHERE|AND).*(repo|branch).*[=><]`)

// isGo reports whether the file is a Go file.

func (r SLP038) Check(d *diff.Diff) []Finding {
	var out []Finding
	for _, f := range d.Files {
		if f.IsDelete {
			continue
		}
		if !isGoFile(f.Path) {
			continue
		}

		for _, line := range f.AddedLines() {
			content := line.Content
			// If query filters by PR number but NOT by repo or branch, flag it.
			if prQueryRe.MatchString(content) && !scopeByRepoOrBranchRe.MatchString(content) {
				out = append(out, Finding{
					RuleID:   r.ID(),
					Severity: r.DefaultSeverity(),
					File:     f.Path,
					Line:     line.NewLineNo,
					Message:  r.Description(),
					Snippet:  strings.TrimSpace(content),
				})
			}
		}
	}
	return out
}
