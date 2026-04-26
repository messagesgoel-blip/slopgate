package rules

import (
	"strings"

	"github.com/messagesgoel-blip/slopgate/pkg/diff"
)

// SLP093 flags when new mock or stub setup is added to a test file without
// a corresponding new assertion. This is a common AI slop pattern: adding
// elaborate mock scaffolding to get tests to compile but forgetting the
// assertions that verify behavior.
//
// Heuristic: count mock/stub terms vs assertion terms per hunk. Flag when
// mocks are added but no corresponding assertions are present.
type SLP093 struct{}

func (SLP093) ID() string                { return "SLP093" }
func (SLP093) DefaultSeverity() Severity { return SeverityWarn }
func (SLP093) Description() string {
	return "new mock/setup added without corresponding assertion — tests may not verify behavior"
}

var slp093MockTerms = []string{
	"mock", "stub", "spyOn", "jest.fn(", "vi.fn(", "sinon.",
	"mockResolvedValue", "mockReturnValue", "mockImplementation",
	"when(", "thenReturn(", "andReturn(",
}

var slp093AssertTerms = []string{
	"expect(", "assert.", "assert(", ".toEqual", ".toBe", ".toHaveBeenCalled",
	"assert.Equal", "assert.EqualValues", "assert.JSONEq",
	"assert.NotNil", "assert.Nil", "assert.True", "assert.False",
	"should(", ".must(",
	"require.", "t.Fatalf", "t.Errorf", "t.Error", "t.Fatal", "t.Skip",
}

func isCommentOnlyLine(content string) bool {
	trim := strings.TrimSpace(content)
	if trim == "" || strings.HasPrefix(trim, "//") || strings.HasPrefix(trim, "#") ||
		strings.HasPrefix(trim, "/*") || strings.HasPrefix(trim, "*") {
		return true
	}
	return false
}

func (r SLP093) Check(d *diff.Diff) []Finding {
	var out []Finding
	for _, f := range d.Files {
		if f.IsDelete || !isTestFile(f.Path) {
			continue
		}
		if !isJSOrTSFile(f.Path) && !isGoFile(f.Path) && !isPythonFile(f.Path) && !isJavaFile(f.Path) {
			continue
		}

		for _, h := range f.Hunks {
			mockCount := 0
			assertCount := 0
			for _, ln := range h.Lines {
				if ln.Kind != diff.LineAdd || isCommentOnlyLine(ln.Content) {
					continue
				}
				content := strings.ToLower(ln.Content)
				for _, t := range slp093MockTerms {
					if strings.Contains(content, strings.ToLower(t)) {
						mockCount++
						break
					}
				}
				for _, t := range slp093AssertTerms {
					if strings.Contains(content, strings.ToLower(t)) {
						assertCount++
						break
					}
				}
			}
			if mockCount > 0 && assertCount == 0 {
				for _, ln := range h.Lines {
					if ln.Kind != diff.LineAdd || isCommentOnlyLine(ln.Content) {
						continue
					}
					content := strings.ToLower(ln.Content)
					for _, t := range slp093MockTerms {
						if strings.Contains(content, strings.ToLower(t)) {
							out = append(out, Finding{
								RuleID:   r.ID(),
								Severity: r.DefaultSeverity(),
								File:     f.Path,
								Line:     ln.NewLineNo,
								Message:  "mock added but no new assertion in this hunk — add expectations to verify behavior",
								Snippet:  strings.TrimSpace(ln.Content),
							})
							break
						}
					}
				}
			}
		}
	}
	return out
}
