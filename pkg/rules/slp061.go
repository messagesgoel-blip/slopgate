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

func (r SLP061) Check(d *diff.Diff) []Finding {
	var out []Finding
	for _, f := range d.Files {
		if f.IsDelete || !isGoFile(f.Path) {
			continue
		}
		added := f.AddedLines()

		// Build a map of struct name -> field count from added lines.
		structFields := map[string]int{}
		i := 0
		for i < len(added) {
			ln := added[i]
			m := slp061StructDef.FindStringSubmatch(strings.TrimSpace(ln.Content))
			if m == nil {
				i++
				continue
			}
			structName := m[1]
			// Count fields until closing brace.
			j := i + 1
			depth := 1 // we are inside struct {
			for j < len(added) && depth > 0 {
				bl := added[j]
				trimmed := strings.TrimSpace(bl.Content)
				if trimmed == "" || strings.HasPrefix(trimmed, "//") {
					j++
					continue
				}
				depth += strings.Count(trimmed, "{") - strings.Count(trimmed, "}")
				if depth == 0 {
					break
				}
				// Heuristic field line: contains a space (field + type).
				if depth == 1 && strings.Contains(trimmed, " ") {
					structFields[structName]++
				}
				j++
			}
			i = j + 1
		}

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
						// Strip pointer star.
						beforeBrace = strings.TrimPrefix(beforeBrace, "*")
						if beforeBrace != "" && !strings.Contains(beforeBrace, " ") && !strings.Contains(beforeBrace, "(") {
							structName = beforeBrace
							startLine = ln.NewLineNo
						}
						break
					}
					if strings.Contains(next, "func") || strings.HasPrefix(next, "//") {
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
