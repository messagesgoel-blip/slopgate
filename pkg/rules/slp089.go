package rules

import (
	"path/filepath"
	"regexp"
	"strings"

	"github.com/messagesgoel-blip/slopgate/pkg/diff"
)

// SLP089 flags exported functions, classes, and modules that lack documentation.
// Documentation is critical for maintainability and onboarding.
type SLP089 struct{}

func (SLP089) ID() string                { return "SLP089" }
func (SLP089) DefaultSeverity() Severity { return SeverityInfo }
func (SLP089) Description() string {
	return "exported function/class/module missing docstring - add JSDoc, comments, or documentation"
}

var (
	// Export patterns for JS/TS
	slp089JSExportPatterns = []*regexp.Regexp{
		regexp.MustCompile(`(?i)^export\s+(default\s+)?function\s+\w+`),
		regexp.MustCompile(`(?i)^export\s+(async\s+)?function\s+\w+`),
		regexp.MustCompile(`(?i)^export\s+const\s+\w+\s*=\s*\([^)]*\)\s*=>`),
		regexp.MustCompile(`(?i)^export\s+(async\s+)?const\s+\w+\s*=\s*async?\s*\(`),
		regexp.MustCompile(`(?i)^export\s+(default\s+)?class\s+\w+`),
		regexp.MustCompile(`(?i)^export\s+(interface|type)\s+\w+`),
		regexp.MustCompile(`(?i)^export\s+(default\s+)?module`),
	}

	// Const/function declarations (standalone, exported)
	// These patterns match exported const functions that start with 'export'
	slp089ConstExportPatterns = []*regexp.Regexp{
		regexp.MustCompile(`^export\s+const\s+\w+\s*=\s*\([^)]*\)\s*=>`),
		regexp.MustCompile(`^export\s+const\s+\w+\s*=\s*async?\s*\(|^export\s+async\s+const\s+\w+\s*=\s*\([^)]*\)\s*=>`),
		regexp.MustCompile(`^export\s+(async\s+)?function\s+\w+\s*\(.*?\)`),
		regexp.MustCompile(`^export\s+default\s+function\s+\w+\s*\(.*?\)|^export\s+default\s+const\s+\w+\s*=\s*function|^export\s+default\s+const\s+\w+\s*=\s*\([^)]*\)\s*=>`),
	}

	// Go patterns - must start with uppercase to be exported
	slp089GoExportPatterns = []*regexp.Regexp{
		regexp.MustCompile(`^func\s+[A-Z]\w*\s*\(`),
		regexp.MustCompile(`^type\s+[A-Z]\w*\s+\w+`),
	}

	// Python patterns
	slp089PythonExportPatterns = []*regexp.Regexp{
		regexp.MustCompile(`^def\s+\w+\s*\(`),
	}

	// Comment/docstring patterns
	slp089DocPatterns = []*regexp.Regexp{
		regexp.MustCompile(`(?i)/\*\*`),
		regexp.MustCompile(`(?i)\*/`),
		regexp.MustCompile(`(?i)^\/\/\s+[\w]`),
		regexp.MustCompile(`(?i)['"]{3}[\s\S]*?['"]{3}`),
		regexp.MustCompile(`(?i)^\/\/\s+\w+\s*$`),
	}

	goDocPattern = regexp.MustCompile(`(?i)^\/\/\s+\w`)

	// Pre-compiled regex for extracting export names
	exportNamePattern = regexp.MustCompile(`^(?:export\s+)?(?:const\s+|function\s+|async\s+function\s+|class\s+)\s*(\w+)`)
)

// isExportDeclaration checks if a line is an export statement or export declaration
func isExportDeclaration(content string) bool {
	for _, pattern := range slp089JSExportPatterns {
		if pattern.MatchString(content) {
			return true
		}
	}
	return false
}

// isConstExportDeclaration checks if a line declares a const export (without export prefix)
func isConstExportDeclaration(content string) bool {
	for _, pattern := range slp089ConstExportPatterns {
		if pattern.MatchString(content) {
			return true
		}
	}
	return false
}

