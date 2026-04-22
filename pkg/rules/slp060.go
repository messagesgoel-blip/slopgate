package rules

import (
	"regexp"
	"strings"

	"github.com/messagesgoel-blip/slopgate/pkg/diff"
)

// SLP060 flags interfaces with only one struct declaration (or none) added
// in the same diff. This is a heuristic, not a verified implementation count.
type SLP060 struct{}

func (SLP060) ID() string                { return "SLP060" }
func (SLP060) DefaultSeverity() Severity { return SeverityInfo }
func (SLP060) Description() string {
	return "interface with only one struct declaration found — this heuristic counts structs, not verified implementations"
}

var interfaceDeclPattern = regexp.MustCompile(`^\s*type\s+(\w+)\s+interface\b`)
var structDeclPattern = regexp.MustCompile(`^\s*type\s+(\w+)\s+struct\b`)

func (r SLP060) Check(d *diff.Diff) []Finding {
	var interfaceFindings []Finding
	structCount := 0

	for _, f := range d.Files {
		if f.IsDelete || !strings.HasSuffix(f.Path, ".go") {
			continue
		}
		for _, ln := range f.AddedLines() {
			if structDeclPattern.MatchString(ln.Content) {
				structCount++
				continue
			}
			if m := interfaceDeclPattern.FindStringSubmatch(ln.Content); m != nil {
				interfaceFindings = append(interfaceFindings, Finding{
					RuleID:   r.ID(),
					Severity: r.DefaultSeverity(),
					File:     f.Path,
					Line:     ln.NewLineNo,
					Message:  "interface " + m[1] + " has only one struct declaration found — this heuristic counts structs, not verified implementations; consider using a concrete type if appropriate",
					Snippet:  strings.TrimSpace(ln.Content),
				})
			}
		}
	}

	if len(interfaceFindings) == 0 {
		return nil
	}
	if structCount <= 1 {
		return interfaceFindings
	}
	return nil
}
