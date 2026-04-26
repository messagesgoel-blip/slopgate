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

var slp104NumLit = `(?:0x[0-9A-Fa-f][0-9A-Fa-f_]*|0b[01][01_]*|0o[0-7][0-7_]*|[0-9][0-9_]*(?:\.[0-9_]+)?(?:[eE][+\-]?[0-9_]+)?)`
var slp104MakeByte = regexp.MustCompile(`make\s*\(\s*\[\s*\]byte\s*,\s*(` + slp104NumLit + `)(?:\s*,\s*(` + slp104NumLit + `))?\s*\)`)
var slp104BufioSize = regexp.MustCompile(`bufio\.NewReaderSize\s*\((?:[^(),]+|\([^()]*\))+,\s*` + slp104NumLit + `\s*\)`)
var slp104BufferConfig = regexp.MustCompile(`(?i)\b(?:bufferSize|maxSize|bufSize|chunkSize)\b\s*(?:[:=]|:=)\s*` + slp104NumLit)

func (r SLP104) Check(d *diff.Diff) []Finding {
	var out []Finding
	for _, f := range d.Files {
		if f.IsDelete || isDocFile(f.Path) {
			continue
		}
		if isTestFile(f.Path) {
			continue
		}
		if !isGoFile(f.Path) && !isJSOrTSFile(f.Path) {
			continue
		}

		for _, ln := range f.AddedLines() {
			trimmed := strings.TrimSpace(ln.Content)
			var msg string
			switch {
			case slp104MakeByte.MatchString(trimmed):
				if m := slp104MakeByte.FindStringSubmatch(trimmed); m != nil {
					lenVal := m[1]
					capVal := m[2]
					// Flag if len > 0 or if cap is explicitly provided (even if len is 0)
					if lenVal != "0" || capVal != "" {
						msg = "hardcoded buffer size in make — use a named constant"
					}
				}
			case slp104BufioSize.MatchString(trimmed):
				msg = "hardcoded buffer size in bufio — use a named constant"
			case slp104BufferConfig.MatchString(trimmed):
				msg = "hardcoded buffer limit — use a named constant"
			}
			if msg != "" {
				out = append(out, Finding{
					RuleID:   r.ID(),
					Severity: r.DefaultSeverity(),
					File:     f.Path,
					Line:     ln.NewLineNo,
					Message:  msg,
					Snippet:  ln.Content,
				})
			}
		}
	}
	return out
}