// isGoExport checks if a line is a Go export declaration
func isGoExport(content string) bool {
	for _, pattern := range slp089GoExportPatterns {
		if pattern.MatchString(content) {
			return true
		}
	}
	return false
}

// isPythonExport checks if a line is a Python export declaration
func isPythonExport(content string) bool {
	for _, pattern := range slp089PythonExportPatterns {
		if pattern.MatchString(content) {
			return true
		}
	}
	return false
}

// isBraceExport checks if a line is a brace export (re-export)
func isBraceExport(content string) bool {
	content = strings.TrimSpace(content)
	return strings.HasPrefix(content, "export {")
}

// findReexportedName checks if content (e.g., "const add = () => ...") is re-exported via export { add }
// Returns the identifier name if found, empty string otherwise
func findReexportedName(content string) string {
	// Extract function/variable name from declaration like:
	// - const add = (a, b) => ...
	// - function foo() {}
	// - class Bar {}

	if matches := exportNamePattern.FindStringSubmatch(strings.TrimSpace(content)); len(matches) > 1 {
		return matches[1]
	}
	return ""
}


// doExportedInHunk checks if a const/function is exported via brace export in this hunk
// pass the const/function line index to avoid self-referencing
func doExportedInHunk(h diff.Hunk, constLineIdx int, constName string) bool {
	reNamePattern := regexp.MustCompile(`\b` + regexp.QuoteMeta(constName) + `\b`)
	for j, ln := range h.Lines {
		if j == constLineIdx {
			continue
		}
		if strings.HasPrefix(strings.TrimSpace(ln.Content), "export {") {
			// Check if constName is in the export list
			content := strings.TrimSpace(ln.Content)
			// Match export { name } or export { name as alias } or export { name, other }
			if reNamePattern.MatchString(content) {
				return true
			}
		}
	}
	return false
}

// isExportLine checks if this line is an export-like declaration
func isExportLine(content string, h diff.Hunk, idx int) (IsExport bool, isGo bool, isPython bool) {
	content = strings.TrimSpace(content)

	if isExportDeclaration(content) || isConstExportDeclaration(content) {
		return true, false, false
	}

	// Check if this is a const/function that is re-exported via brace export
	constName := findReexportedName(content)
	if constName != "" && doExportedInHunk(h, idx, constName) {
		return true, false, false
	}

	if isGoExport(content) {
		return true, true, false
	}
	if isPythonExport(content) {
		return true, false, true
	}
	return false, false, false
}

// hasLineAddEquivalent returns true if this LineContext export has a corresponding LineAdd
func hasLineAddEquivalent(h diff.Hunk, idx int) bool {
	for j := idx + 1; j < len(h.Lines); j++ {
		ln := h.Lines[j]
		if ln.Kind == diff.LineAdd && ln.Content == h.Lines[idx].Content {
			return true
		}
	}
	return false
}

