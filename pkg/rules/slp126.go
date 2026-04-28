package rules

import (
	"regexp"
	"strconv"
	"strings"

	"github.com/messagesgoel-blip/slopgate/pkg/diff"
)

// SLP126 flags migration SQL that introduces *_id references without adding
// a matching index in the same diff.
type SLP126 struct{}

func (SLP126) ID() string                { return "SLP126" }
func (SLP126) DefaultSeverity() Severity { return SeverityWarn }
func (SLP126) Description() string {
	return "migration adds *_id reference without index — add CREATE INDEX for join/cascade performance"
}

var slp126RefLineRe = regexp.MustCompile(`(?i)(foreign\s+key|references\s+[a-z0-9_]+|add\s+column\s+[a-z0-9_]+_id\b)`)
var slp126IDTokenRe = regexp.MustCompile(`(?i)\b([a-z0-9_]+_id)\b`)
var slp126IndexLineRe = regexp.MustCompile(`(?i)\b(create\s+(unique\s+)?index|add\s+index|index\s*\()`)

type slp126Candidate struct {
	line    diff.Line
	column  string
	snippet string
}

func slp126IsMigrationSQL(path string) bool {
	lower := strings.ToLower(path)
	return strings.HasSuffix(lower, ".sql") &&
		(strings.Contains(lower, "migration") || strings.Contains(lower, "migrations"))
}

func (r SLP126) Check(d *diff.Diff) []Finding {
	var out []Finding
	for _, f := range d.Files {
		if f.IsDelete || !slp126IsMigrationSQL(f.Path) {
			continue
		}

		indexedCols := map[string]bool{}
		var candidates []slp126Candidate

		for _, ln := range f.AddedLines() {
			content := strings.TrimSpace(stripCommentAndStrings(ln.Content))
			if content == "" {
				continue
			}

			if slp126IndexLineRe.MatchString(content) {
				for _, m := range slp126IDTokenRe.FindAllStringSubmatch(strings.ToLower(content), -1) {
					if len(m) == 2 {
						indexedCols[m[1]] = true
					}
				}
			}

			if !slp126RefLineRe.MatchString(strings.ToLower(content)) {
				continue
			}
			for _, m := range slp126IDTokenRe.FindAllStringSubmatch(strings.ToLower(content), -1) {
				if len(m) != 2 {
					continue
				}
				candidates = append(candidates, slp126Candidate{
					line:    ln,
					column:  m[1],
					snippet: strings.TrimSpace(ln.Content),
				})
			}
		}

		if len(candidates) == 0 {
			continue
		}

		seen := map[string]bool{}
		for _, c := range candidates {
			key := c.column + ":" + strconv.Itoa(c.line.NewLineNo)
			if seen[key] {
				continue
			}
			seen[key] = true

			if indexedCols[c.column] {
				continue
			}
			out = append(out, Finding{
				RuleID:   r.ID(),
				Severity: r.DefaultSeverity(),
				File:     f.Path,
				Line:     c.line.NewLineNo,
				Message:  "migration adds reference column '" + c.column + "' without matching index — add CREATE INDEX for lookup/cascade performance",
				Snippet:  c.snippet,
			})
		}
	}
	return out
}
