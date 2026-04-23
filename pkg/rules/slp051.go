package rules

import (
	"regexp"
	"strings"

	"github.com/messagesgoel-blip/slopgate/pkg/diff"
)

// SLP051 flags bare function calls in added code that may be undefined.
// We look for identifier(arg...) patterns and skip builtins, keywords,
// and method calls (which come from imported packages).
type SLP051 struct{}

func (SLP051) ID() string                { return "SLP051" }
func (SLP051) DefaultSeverity() Severity { return SeverityBlock }
func (SLP051) Description() string {
	return "call to potentially undefined function — implement or import it"
}

// goKeywords are Go keywords, predeclared identifiers, builtins, and type
// names that may legally appear before '(' without implying an undefined call.
var goKeywords = map[string]bool{
	"if": true, "for": true, "switch": true, "select": true,
	"return": true, "defer": true, "go": true, "panic": true,
	"recover": true, "print": true, "println": true,
	"new": true, "make": true, "len": true, "cap": true,
	"append": true, "copy": true, "delete": true, "close": true,
	"complex": true, "real": true, "imag": true,
	"min": true, "max": true, "clear": true,
	"func": true,
	"bool": true, "byte": true, "rune": true,
	"string": true, "int": true, "int8": true, "int16": true, "int32": true, "int64": true,
	"uint": true, "uint8": true, "uint16": true, "uint32": true, "uint64": true, "uintptr": true,
	"float32": true, "float64": true, "complex64": true, "complex128": true,
	"error": true, "any": true,
	"true": true, "false": true, "nil": true,
}

// undefinedCallPattern matches a bare identifier followed immediately by '('.
var undefinedCallPattern = regexp.MustCompile(`\b([a-zA-Z_]\w*)\s*\(`)

func (r SLP051) Check(d *diff.Diff) []Finding {
	var out []Finding
	for _, f := range d.Files {
		if f.IsDelete || !isGoFile(f.Path) {
			continue
		}
		localFuncs := slp051LocalSymbols(f)
		for _, h := range f.Hunks {
			for _, ln := range h.Lines {
				if ln.Kind != diff.LineAdd {
					continue
				}
				content := strings.TrimSpace(stripCommentAndStrings(ln.Content))
				if content == "" {
					continue
				}
				// Find all bare calls.
				for _, m := range undefinedCallPattern.FindAllStringSubmatchIndex(content, -1) {
					name := content[m[2]:m[3]]
					if goKeywords[name] || isSafetyTestName(name) {
						continue
					}
					// Skip method calls: check the character immediately before the match start.
					if m[2] > 0 && content[m[2]-1] == '.' {
						continue
					}
					// Skip if there is a local func definition for this name somewhere in the file.
					if localFuncs[name] {
						continue
					}
					out = append(out, Finding{
						RuleID:   r.ID(),
						Severity: r.DefaultSeverity(),
						File:     f.Path,
						Line:     ln.NewLineNo,
						Message:  "call to undefined function " + name + " — implement or import it",
						Snippet:  strings.TrimSpace(ln.Content),
					})
				}
			}
		}
	}
	return out
}

// funcDefPattern matches a function/method definition.
var funcDefPattern = regexp.MustCompile(`^func\s+(?:\([^)]+\)\s+)?([a-zA-Z_]\w*)(?:\s*\[[^\]]+\])?\s*\(`)

var typeDefPattern = regexp.MustCompile(`^type\s+([a-zA-Z_]\w*)\b`)

func slp051LocalSymbols(f diff.File) map[string]bool {
	localSymbols := make(map[string]bool)
	for _, h := range f.Hunks {
		for _, ln := range h.Lines {
			if ln.Kind == diff.LineDelete {
				continue
			}
			if m := funcDefPattern.FindStringSubmatch(strings.TrimSpace(ln.Content)); m != nil {
				localSymbols[m[1]] = true
			}
			if m := typeDefPattern.FindStringSubmatch(strings.TrimSpace(ln.Content)); m != nil {
				localSymbols[m[1]] = true
			}
		}
	}
	return localSymbols
}
