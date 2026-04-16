package rules

import (
	"fmt"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/messagesgoel-blip/slopgate/pkg/diff"
)

// SLP012 flags TODO / FIXME / HACK / XXX comments added in the current
// diff. Pre-existing markers in the file are ignored — only the lines
// the diff *adds* count. Markdown / text / docs are excluded.
//
// Rationale: TODO comments in backlog docs or pre-existing code are
// fine. TODO comments in freshly generated code are a tell that an AI
// agent stopped before finishing the job and committed the stub anyway.
type SLP012 struct{}

func (SLP012) ID() string                { return "SLP012" }
func (SLP012) DefaultSeverity() Severity { return SeverityBlock }
func (SLP012) Description() string {
	return "TODO/FIXME/HACK/XXX comment added in new code"
}

// slp012Pattern matches a comment whose body STARTS with a TODO-class
// marker. Requiring the marker to be the first word after the comment
// prefix distinguishes real TODO markers from prose that happens to
// mention the word "TODO" — e.g. "// This function detects TODO comments"
// is not slop, but "// TODO: fix this" is.
var slp012Pattern = regexp.MustCompile(`^\s*(?://|/\*|\*|#)\s*(TODO|FIXME|HACK|XXX)\b`)

// slp012TrackedPattern matches a TODO/FIXME/HACK marker that is the
// leading marker at comment start AND is followed by a parenthetical
// ticket or issue reference. These are considered tracked work, not
// slop. Anchored to the comment prefix so a trailing TODO(TICKET) in
// the same line cannot mask an untracked leading TODO.
var slp012TrackedPattern = regexp.MustCompile(`^\s*(?://|/\*|\*|#)\s*(TODO|FIXME|HACK|XXX)\([^)]+\)`)

// docExtensions marks file types where TODO markers are part of normal
// writing, not slop.
var docExtensions = map[string]bool{
	".md":       true,
	".markdown": true,
	".txt":      true,
	".rst":      true,
	".adoc":     true,
	".html":     true,
	".tex":      true,
}

// isDocFile reports whether the given path is a documentation file
// where TODO markers should be ignored.
func isDocFile(path string) bool {
	return docExtensions[strings.ToLower(filepath.Ext(path))]
}

func (r SLP012) Check(d *diff.Diff) []Finding {
	var out []Finding
	for _, f := range d.Files {
		if f.IsDelete || isDocFile(f.Path) {
			continue
		}
		for _, ln := range f.AddedLines() {
			m := slp012Pattern.FindStringSubmatch(ln.Content)
			if m == nil {
				continue
			}
			// Tracked TODO/FIXME with a ticket reference is legitimate.
			if slp012TrackedPattern.MatchString(ln.Content) {
				continue
			}
			out = append(out, Finding{
				RuleID:   r.ID(),
				Severity: r.DefaultSeverity(),
				File:     f.Path,
				Line:     ln.NewLineNo,
				Message:  fmt.Sprintf("new %s comment in committed code — finish or delete it", m[1]),
				Snippet:  strings.TrimSpace(ln.Content),
			})
		}
	}
	return out
}
