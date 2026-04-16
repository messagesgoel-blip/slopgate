package rules

import (
	"fmt"
	"regexp"

	"github.com/messagesgoel-blip/slopgate/pkg/diff"
)

// SLP009 flags env-var lookups that are added in the diff where no
// corresponding env-var setup (os.Setenv / process.env.X = ...) exists
// in any added line across the entire diff. This is a "drift" pattern:
// the code reads an env var that nothing in this change writes, making
// the new code fragile and dependent on external state that may not
// exist.
//
// Languages: Go, JS/TS.
//
// Scope: this rule only looks within the diff itself. It does NOT
// check .env files, CI config, or pre-existing code.
type SLP009 struct{}

func (SLP009) ID() string                { return "SLP009" }
func (SLP009) DefaultSeverity() Severity { return SeverityInfo }
func (SLP009) Description() string {
	return "env-var lookup added without corresponding setup in diff"
}

// --- Go regexes ---

// slp009GoGetenv matches os.Getenv("NAME") and captures the var name.
var slp009GoGetenv = regexp.MustCompile(`os\.Getenv\s*\(\s*"([^"]+)"\s*\)`)

// slp009GoSetenv matches os.Setenv("NAME", ... and captures the var name.
var slp009GoSetenv = regexp.MustCompile(`os\.Setenv\s*\(\s*"([^"]+)"\s*,`)

// slp009GoLookupEnv matches os.LookupEnv("NAME") and captures the var name.
var slp009GoLookupEnv = regexp.MustCompile(`os\.LookupEnv\s*\(\s*"([^"]+)"\s*\)`)

// --- JS/TS regexes ---

// slp009JSDotAccess matches process.env.NAME (dot access).
var slp009JSDotAccess = regexp.MustCompile(`process\.env\.([A-Za-z_][A-Za-z0-9_]*)`)

// slp009JSBracketAccess matches process.env["NAME"] (bracket access with string key).
var slp009JSBracketAccess = regexp.MustCompile(`process\.env\[\s*"([^"]+)"\s*\]`)

// slp009JSDotAssign matches process.env.NAME = (dot assignment).
var slp009JSDotAssign = regexp.MustCompile(`process\.env\.([A-Za-z_][A-Za-z0-9_]*)\s*=`)

// slp009JSBracketAssign matches process.env["NAME"] = (bracket assignment).
var slp009JSBracketAssign = regexp.MustCompile(`process\.env\[\s*"([^"]+)"\s*\]\s*=`)

// --- envLoc tracks a single env-var access site in the diff. ---

type envLoc struct {
	name string
	file string
	line int
}

// --- Check ---

func (r SLP009) Check(d *diff.Diff) []Finding {
	// First and second pass: collect all env-var reads and writes across
	// ALL files in the diff, considering only added lines.
	var reads []envLoc
	setVars := map[string]bool{}

	for _, f := range d.Files {
		if f.IsDelete {
			continue
		}
		isGo := isGoFile(f.Path)
		isJS := isJSOrTSFile(f.Path)
		if !isGo && !isJS {
			continue
		}

		for _, ln := range f.AddedLines() {
			if isGo {
				// Collect reads: os.Getenv("NAME")
				for _, m := range slp009GoGetenv.FindAllStringSubmatch(ln.Content, -1) {
					reads = append(reads, envLoc{name: m[1], file: f.Path, line: ln.NewLineNo})
				}
				// Collect writes: os.Setenv("NAME", ...)
				for _, m := range slp009GoSetenv.FindAllStringSubmatch(ln.Content, -1) {
					setVars[m[1]] = true
				}
				// os.LookupEnv("NAME") also counts as a write for our
				// purposes — it provides a way to handle the missing-var
				// case inline. If LookupEnv exists we don't flag Getenv.
				for _, m := range slp009GoLookupEnv.FindAllStringSubmatch(ln.Content, -1) {
					setVars[m[1]] = true
				}
			}
			if isJS {
				// Collect reads: process.env.NAME
				for _, m := range slp009JSDotAccess.FindAllStringSubmatch(ln.Content, -1) {
					// Skip if this line is actually an assignment (process.env.X = ...).
					if slp009JSDotAssign.MatchString(ln.Content) {
						continue
					}
					reads = append(reads, envLoc{name: m[1], file: f.Path, line: ln.NewLineNo})
				}
				// Collect reads: process.env["NAME"]
				for _, m := range slp009JSBracketAccess.FindAllStringSubmatch(ln.Content, -1) {
					if slp009JSBracketAssign.MatchString(ln.Content) {
						continue
					}
					reads = append(reads, envLoc{name: m[1], file: f.Path, line: ln.NewLineNo})
				}
				// Collect writes: process.env.NAME = ...
				for _, m := range slp009JSDotAssign.FindAllStringSubmatch(ln.Content, -1) {
					setVars[m[1]] = true
				}
				// Collect writes: process.env["NAME"] = ...
				for _, m := range slp009JSBracketAssign.FindAllStringSubmatch(ln.Content, -1) {
					setVars[m[1]] = true
				}
			}
		}
	}

	// Build findings: for each read without a corresponding write.
	var out []Finding
	seen := map[string]bool{} // deduplicate by name+file+line
	for _, r := range reads {
		if setVars[r.name] {
			continue
		}
		key := fmt.Sprintf("%s:%s:%d", r.name, r.file, r.line)
		if seen[key] {
			continue
		}
		seen[key] = true
		out = append(out, Finding{
			RuleID:   "SLP009",
			Severity: SeverityInfo,
			File:     r.file,
			Line:     r.line,
			Message:  fmt.Sprintf("env-var %q read but not set anywhere in this diff", r.name),
			Snippet:  fmt.Sprintf("env-var drift: %s", r.name),
		})
	}
	return out
}
