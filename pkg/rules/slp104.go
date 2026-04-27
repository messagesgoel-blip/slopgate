package rules

import (
	"regexp"
	"strconv"
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

var slp104NumLit = `(?:0[xX][0-9A-Fa-f][0-9A-Fa-f_]*|0[bB][01][01_]*|0[oO][0-7][0-7_]*|[0-9][0-9_]*(?:\.[0-9_]+)?(?:[eE][+\-]?[0-9_]+)?)`
var slp104IntLit = `(?:0[xX][0-9A-Fa-f][0-9A-Fa-f_]*|0[bB][01][01_]*|0[oO][0-7][0-7_]*|[0-9][0-9_]*)`
var slp104MakeByte = regexp.MustCompile(`make\s*\(\s*\[\s*\]byte\s*,\s*(` + slp104IntLit + `)(?:\s*,\s*(` + slp104IntLit + `))?\s*\)`)
var slp104BufioSize = regexp.MustCompile(`bufio\.NewReaderSize\s*\((?:[^(),]+|\([^()]*\))+,\s*` + slp104IntLit + `\s*\)`)
var slp104BufferConfig = regexp.MustCompile(`(?i)\b(?:bufferSize|maxSize|bufSize|chunkSize)\b(?:\s*:\s*[^:=]+?)?\s*(?:[:=]|:=)\s*` + slp104NumLit)

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

		isGo := isGoFile(f.Path)
		newFinding := func(ln diff.Line, msg string) Finding {
			return Finding{
				RuleID:   r.ID(),
				Severity: r.DefaultSeverity(),
				File:     f.Path,
				Line:     ln.NewLineNo,
				Message:  msg,
				Snippet:  ln.Content,
			}
		}

		for _, ln := range f.AddedLines() {
			trimmed := strings.TrimSpace(ln.Content)
			// Skip comment lines
			if strings.HasPrefix(trimmed, "//") || strings.HasPrefix(trimmed, "/*") || strings.HasPrefix(trimmed, "*") {
				continue
			}
			if isGo {
				if slp104MakeByte.MatchString(trimmed) {
					for _, m := range slp104MakeByte.FindAllStringSubmatch(trimmed, -1) {
						lenVal := m[1]
						capVal := m[2]
						isZero := false
						if parsed, err := strconv.ParseInt(strings.ReplaceAll(lenVal, "_", ""), 0, 64); err == nil {
							isZero = parsed == 0
						}
						if !isZero || capVal != "" {
							out = append(out, newFinding(ln, "hardcoded buffer size in make — use a named constant"))
							break
						}
					}
				}
				if slp104BufioSize.MatchString(trimmed) {
					out = append(out, newFinding(ln, "hardcoded buffer size in bufio — use a named constant"))
				}
			}
			if slp104BufferConfig.MatchString(trimmed) {
				out = append(out, newFinding(ln, "hardcoded buffer limit — use a named constant"))
			}
		}
	}
	return out
}
