package rules

import (
	"strings"

	"github.com/messagesgoel-blip/slopgate/pkg/diff"
)

// SLP064 flags test files that set up mocks but contain no behavioural
// assertions — verifying mock calls is not enough.
//
// Reuses assertion token detection from SLP001.
type SLP064 struct{}

func (SLP064) ID() string                { return "SLP064" }
func (SLP064) DefaultSeverity() Severity { return SeverityWarn }
func (SLP064) Description() string {
	return "mocks present but no assertions in added test lines"
}

// slp064MockTokens are substrings that indicate mock usage.
var slp064MockTokens = []string{
	"mock", "Mock", "gomock", "mockery",
}

func hasMock(content string) bool {
	for _, tok := range slp064MockTokens {
		if strings.Contains(content, tok) {
			return true
		}
	}
	return false
}

// hasAssertionLine reports whether the line contains a test assertion.
// We reuse the same token list as SLP001's library assertions plus Go's
// standard testing.T assertions.
func hasAssertionLine(line string) bool {
	// Standard Go testing assertions.
	tSuffixes := []string{".Error(", ".Errorf(", ".Fatal(", ".Fatalf(", ".FailNow(", ".Fail("}
	for _, s := range tSuffixes {
		if strings.Contains(line, "t"+s) {
			return true
		}
	}
	// Library assertions from SLP001.
	libraryTokens := []string{
		"assert.", "require.",
		"Expect(", "Eventually(", "Consistently(",
		"So(",
		"is.",
		"qt.",
		"c.Check(", "c.Assert(",
	}
	for _, tok := range libraryTokens {
		if strings.Contains(line, tok) {
			return true
		}
	}
	return false
}

func (r SLP064) Check(d *diff.Diff) []Finding {
	var out []Finding
	for _, f := range d.Files {
		if f.IsDelete {
			continue
		}
		// Only test files.
		if testFileLang(f.Path) == "" {
			continue
		}
		added := f.AddedLines()
		var firstMockLine *diff.Line
		hasAnyAssertion := false
		for _, ln := range added {
			if hasMock(ln.Content) && firstMockLine == nil {
				firstMockLine = &ln
			}
			if hasAssertionLine(ln.Content) {
				hasAnyAssertion = true
			}
		}
		if firstMockLine != nil && !hasAnyAssertion {
			out = append(out, Finding{
				RuleID:   r.ID(),
				Severity: r.DefaultSeverity(),
				File:     f.Path,
				Line:     firstMockLine.NewLineNo,
				Message:  "mocks present but no assertions — verify actual behavior, not just mock calls",
				Snippet:  strings.TrimSpace(firstMockLine.Content),
			})
		}
	}
	return out
}
