package rules

import (
	"regexp"
	"strings"

	"github.com/messagesgoel-blip/slopgate/pkg/diff"
)

// SLP103 flags hardcoded timeout/duration values that should be named constants.
// dátummal: AI agents frequently write time.Second * 30 or setTimeout(fn, 5000)
// instead of referencing a configuration constant or named value.
type SLP103 struct{}

func (SLP103) ID() string                { return "SLP103" }
func (SLP103) DefaultSeverity() Severity { return SeverityInfo }
func (SLP103) Description() string {
	return "hardcoded timeout/duration — define a named constant instead"
}

var slp103GoDuration = regexp.MustCompile(`time\.(?:Second|Minute|Hour|Millisecond|Microsecond)\s*\*\s*\d+`)
var slp103JSTimeout = regexp.MustCompile(`setTimeout\s*\([^,]+,\s*\d{3,}\s*\)`)
var slp103ConfigTimeout = regexp.MustCompile(`(?i)(timeout|ttl|deadline|max_age)\s*[:=]\s*\d+\s*(?:ms|s|m|h)?`)

func (r SLP103) Check(d *diff.Diff) []Finding {
	var out []Finding
	for _, f := range d.Files {
		if f.IsDelete || isDocFile(f.Path) {
			continue
		}
		if strings.Contains(strings.ToLower(f.Path), ".test.") ||
			strings.Contains(strings.ToLower(f.Path), ".spec.") ||
			isTestFile(f.Path) {
			continue
		}
		if !isGoFile(f.Path) && !isJSOrTSFile(f.Path) && !isPythonFile(f.Path) {
			continue
		}

		for _, ln := range f.AddedLines() {
			content := strings.TrimSpace(ln.Content)
			var msg string
			switch {
			case slp103GoDuration.MatchString(content):
				if !strings.Contains(content, "time.Second * 1") && !strings.Contains(content, "time.Second*1") {
					msg = "hardcoded Go duration — define a named timeout constant"
				}
			case slp103JSTimeout.MatchString(content):
				msg = "hardcoded setTimeout with literal ms — extract to a named constant"
			case slp103ConfigTimeout.MatchString(content):
				msg = "hardcoded timeout in config — use a named constant"
			}
			if msg != "" {
				out = append(out, Finding{
					RuleID:   r.ID(),
					Severity: r.DefaultSeverity(),
					File:     f.Path,
					Line:     ln.NewLineNo,
					Message:  msg,
					Snippet:  content,
				})
			}
		}
	}
	return out
}
