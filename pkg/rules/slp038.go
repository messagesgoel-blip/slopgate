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

// prQueryRe matches SQL queries that filter by pr_number/PR/pr without repo/branch.
var prQueryRe = regexp.MustCompile(`(?i)WHERE.*\bpr[_-]?number\b.*=|WHERE.*\bPRNumber\b.*=|WHERE.*\bpr\b.*=`)

// scopeByRepoOrBranchRe matches if the query scopes by repo and/or branch,
// including IN, LIKE, and IS comparisons. Uses word boundaries to avoid
// matching substrings like "reporter" for "repo".
var scopeByRepoOrBranchRe = regexp.MustCompile(`(?i)\b(?:WHERE|AND)\b.*\b(?:repo|branch)\b.*(?:<=|>=|!=|<>|=|>|<|\bIN\b\s*\(|\bLIKE\b|\bIS\b)`)

func (r SLP038) Check(d *diff.Diff) []Finding {
	var out []Finding
	for _, f := range d.Files {
		if f.IsDelete {
			continue
		}
		if !isGoFile(f.Path) {
			continue
		}

		lines := f.AddedLines()
		for i, line := range lines {
			// Accumulate contiguous added lines into a block for multiline SQL detection.
			var block strings.Builder
			block.WriteString(line.Content)
			block.WriteString(" ")
			for j := i + 1; j < len(lines) && lines[j].NewLineNo == lines[j-1].NewLineNo+1; j++ {
				block.WriteString(lines[j].Content)
				block.WriteString(" ")
			}
			blockStr := block.String()

			// Split the block by common SQL statement delimiters so a scoped
			// statement doesn't suppress an unscoped one in the same block.
			statements := strings.Split(blockStr, ";")
			for _, stmt := range statements {
				if prQueryRe.MatchString(stmt) && !scopeByRepoOrBranchRe.MatchString(stmt) {
					out = append(out, Finding{
						RuleID:   r.ID(),
						Severity: r.DefaultSeverity(),
						File:     f.Path,
						Line:     line.NewLineNo,
						Message:  r.Description(),
						Snippet:  strings.TrimSpace(line.Content),
					})
					break
				}
			}
		}
	}
	return out
}
