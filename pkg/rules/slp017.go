package rules

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/messagesgoel-blip/slopgate/pkg/diff"
)

// SLP017 flags magic numbers — unexplained numeric literals in
// computation contexts. AI agents frequently sprinkle raw literals
// (tax rates, thresholds, timeouts) instead of defining named
// constants, making code fragile and hard to review.
//
// Exempt: 0, 1, 2; hex/octal literals; array index patterns [N];
// constant/define declarations; ALL_CAPS assignments; test files;
// doc files.
type SLP017 struct{}

func (SLP017) ID() string                { return "SLP017" }
func (SLP017) DefaultSeverity() Severity { return SeverityInfo }
func (SLP017) Description() string {
	return "unexplained numeric literal — define a named constant instead"
}

// slp017Number matches decimal integer or float literals (not hex/octal).
var slp017Number = regexp.MustCompile(`(?:^|[^\w.])((?:0|[1-9]\d*)(?:\.\d+)?)(?:[^\w.]|$)`)

// slp017SmallNumber matches 0, 1, or 2 (common innocuous values).
var slp017SmallNumber = regexp.MustCompile(`^[012]$`)

// slp017HexOctal matches hex (0x...) or octal (0o...) literals.
var slp017HexOctal = regexp.MustCompile(`0[xXoO][\da-fA-F]+`)

// slp017ArrayIndex matches [N] where N is a literal number.
var slp017ArrayIndex = regexp.MustCompile(`\[\d+\]`)

// slp017ConstDecl matches constant/define declarations.
var slp017ConstDecl = regexp.MustCompile(`(?:^|\s)(?:const|final|static\s+final|#define)\s`)

// slp017AllCapsAssign matches ALL_CAPS_NAME = or ALL_CAPS_NAME :=.
var slp017AllCapsAssign = regexp.MustCompile(`[A-Z][A-Z0-9_]*\s*:?=`)

func (r SLP017) Check(d *diff.Diff) []Finding {
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
		for _, ln := range f.AddedLines() {
			clean := stripCommentAndStrings(ln.Content)
			if clean == "" {
				continue
			}
			trimmed := strings.TrimSpace(clean)
			// Skip constant/define declarations.
			if slp017ConstDecl.MatchString(trimmed) {
				continue
			}
			// Skip ALL_CAPS assignments (likely constant definitions).
			if slp017AllCapsAssign.MatchString(trimmed) {
				continue
			}
			// Skip hex/octal literals — they're usually bitmasks.
			if slp017HexOctal.MatchString(clean) {
				continue
			}
			// Blank out array index patterns so [3] doesn't count as magic.
			clean = slp017ArrayIndex.ReplaceAllString(clean, "[_]")
			for _, m := range slp017Number.FindAllStringSubmatch(clean, -1) {
				num := m[1]
				if slp017SmallNumber.MatchString(num) {
					continue
				}
				out = append(out, Finding{
					RuleID:   r.ID(),
					Severity: r.DefaultSeverity(),
					File:     f.Path,
					Line:     ln.NewLineNo,
					Message:  fmt.Sprintf("magic number %s — define a named constant for clarity", num),
					Snippet:  strings.TrimSpace(ln.Content),
				})
				break
			}
		}
	}
	return out
}
