package rules

import (
	"regexp"
	"strings"

	"github.com/messagesgoel-blip/slopgate/pkg/diff"
)

// SLP060 flags interfaces with only one struct declaration (or none) added
// in the same file. This is a heuristic, not a verified implementation count.
type SLP060 struct{}

func (SLP060) ID() string                { return "SLP060" }
func (SLP060) DefaultSeverity() Severity { return SeverityInfo }
func (SLP060) Description() string {
	return "interface with few struct declarations found — may be premature abstraction"
}

var slp060InterfaceDeclPattern = regexp.MustCompile(`^[+\s]*type\s+(\w+)\s+interface\b`)
var slp060StructDeclPattern = regexp.MustCompile(`^[+\s]*type\s+(\w+)\s+struct\b`)

func (r SLP060) Check(d *diff.Diff) []Finding {
	var out []Finding

	for _, f := range d.Files {
		if f.IsDelete || !strings.HasSuffix(f.Path, ".go") {
			continue
		}

		var interfaceFindings []Finding
		structCount := 0

		for _, ln := range f.AddedLines() {
			if slp060StructDeclPattern.MatchString(ln.Content) {
				structCount++
				continue
			}
			if m := slp060InterfaceDeclPattern.FindStringSubmatch(ln.Content); m != nil {
				interfaceFindings = append(interfaceFindings, Finding{
					RuleID:   r.ID(),
					Severity: r.DefaultSeverity(),
					File:     f.Path,
					Line:     ln.NewLineNo,
					Message:  "interface " + m[1] + " has only one struct declaration found — heuristic counts structs, not verified implementations",
					Snippet:  ln.Content,
				})
			}
		}

		// Only flag if this file has 0 or 1 struct declarations alongside interface(s).
		if len(interfaceFindings) > 0 && structCount <= 1 {
			out = append(out, interfaceFindings...)
		}
	}

	return out
}