func (r SLP089) Check(d *diff.Diff) []Finding {
	var out []Finding

	for _, f := range d.Files {
		if f.IsDelete {
			continue
		}

		fileLower := strings.ToLower(f.Path)

		if strings.Contains(fileLower, "_test.") ||
			strings.Contains(fileLower, "test_") ||
			strings.Contains(fileLower, "/vendor/") ||
			strings.Contains(fileLower, "\\vendor\\") {
			continue
		}

		// Use filepath.Ext for exact extension matching
		ext := strings.ToLower(filepath.Ext(f.Path))
		hasExtension := ext == ".js" || ext == ".ts" || ext == ".go" || ext == ".py"
		if !hasExtension {
			continue
		}

		for _, h := range f.Hunks {
			lastExportLine := -1
			lastExportContent := ""
			lastExportNewLineNo := 0
			lastExportIsGo := false
			lastExportIsPython := false

			for j, ln := range h.Lines {
				// Process LineContext, LineAdd, and LineDelete
				if ln.Kind != diff.LineAdd && ln.Kind != diff.LineContext && ln.Kind != diff.LineDelete {
					continue
				}

				content := strings.TrimSpace(ln.Content)

				// Skip brace exports (re-exports) entirely
				if isBraceExport(content) {
					continue
				}

				isExport, isGo, isPython := isExportLine(content, h, j)

				if isExport {
					if lastExportLine >= 0 && lastExportContent != "" {
						// Use switch statement to determine if this export should be reported
						report := false
						switch h.Lines[lastExportLine].Kind {
						case diff.LineAdd:
							report = true
						case diff.LineContext:
							report = !hasLineAddEquivalent(h, lastExportLine)
						case diff.LineDelete:
							// LineDelete: do not report deletions
							report = false
						}

						if report {
							hasDocs := r.hasDocsBefore(h, lastExportLine, fileLower, lastExportIsGo, lastExportIsPython)

							if !hasDocs {
								out = append(out, Finding{
									RuleID:   r.ID(),
									Severity: r.DefaultSeverity(),
									File:     f.Path,
									Line:     lastExportNewLineNo,
									Message:  "exported function/class missing docstring - add JSDoc comment or description for maintainability",
									Snippet:  lastExportContent,
								})
							}
						}
					}

					lastExportLine = j
					lastExportContent = content
					lastExportNewLineNo = ln.NewLineNo
					lastExportIsGo = isGo
					lastExportIsPython = isPython
				}
			}

			// Handle last export in hunk
			if lastExportLine >= 0 && lastExportContent != "" {
				// Use switch statement to determine if this export should be reported
				report := false
				switch h.Lines[lastExportLine].Kind {
				case diff.LineAdd:
					report = true
				case diff.LineContext:
					report = !hasLineAddEquivalent(h, lastExportLine)
				case diff.LineDelete:
					// LineDelete: do not report deletions
					report = false
				}

				if report {
					hasDocs := r.hasDocsBefore(h, lastExportLine, fileLower, lastExportIsGo, lastExportIsPython)

					if !hasDocs {
						out = append(out, Finding{
							RuleID:   r.ID(),
							Severity: r.DefaultSeverity(),
							File:     f.Path,
							Line:     lastExportNewLineNo,
							Message:  "exported function/class missing docstring - add JSDoc comment or description for maintainability",
							Snippet:  lastExportContent,
						})
					}
				}
			}
		}
	}
	return out
}

func (r SLP089) hasDocsBefore(h diff.Hunk, exportIdx int, filePath string, isGo, isPython bool) bool {
	foundAnyDoc := false

	exportLeadingSpaces := 0
	for _, r := range h.Lines[exportIdx].Content {
		if r == ' ' || r == '\t' {
			exportLeadingSpaces++
		} else {
			break
		}
	}

	for k := exportIdx - 1; k >= 0; k-- {
		prev := h.Lines[k]
		// Only check LineAdd and LineContext - ignore LineDelete (old file content)
		if prev.Kind != diff.LineAdd && prev.Kind != diff.LineContext {
			continue
		}

		content := strings.TrimSpace(prev.Content)

		// Skip empty lines - they separate doc blocks from code
		if content == "" {
			break
		}

		// For Go, skip indented content (body comments, not docstrings)
		// A docstring should have <= indentation of the export
		if isGo || strings.Contains(strings.ToLower(filePath), ".go") {
			// Count leading spaces in original content
			leadingSpaces := 0
			for _, r := range prev.Content {
				if r == ' ' || r == '\t' {
					leadingSpaces++
				} else {
					break
				}
			}
			// Skip if indented more than export (inside function body)
			if leadingSpaces > exportLeadingSpaces {
				continue
			}
		}

		// Check doc patterns - JSDoc style
		if slp089DocPatterns[0].MatchString(content) {
			foundAnyDoc = true
		}
		if foundAnyDoc && slp089DocPatterns[1].MatchString(content) {
			foundAnyDoc = true
		}
		// Check single-line comment/docstring patterns
		if !foundAnyDoc {
			for i := 2; i < len(slp089DocPatterns); i++ {
				if slp089DocPatterns[i].MatchString(content) {
					foundAnyDoc = true
					break
				}
			}
		}

		// Go-specific doc pattern (comments starting with //)
		if isGo || strings.Contains(strings.ToLower(filePath), ".go") {
			if goDocPattern.MatchString(content) {
				foundAnyDoc = true
			}
		}
	}

	return foundAnyDoc
}
