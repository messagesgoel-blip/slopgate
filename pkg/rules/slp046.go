package rules

import (
	"regexp"
	"strings"

	"github.com/messagesgoel-blip/slopgate/pkg/diff"
)

// SLP046 flags when related functions (one calls another) are scattered
// across different packages in the same diff.
//
// Rationale: Functions that call each other should be colocated in the same
// package when possible. Splitting them across packages increases coupling
// and makes the code harder to understand and maintain.
type SLP046 struct{}

func (SLP046) ID() string                { return "SLP046" }
func (SLP046) DefaultSeverity() Severity { return SeverityWarn }
func (SLP046) Description() string {
	return "function defined in one file is called from another file — consider colocating related logic"
}

// callPattern returns a regexp matching a bare call to funcName, ensuring it
// is not preceded by a dot and is followed by '('.
func callPattern(funcName string) *regexp.Regexp {
	return regexp.MustCompile(`(?m)(^|[^.\w])` + regexp.QuoteMeta(funcName) + `\s*\(`)
}

func (r SLP046) Check(d *diff.Diff) []Finding {
	// fileFuncs maps file path -> set of function names defined in that file
	fileFuncs := make(map[string]map[string]bool)
	// fileBodies maps file path -> concatenated added lines content (for call scanning)
	fileBodies := make(map[string]string)

	for _, f := range d.Files {
		if f.IsDelete || !isGoFile(f.Path) {
			continue
		}
		funcs := make(map[string]bool)
		var bodyParts []string
		for _, ln := range f.AddedLines() {
			bodyParts = append(bodyParts, ln.Content)
			m := funcDefPattern.FindStringSubmatch(ln.Content)
			if m != nil {
				funcs[m[1]] = true
			}
		}
		if len(funcs) > 0 {
			fileFuncs[f.Path] = funcs
		}
		if len(bodyParts) > 0 {
			fileBodies[f.Path] = "\n" + strings.Join(bodyParts, "\n") + "\n"
		}
	}

	// If only one file has added functions, nothing to flag.
	if len(fileFuncs) < 2 {
		return nil
	}

	// Collect cross-file call pairs: for each (fileA, funcName, fileB), report
	// that funcName defined in fileA is called from fileB.
	var out []Finding

	for fileA, funcsA := range fileFuncs {
		for fileB, funcsB := range fileFuncs {
			if fileA == fileB {
				continue
			}
			bodyB := fileBodies[fileB]
			for funcName := range funcsA {
				// Skip if fileB also defines this function (duplicate name).
				if funcsB[funcName] {
					continue
				}
				// Use word-boundary regex to detect bare calls (not method calls).
				if callPattern(funcName).MatchString(bodyB) {
					out = append(out, Finding{
						RuleID:   r.ID(),
						Severity: r.DefaultSeverity(),
						File:     fileA,
						Line:     0,
						Message:  "function " + funcName + " defined in " + fileA + " is called from " + fileB + " — consider colocating related logic",
						Snippet:  "",
					})
				}
			}
		}
	}

	return out
}
