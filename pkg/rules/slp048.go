package rules

import (
	"path/filepath"
	"regexp"
	"strings"

	"github.com/messagesgoel-blip/slopgate/pkg/diff"
)

// SLP048 flags inconsistent error-handling patterns within the same Go package.
//
// Rationale: In a single package, all files should follow the same style for
// error checks. Mixing "if err != nil { return err }" with silent error
// swallowing makes the code unpredictable and harder to review.
type SLP048 struct{}

func (SLP048) ID() string                { return "SLP048" }
func (SLP048) DefaultSeverity() Severity { return SeverityWarn }
func (SLP048) Description() string {
	return "error handling pattern differs from other files in this package — be consistent"
}

var slp048ErrCheckRe = regexp.MustCompile(`if\s+err\s*!=\s*nil`)

func (r SLP048) Check(d *diff.Diff) []Finding {
	// Group files by package directory.
	// package key = directory of the file.
	// For each directory, track which files check errors and which don't.
	type fileInfo struct {
		path      string
		hasCheck  bool
		firstLine int // added line number for the first added line in the file (for Line in finding)
	}

	dirFiles := make(map[string][]fileInfo)

	for _, f := range d.Files {
		if f.IsDelete || !isGoFile(f.Path) {
			continue
		}
		// Determine package directory.
		dir := filepath.Dir(f.Path)
		if dir == "." {
			dir = ""
		}

		added := f.AddedLines()
		if len(added) == 0 {
			continue
		}

		// Check if any added line has `if err != nil`.
		hasCheck := false
		for _, ln := range added {
			if slp048ErrCheckRe.MatchString(ln.Content) {
				hasCheck = true
				break
			}
		}

		// Also, decide whether this file "should" check errors:
		// heuristic: if the file has a function that returns `error` in its signature
		// but doesn't contain any error check, consider it inconsistent when another
		// file in the same directory does check errors.
		returnsError := false
		for _, ln := range added {
			if strings.Contains(ln.Content, "error") && strings.Contains(ln.Content, "func ") {
				returnsError = true
				break
			}
		}

		fi := fileInfo{
			path:      f.Path,
			hasCheck:  hasCheck,
			firstLine: added[0].NewLineNo,
		}

		// Store only files that either have checks or return errors (so they participate
		// in inconsistency detection).
		if hasCheck || returnsError {
			dirFiles[dir] = append(dirFiles[dir], fi)
		}
	}

	var out []Finding
	for _, files := range dirFiles {
		if len(files) < 2 {
			continue
		}
		// Count how many files have error checks.
		checkCount := 0
		for _, fi := range files {
			if fi.hasCheck {
				checkCount++
			}
		}
		// If all or none have checks, no inconsistency.
		if checkCount == 0 || checkCount == len(files) {
			continue
		}
		// Flag the files that differ from the majority.
		// Majority = at least one file checks, and some don't.
		// We flag the minority side.
		for _, fi := range files {
			if !fi.hasCheck {
				out = append(out, Finding{
					RuleID:   r.ID(),
					Severity: r.DefaultSeverity(),
					File:     fi.path,
					Line:     fi.firstLine,
					Message:  r.Description(),
					Snippet:  "",
				})
			}
		}
	}
	return out
}
