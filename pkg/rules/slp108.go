package rules

import (
	"regexp"
	"strings"

	"github.com/messagesgoel-blip/slopgate/pkg/diff"
)

// SLP108 flags Open/Connect calls without a preceding or following defer
// close or timeout/deadline setup. Connection management without guaranteed
// cleanup is a common AI slop pattern.
//
// Known limitation: uses hunk-level correlation — any defer in the same hunk
// suppresses findings for all opens, even if they refer to different variables.
type SLP108 struct{}

func (SLP108) ID() string                { return "SLP108" }
func (SLP108) DefaultSeverity() Severity { return SeverityBlock }
func (SLP108) Description() string {
	return "open/connect without defer close or timeout — resources will leak on panic"
}

var slp108Open = regexp.MustCompile(`(?i)\b(os\.Open|OpenFile|sql\.Open|net\.Dial|http\.Get|http\.Post|Connect|Listen|NewClient)\b\s*\(`)
var slp108Fetch = regexp.MustCompile(`(?i)\bfetch\b\s*\(\s*['"\x60]`)
var slp108DeferClose = regexp.MustCompile(`(?i)defer\s+.*(?:Close|Cancel)\s*\(`)
var slp108Timeout = regexp.MustCompile(`(?i)(?:context\.WithTimeout|context\.WithDeadline|setTimeout|time\.After|setRequestTimeout|\.timeout|AbortSignal|AbortController)`)

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
			hasFetch := false
			hasDeferClose := false
			hasTimeout := false

			for _, ln := range h.Lines {
				if ln.Kind != diff.LineAdd {
					continue
				}
				if slp108Open.MatchString(ln.Content) {
					hasOpen = true
				}
				if slp108Fetch.MatchString(ln.Content) {
					hasFetch = true
				}
				if slp108DeferClose.MatchString(ln.Content) {
					hasDeferClose = true
				}
				if slp108Timeout.MatchString(ln.Content) {
					hasTimeout = true
				}
			}

			if (hasOpen && !hasDeferClose) || (hasFetch && !hasTimeout) {
				for _, ln := range h.Lines {
					if ln.Kind != diff.LineAdd {
						continue
					}
					msg := ""
					if hasOpen && !hasDeferClose && slp108Open.MatchString(ln.Content) {
						msg = "open/connect without defer close — add resource lifecycle management"
					} else if hasFetch && !hasTimeout && slp108Fetch.MatchString(ln.Content) {
						msg = "fetch without timeout — add a timeout or AbortController to prevent hanging"
					}

					if msg != "" {
						out = append(out, Finding{
							RuleID:   r.ID(),
							Severity: r.DefaultSeverity(),
							File:     f.Path,
							Line:     ln.NewLineNo,
							Message:  msg,
							Snippet:  strings.TrimSpace(ln.Content),
						})
					}
				}
			}
		}
	}
	return out
}
