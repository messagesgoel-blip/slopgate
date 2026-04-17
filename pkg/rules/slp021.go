package rules

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/messagesgoel-blip/slopgate/pkg/diff"
)

// SLP021 flags inconsistent naming style — when both camelCase and
// snake_case identifiers appear in the same hunk. AI agents often
// mix naming conventions (e.g. userName + user_name) because they
// don't internalize project style.
//
// Exempt: SCREAMING_SNAKE (constants); single-char names; test
// files; doc files.
type SLP021 struct{}

func (SLP021) ID() string                { return "SLP021" }
func (SLP021) DefaultSeverity() Severity { return SeverityInfo }
func (SLP021) Description() string {
	return "mixed camelCase and snake_case naming in the same hunk — pick one style"
}

// slp021Identifier matches likely identifier names from declarations.
var slp021Identifier = regexp.MustCompile(`(?:func|def|var|let|const|int|long|float|double|string|bool|auto|public|private|protected)\s+(\w+)`)
var slp021AssignIdent = regexp.MustCompile(`(\w+)\s*:?=`)

func slp021IsScreamingSnake(s string) bool {
	return s == strings.ToUpper(s) && strings.Contains(s, "_") && len(s) > 1
}

func slp021IsCamelCase(s string) bool {
	if len(s) < 2 {
		return false
	}
	hasLower := false
	hasUpper := false
	for _, r := range s {
		if r >= 'a' && r <= 'z' {
			hasLower = true
		}
		if r >= 'A' && r <= 'Z' {
			hasUpper = true
		}
	}
	return hasLower && hasUpper && !strings.Contains(s, "_")
}

func slp021IsSnakeCase(s string) bool {
	if len(s) < 2 {
		return false
	}
	return strings.Contains(s, "_") && s == strings.ToLower(s)
}

func slp021ExtractIdentifiers(content string) []string {
	clean := stripCommentAndStrings(content)
	var ids []string
	for _, m := range slp021Identifier.FindAllStringSubmatch(clean, -1) {
		if len(m) > 1 {
			ids = append(ids, m[1])
		}
	}
	for _, m := range slp021AssignIdent.FindAllStringSubmatch(clean, -1) {
		if len(m) > 1 && !slp021IsScreamingSnake(m[1]) {
			ids = append(ids, m[1])
		}
	}
	return ids
}

func (r SLP021) Check(d *diff.Diff) []Finding {
	var out []Finding
	for _, f := range d.Files {
		if f.IsDelete || isDocFile(f.Path) {
			continue
		}
		isTest := isGoTestFile(f.Path) || isJavaTestFile(f.Path) ||
			isPythonTestFile(f.Path) || isJSTestFile(f.Path) || isRustTestFile(f.Path)
		if isTest {
			continue
		}
		for _, h := range f.Hunks {
			var camelNames, snakeNames []string
			var firstAddedLineNo int
			for _, ln := range h.Lines {
				if ln.Kind != diff.LineAdd {
					continue
				}
				if firstAddedLineNo == 0 {
					firstAddedLineNo = ln.NewLineNo
				}
				for _, id := range slp021ExtractIdentifiers(ln.Content) {
					if len(id) <= 1 || slp021IsScreamingSnake(id) {
						continue
					}
					if slp021IsCamelCase(id) {
						camelNames = append(camelNames, id)
					} else if slp021IsSnakeCase(id) {
						snakeNames = append(snakeNames, id)
					}
				}
			}
			if len(camelNames) >= 1 && len(snakeNames) >= 1 {
				out = append(out, Finding{
					RuleID:   r.ID(),
					Severity: r.DefaultSeverity(),
					File:     f.Path,
					Line:     firstAddedLineNo,
					Message:  fmt.Sprintf("mixed naming styles — camelCase (%s) and snake_case (%s) in same hunk", strings.Join(camelNames, ", "), strings.Join(snakeNames, ", ")),
					Snippet:  fmt.Sprintf("camelCase: %v, snake_case: %v", camelNames, snakeNames),
				})
			}
		}
	}
	return out
}
