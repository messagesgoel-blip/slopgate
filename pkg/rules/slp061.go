package rules

import (
	"regexp"
	"strconv"
	"strings"

	"github.com/messagesgoel-blip/slopgate/pkg/diff"
)

// SLP061 flags factory/builder functions for structs with fewer than 3
// fields, which is over-engineering — a struct literal is simpler.
//
// Scope: Go files only.
type SLP061 struct{}

func (SLP061) ID() string                { return "SLP061" }
func (SLP061) DefaultSeverity() Severity { return SeverityInfo }
func (SLP061) Description() string {
	return "factory/builder for trivial structs with <3 fields"
}

// slp061FactorySignature matches `func NewXXX(...) StructType {` or
// `func Build(...) StructType {` on a single line. It captures the
// struct type name in group 2 (group 1 is the function name).
var slp061FactorySignature = regexp.MustCompile(`^func\s+(New\w*|Build\w*|New|Build)\s*\(.*\)\s*\*?(\w+)\s*\{`)

// slp061FactorySignatureMulti matches factory signatures where the
// return type is on the next line or has a pointer.
var slp061FactorySignatureMulti = regexp.MustCompile(`^func\s+(New\w*|Build\w*|New|Build)\s*\(`)

// slp061StructDef matches `type XXX struct {` and captures the name.
var slp061StructDef = regexp.MustCompile(`^type\s+(\w+)\s+struct\s*\{`)
var slp061MultiFieldNames = regexp.MustCompile(`^([A-Za-z_]\w*(?:\s*,\s*[A-Za-z_]\w*)+)\s+.+$`)

func (r SLP061) Check(d *diff.Diff) []Finding {
	var out []Finding
	for _, f := range d.Files {
		if f.IsDelete || !isGoFile(f.Path) {
			continue
		}

		// Build a map of struct name -> field count from visible hunk lines, while
		// counting only newly added fields.
		structFields := map[string]int{}
		for _, h := range f.Hunks {
			lines := h.Lines
			for i := 0; i < len(lines); i++ {
				ln := lines[i]
				if ln.Kind == diff.LineDelete {
					continue
				}
				m := slp061StructDef.FindStringSubmatch(strings.TrimSpace(ln.Content))
				if m == nil {
					continue
				}
				structName := m[1]
				depth := 1
				for j := i + 1; j < len(lines) && depth > 0; j++ {
					bl := lines[j]
					if bl.Kind == diff.LineDelete {
						continue
					}
					trimmed := strings.TrimSpace(bl.Content)
					clean := strings.TrimSpace(stripCommentAndStrings(bl.Content))
					if clean == "" {
						continue
					}
					if depth == 1 && bl.Kind == diff.LineAdd {
						fieldPart := trimmed
						if tagIdx := strings.Index(fieldPart, "`"); tagIdx >= 0 {
							fieldPart = strings.TrimSpace(fieldPart[:tagIdx])
						}
						if fieldPart != "" && !strings.HasPrefix(fieldPart, "}") && !strings.HasPrefix(fieldPart, "{") {
							if m := slp061MultiFieldNames.FindStringSubmatch(fieldPart); m != nil {
								structFields[structName] += strings.Count(m[1], ",") + 1
							} else if strings.Contains(fieldPart, " ") {
								structFields[structName]++
							} else {
								structFields[structName]++
							}
						}
					}
					depth += strings.Count(clean, "{") - strings.Count(clean, "}")
				}
			}
		}

		added := f.AddedLines()
		// Scan for factory/builder functions.
		for i := 0; i < len(added); i++ {
			ln := added[i]
			trimmed := strings.TrimSpace(ln.Content)
			var structName string
			var startLine int
			if m := slp061FactorySignature.FindStringSubmatch(trimmed); m != nil {
				structName = m[2]
				startLine = ln.NewLineNo
			} else if slp061FactorySignatureMulti.MatchString(trimmed) {
				// Multi-line signature: consume subsequent added lines
				// until we hit the opening brace, looking for a return
				// type that is a simple identifier.
				j := i + 1
				for j < len(added) {
					next := strings.TrimSpace(added[j].Content)
					if strings.Contains(next, "{") {
						// Extract the simple type name right before {
						beforeBrace := strings.TrimSpace(strings.Split(next, "{")[0])
						beforeBrace = strings.TrimSpace(strings.TrimPrefix(beforeBrace, ")"))
						// Strip pointer star.
						beforeBrace = strings.TrimPrefix(beforeBrace, "*")
						if beforeBrace != "" && !strings.Contains(beforeBrace, " ") && !strings.Contains(beforeBrace, "(") {
							structName = beforeBrace
							startLine = ln.NewLineNo
						}
						break
					}
					if strings.HasPrefix(next, "func ") || strings.HasPrefix(next, "//") {
						break
					}
					j++
				}
			}
			if structName == "" {
				continue
			}
			count, ok := structFields[structName]
			if !ok {
				// Struct definition not in added lines — skip.
				continue
			}
			if count < 3 {
				out = append(out, Finding{
					RuleID:   r.ID(),
					Severity: r.DefaultSeverity(),
					File:     f.Path,
					Line:     startLine,
					Message:  "factory/builder for " + structName + " with only " + strconv.Itoa(count) + " fields is over-engineered — use struct literal",
					Snippet:  trimmed,
				})
			}
		}
	}
	return out
}
