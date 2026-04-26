package rules

import (
	"regexp"
	"strings"

	"github.com/messagesgoel-blip/slopgate/pkg/diff"
)

// SLP108 flags Open/Connect calls without a preceding or following defer
// close or timeout/deadline setup. Connection management without guaranteed
// cleanup is a common AI slop pattern.
type SLP108 struct{}

func (SLP108) ID() string                { return "SLP108" }
func (SLP108) DefaultSeverity() Severity { return SeverityBlock }
func (SLP108) Description() string {
	return "open/connect without defer close or timeout — resources will leak on panic"
}

var slp108Open = regexp.MustCompile(`(?i)(?:os\.Open|OpenFile|sql\.Open|net\.Dial|http\.Get|http\.Post|Connect|Listen|NewClient)\s*\(`)
var slp108Fetch = regexp.MustCompile(`(?i)fetch\s*\(\s*['"\x60]`)
var slp108DeferClose = regexp.MustCompile(`(?i)defer\s+.*(?:Close|Cancel)\s*\(`)
var slp108Timeout = regexp.MustCompile(`(?i)(?:context\.WithTimeout|context\.WithDeadline|setTimeout|time\.After|setRequestTimeout)`)

func (r SLP108) Check(d *diff.Diff) []Finding {
	var out []Finding
	for _, f := range d.Files {
		if f.IsDelete || isDocFile(f.Path) {
			continue
		}
		if !isGoFile(f.Path) && !isJSOrTSFile(f.Path) {
			continue
		}

		for _, h := range f.Hunks {
			hasOpen := false
			hasDeferClose := false
			hasTimeout := false
			for _, ln := range h.Lines {
				if ln.Kind != diff.LineAdd {
					continue
				}
				if slp108Open.MatchString(ln.Content) || slp108Fetch.MatchString(ln.Content) {
					hasOpen = true
				}
				if slp108DeferClose.MatchString(ln.Content) {
					hasDeferClose = true
				}
				if slp108Timeout.MatchString(ln.Content) {
					hasTimeout = true
				}
			}
			if hasOpen && !hasDeferClose && !hasTimeout {
				for _, ln := range h.Lines {
					if ln.Kind == diff.LineAdd && (slp108Open.MatchString(ln.Content) || slp108Fetch.MatchString(ln.Content)) {
						out = append(out, Finding{
							RuleID:   r.ID(),
							Severity: r.DefaultSeverity(),
							File:     f.Path,
							Line:     ln.NewLineNo,
							Message:  "open/connect without defer close or timeout — add resource lifecycle management",
							Snippet:  strings.TrimSpace(ln.Content),
						})
					}
				}
			}
		}
	}
	return out
}
