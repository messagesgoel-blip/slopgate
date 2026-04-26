package rules

import (
	"regexp"
	"strings"

	"github.com/messagesgoel-blip/slopgate/pkg/diff"
)

// SLP104 flags hardcoded buffer sizes, capacity limits, or pre-allocations
// that should be named constants or configuration values.
type SLP104 struct{}

func (SLP104) ID() string                { return "SLP104" }
func (SLP104) DefaultSeverity() Severity { return SeverityInfo }
func (SLP104) Description() string {
	return "hardcoded buffer/size limit — define a named constant instead"
}

var slp104MakeByte = regexp.MustCompile(`make\s*\(\s*\[\s*\]byte\s*,\s*\d+\s*\)`)
var slp104BufioSize = regexp.MustCompile(`bufio\.NewReaderSize\s*\([^,]+,\s*\d+\s*\)`)
var slp104BufferConfig = regexp.MustCompile(`(?i)(?:bufferSize|maxSize|bufSize|chunkSize|limit)\s*[:=]\s*\d+`)

func (r SLP104) Check(d *diff.Diff) []Finding {
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
		if !isGoFile(f.Path) && !isJSOrTSFile(f.Path) {
			continue
		}

		for _, ln := range f.AddedLines() {
			content := strings.TrimSpace(ln.Content)
			var msg string
			switch {
			case slp104MakeByte.MatchString(content):
				if !strings.Contains(content, "make([]byte, 0") {
					msg = "hardcoded buffer size in make — use a named constant"
				}
			case slp104BufioSize.MatchString(content):
				msg = "hardcoded buffer size in bufio — use a named constant"
			case slp104BufferConfig.MatchString(content):
				msg = "hardcoded buffer limit — use a named constant"
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
