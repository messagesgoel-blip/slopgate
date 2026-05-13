package rules

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/messagesgoel-blip/slopgate/pkg/diff"
)

// SLP203 flags SQL INSERT statements that lack conflict-handling clauses.
//
// Primary pattern (high signal): an INSERT INTO ... VALUES statement is added
// in the diff without an ON CONFLICT / ON DUPLICATE KEY / INSERT OR
// REPLACE|IGNORE / MERGE / UPSERT clause. This commonly causes unique-
// constraint violations in production (Sentry crashes).
//
// Languages: Go, Python, Java, JS/TS.
// Scope: diff only — scans added lines within each file hunk.
type SLP203 struct{}

func (SLP203) ID() string                { return "SLP203" }
func (SLP203) DefaultSeverity() Severity { return SeverityBlock }
func (SLP203) Description() string {
	return "INSERT without conflict handling — may cause unique-constraint violation at runtime"
}

// ---------------------------------------------------------------------------
// Regex library
// ---------------------------------------------------------------------------

// insertPattern matches a bare INSERT INTO ... VALUES statement.
var insertPattern = regexp.MustCompile(
	`(?i)INSERT\s+INTO\s+\w+\s*\([^)]*\)\s*VALUES\s*\(`)

// insertTargetPattern extracts the table name from INSERT INTO <table>.
var insertTargetPattern = regexp.MustCompile(`(?i)INSERT\s+INTO\s+(\w+)`)

// upsertPatterns match known conflict-handling clauses.
var upsertPatterns = []*regexp.Regexp{
	// PostgreSQL / SQLite
	regexp.MustCompile(`(?i)\bON\s+CONFLICT\b`),
	regexp.MustCompile(`(?i)\bON\s+DUPLICATE\s+KEY\s+(UPDATE|IGNORE)\b`),
	regexp.MustCompile(`(?i)INSERT\s+OR\s+(REPLACE|IGNORE|ROLLBACK|ABORT|FAIL)\b`),
	// Other
	regexp.MustCompile(`(?i)\bMERGE\s+INTO\b`),
	regexp.MustCompile(`(?i)\bUPSERT\b`),
}

// skipLinePatterns are lines we should never flag.
var slp203SkipLinePatterns = []*regexp.Regexp{
	regexp.MustCompile(`^\s*//`),  // Go/JS/TS/Java comment
	regexp.MustCompile(`^\s*#`),   // Python comment
	regexp.MustCompile(`^\s*/\*`), // block comment start
	regexp.MustCompile(`^\s*\*/`), // block comment end
	regexp.MustCompile(`^\s*\*`),  // doc comment line
	regexp.MustCompile(`^\s*$`),   // blank/whitespace-only line
}

// ---------------------------------------------------------------------------
// Check
// ---------------------------------------------------------------------------

func (r SLP203) Check(d *diff.Diff) []Finding {
	var out []Finding

	for _, f := range d.Files {
		if f.IsDelete {
			continue
		}

		ext := strings.ToLower(f.Path[strings.LastIndex(f.Path, "."):])
		if !slp203SupportedExt(ext) {
			continue
		}

		if isTestFile(f.Path) {
			continue
		}

		for _, h := range f.Hunks {
			for _, ln := range h.Lines {
				if ln.Kind != diff.LineAdd {
					continue
				}
				content := strings.TrimSpace(ln.Content)

				if isSlp203Skippable(content) {
					continue
				}

				if !insertPattern.MatchString(content) {
					continue
				}

				if hasUpsertClause(content) {
					continue
				}

				table := extractInsertTarget(content)
				out = append(out, Finding{
					RuleID:   r.ID(),
					Severity: r.DefaultSeverity(),
					File:     f.Path,
					Line:     ln.NewLineNo,
					Message:  fmt.Sprintf("INSERT without conflict handling — table %q may violate unique constraint", table),
					Snippet:  content,
				})
			}
		}
	}

	return out
}

// slp203SupportedExt returns true for languages SLP203 covers.
func slp203SupportedExt(ext string) bool {
	switch ext {
	case ".go", ".js", ".ts", ".tsx", ".jsx", ".py", ".java", ".kt":
		return true
	}
	return false
}

// isSlp203Skippable returns true for lines that should never be flagged.
func isSlp203Skippable(content string) bool {
	trimmed := strings.TrimLeft(content, " \t")
	for _, pat := range slp203SkipLinePatterns {
		if pat.MatchString(trimmed) {
			return true
		}
	}
	return false
}

// hasUpsertClause returns true if the line contains a known conflict-handling
// clause (ON CONFLICT, ON DUPLICATE KEY, INSERT OR REPLACE/IGNORE, etc.).
func hasUpsertClause(line string) bool {
	for _, pat := range upsertPatterns {
		if pat.MatchString(line) {
			return true
		}
	}
	return false
}

// extractInsertTarget returns the table name from an INSERT INTO line.
func extractInsertTarget(line string) string {
	if m := insertTargetPattern.FindStringSubmatch(line); len(m) > 1 {
		return m[1]
	}
	return "unknown"
}
