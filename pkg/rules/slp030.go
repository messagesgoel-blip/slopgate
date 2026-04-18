package rules

import (
	"strings"

	"github.com/messagesgoel-blip/slopgate/pkg/diff"
)

// SLP030 flags ORM/query methods that select single records without
// excluding sentinel values. AI-generated queries often do:
//
//   File.query().only()       // missing .where('hash', '!=', 'folder-marker')
//   User.find().first()       // could return placeholder user
//   Record.findOne().last()   // missing sentinel filter
//
// This is a semantic bug: the query returns the first/last/only record,
// which might be a sentinel placeholder like 'folder-marker' or 'null-string'.
//
// Exempt: test files, docs, explicit sentinel exclusion present.
type SLP030 struct{}

func (SLP030) ID() string                { return "SLP030" }
func (SLP030) DefaultSeverity() Severity { return SeverityWarn }
func (SLP030) Description() string {
	return "query .only/.first/.last without sentinel exclusion — could return placeholder record"
}

func (r SLP030) Check(d *diff.Diff) []Finding {
	var out []Finding
	for _, f := range d.Files {
		if f.IsDelete || isDocFile(f.Path) {
			continue
		}
		lower := strings.ToLower(f.Path)
		if strings.Contains(lower, ".test.") || strings.Contains(lower, ".spec.") {
			continue
		}
		// Check JS/TS (ORM patterns), Python (Django/SQLAlchemy), Go (GORM)
		if !strings.HasSuffix(lower, ".js") && !strings.HasSuffix(lower, ".ts") &&
			!strings.HasSuffix(lower, ".py") && !strings.HasSuffix(lower, ".go") {
			continue
		}

		for _, ln := range f.AddedLines() {
			content := strings.ToLower(ln.Content)
			// Check for .only()/.first()/.last()/.findOne() patterns
			hasQuerySelect := strings.Contains(content, ".only(") ||
				strings.Contains(content, ".first(") ||
				strings.Contains(content, ".last(") ||
				strings.Contains(content, ".findone(") ||
				strings.Contains(content, "find_one(") ||
				strings.Contains(content, "getone(") ||
				strings.Contains(content, "get_one(") ||
				strings.Contains(content, ".take(1)") ||
				strings.Contains(content, ".limit(1)")

			if hasQuerySelect {
				// Check for sentinel exclusion
				hasSentinelFilter := strings.Contains(content, "!= 'marker") ||
					strings.Contains(content, "!= 'folder") ||
					strings.Contains(content, "!= 'sentinel") ||
					strings.Contains(content, "!= 'placeholder") ||
					strings.Contains(content, "<> 'marker") ||
					strings.Contains(content, "<> 'folder") ||
					strings.Contains(content, "not in [") ||
					strings.Contains(content, ".where('hash'") ||
					strings.Contains(content, ".wherehash") ||
					strings.Contains(content, "exclude(") ||
					strings.Contains(content, ".exclude(") ||
					strings.Contains(content, "filter(hash")

				if !hasSentinelFilter {
					out = append(out, Finding{
						RuleID:   r.ID(),
						Severity: r.DefaultSeverity(),
						File:     f.Path,
						Line:     ln.NewLineNo,
						Message:  "query selects single record without sentinel exclusion — add .where('hash', '!=', 'folder-marker') or similar",
						Snippet:  strings.TrimSpace(ln.Content),
					})
				}
			}
		}
	}
	return out
}