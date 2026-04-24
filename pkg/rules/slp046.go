package rules

import (
	"path/filepath"
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

// callPattern returns a regexp matching a bare call to funcName.
// It matches qualified calls like pkg.Func() but ignores likely method calls
// (receiver.Func where receiver is lowercase like r., t., x.)
func callPattern(funcName string) *regexp.Regexp {
	return regexp.MustCompile(`(?m)(^|[^\w])` + regexp.QuoteMeta(funcName) + `\s*\(`)
}

func (r SLP046) Check(d *diff.Diff) []Finding {
	// fileFuncs maps file path -> set of function names defined in that file
	fileFuncs := make(map[string]map[string]bool)
	filePkgs := make(map[string]string)
	// fileBodies maps file path -> concatenated added lines content (for call scanning)
	fileBodies := make(map[string]string)

	for _, f := range d.Files {
		if f.IsDelete || !isGoFile(f.Path) {
			continue
		}
		filePkgs[f.Path] = slp046PackageName(f)
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

	relevantFiles := slp046RelevantFiles(fileFuncs, fileBodies)

	// If fewer than two files participate in the changed call graph, nothing to flag.
	if len(relevantFiles) < 2 {
		return nil
	}

	// Collect cross-file call pairs: for each (fileA, funcName, fileB), report
	// that funcName defined in fileA is called from fileB.
	var out []Finding

	for fileA, funcsA := range fileFuncs {
		for fileB := range relevantFiles {
			if fileA == fileB {
				continue
			}
			if filePkgs[fileA] != "" && filePkgs[fileA] == filePkgs[fileB] {
				continue
			}
			bodyB := fileBodies[fileB]
			if bodyB == "" {
				continue
			}
			for funcName := range funcsA {
				// Skip if fileB also defines this function (duplicate name).
				if funcsB := fileFuncs[fileB]; funcsB != nil && funcsB[funcName] {
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

func slp046RelevantFiles(fileFuncs map[string]map[string]bool, fileBodies map[string]string) map[string]bool {
	out := make(map[string]bool, len(fileFuncs)+len(fileBodies))
	for path := range fileFuncs {
		out[path] = true
	}
	for path := range fileBodies {
		out[path] = true
	}
	return out
}

func slp046PackageName(f diff.File) string {
	for _, h := range f.Hunks {
		for _, ln := range h.Lines {
			if ln.Kind == diff.LineDelete {
				continue
			}
			fields := strings.Fields(strings.TrimSpace(ln.Content))
			if len(fields) >= 2 && fields[0] == "package" {
				return fields[1]
			}
		}
	}
	dir := filepath.Base(filepath.Dir(f.Path))
	if dir == "." || dir == string(filepath.Separator) {
		return ""
	}
	return dir
}
