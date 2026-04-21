package rules

import (
	"regexp"
	"strings"

	"github.com/messagesgoel-blip/slopgate/pkg/diff"
)

// SLP037 flags INSERT or UPDATE statements in Go files that are not wrapped in a transaction,
// when there is no evidence of transaction handling (BeginTx, Commit, Rollback) in the added lines.
//
// Rationale: AI agents generating database code might forget to wrap write operations in transactions,
// leading to potential race conditions or inconsistent state when concurrent writers are present.
//
// Languages: Go.
//
// Scope: only added lines in Go files.
type SLP037 struct{}

func (SLP037) ID() string                { return "SLP037" }
func (SLP037) DefaultSeverity() Severity { return SeverityWarn }
func (SLP037) Description() string {
	return "INSERT or UPDATE statement without apparent transaction handling"
}

// insertUpdateRe matches common INSERT or UPDATE statements in Go database/sql.
// Uses word boundaries to avoid matching "updated_at" as UPDATE.
// Includes QueryRow variants since they can also contain write statements.
var insertUpdateRe = regexp.MustCompile(`(?i)\.(Exec(Context)?|Query(Context)?|QueryRow(Context)?)\s*\([^)]*\b(INSERT|UPDATE)\b`)

// dbCallStartRe matches the start of a DB call (Exec, Query, QueryRow) to find
// the line that begins the call when the INSERT/UPDATE is on a different line.
var dbCallStartRe = regexp.MustCompile(`(?i)\.(Exec(Context)?|Query(Context)?|QueryRow(Context)?)\s*\(`)

// txAssignRe matches "tx :=" or "tx, err :=" etc., with word boundary to avoid "ctx :=".
var txAssignRe = regexp.MustCompile(`(?:^|\s)tx(?:\s*,\s*\w+)*\s*:?=`)

func hasTransactionSignal(content string) bool {
	return strings.Contains(content, "BeginTx") ||
		strings.Contains(content, "sql.Tx") ||
		txAssignRe.MatchString(content) ||
		strings.Contains(content, ".Commit(") ||
		strings.Contains(content, ".Rollback(") ||
		strings.Contains(content, "Commit(") ||
		strings.Contains(content, "Rollback(")
}

func (r SLP037) Check(d *diff.Diff) []Finding {
	var out []Finding
	for _, f := range d.Files {
		if f.IsDelete {
			continue
		}
		if !isGoFile(f.Path) {
			continue
		}
		// Check per-hunk so a transaction in one hunk doesn't suppress
		// findings in an unrelated hunk.
		for _, h := range f.Hunks {
			var addedLines []diff.Line
			var addedContent strings.Builder
			for _, line := range h.Lines {
				if line.Kind != diff.LineAdd {
					continue
				}
				addedLines = append(addedLines, line)
				addedContent.WriteString(line.Content)
				addedContent.WriteString("\n")
			}
			addedStr := addedContent.String()
			if !insertUpdateRe.MatchString(addedStr) {
				continue
			}
			if hasTransactionSignal(addedStr) {
				continue
			}
			// The regex may match across lines. For single-line matches, report
			// that line. For multi-line matches, find the line starting the DB call.
			reported := make(map[int]bool)
			for _, line := range addedLines {
				if insertUpdateRe.MatchString(line.Content) && !reported[line.NewLineNo] {
					reported[line.NewLineNo] = true
					out = append(out, Finding{
						RuleID:   r.ID(),
						Severity: r.DefaultSeverity(),
						File:     f.Path,
						Line:     line.NewLineNo,
						Message:  r.Description(),
						Snippet:  strings.TrimSpace(line.Content),
					})
				}
			}
			// If no single line matched but the whole block did, find the DB call start.
			if len(reported) == 0 {
				for _, line := range addedLines {
					if dbCallStartRe.MatchString(line.Content) {
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
	}
	return out
}
