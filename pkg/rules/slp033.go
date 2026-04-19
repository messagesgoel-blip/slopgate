package rules

import (
	"regexp"
	"strings"
	"unicode"

	"github.com/messagesgoel-blip/slopgate/pkg/diff"
)

// SLP033 flags missing or improper import statements in TypeScript/JavaScript files.
//
// Pattern: Files using types/functions without proper imports.
//
// Rationale: Missing imports cause runtime errors and type checking failures.
type SLP033 struct{}

func (SLP033) ID() string                { return "SLP033" }
func (SLP033) DefaultSeverity() Severity { return SeverityBlock }
func (SLP033) Description() string {
	return "missing import statement for referenced type/function"
}

// slp033ImportPatterns matches various import patterns.
var slp033ImportPatterns = []*regexp.Regexp{
	// ES6 imports
	regexp.MustCompile(`(?i)import\s+.*\s+from\s+["'][^"']*["']`),
	// Import types
	regexp.MustCompile(`(?i)import\s+type\s+.*\s+from\s+["'][^"']*["']`),
	// Require statements
	regexp.MustCompile(`(?i)const\s+\w+\s+=\s+require\s*\(\s*["'][^"']*["']\s*\)`),
	// Dynamic imports
	regexp.MustCompile(`(?i)import\s*\(\s*["'][^"']*["']\s*\)`),
}

// slp033CommonTypes lists common types that should be imported.
var slp033CommonTypes = []string{
	"React", "Component", "FunctionComponent", "ReactNode", "ReactElement", "ComponentProps",
	"MouseEvent", "KeyboardEvent", "ChangeEvent", "FormEvent",
	"ComponentType", "PropsWithChildren", "Dispatch", "SetStateAction",
	"RefObject", "MutableRefObject", "ForwardedRef",
	"CSSProperties", "HTMLElement", "HTMLAttributes", "DetailedHTMLProps",
}

// slp033ReactHooks lists React hooks that should be imported from React.
var slp033ReactHooks = []string{
	"useState", "useEffect", "useContext", "useReducer", "useCallback",
	"useMemo", "useRef", "useImperativeHandle", "useLayoutEffect", "useDebugValue",
	"useDeferredValue", "useId", "useSyncExternalStore", "useTransition",
}

func (r SLP033) Check(d *diff.Diff) []Finding {
	var out []Finding
	for _, f := range d.Files {
		if f.IsDelete {
			continue
		}
		
		// Only check TypeScript/JavaScript files
		lowerPath := strings.ToLower(f.Path)
		if !strings.HasSuffix(lowerPath, ".ts") && 
		   !strings.HasSuffix(lowerPath, ".tsx") && 
		   !strings.HasSuffix(lowerPath, ".js") &&
		   !strings.HasSuffix(lowerPath, ".jsx") {
			continue
		}

		// Extract ALL imports from the entire file first
		importedItems := make(map[string]bool)
		for _, h := range f.Hunks {
			for _, ln := range h.Lines {
				if ln.Kind == diff.LineAdd {
					content := ln.Content
					// Check if it's an import statement
					if strings.HasPrefix(strings.TrimSpace(strings.ToLower(content)), "import") {
						// Determine which type of import statement this is
						if strings.Contains(content, "{") && strings.Contains(content, "}") {
							// Handle destructured imports like: import { useState, useEffect } from 'react'
							start := strings.Index(content, "{")
							end := strings.Index(content, "}")
							if start != -1 && end != -1 && end > start {
								destructured := content[start+1:end]
								items := strings.Split(destructured, ",")
								for _, item := range items {
									item = strings.TrimSpace(item)
									item = strings.Trim(item, "{}* ")
									if item != "" {
										importedItems[item] = true
									}
								}
							}
						} else if strings.Contains(content, ",") && strings.Contains(content, "from") {
							// Handle mixed imports like: import React, { Component } from 'react'
							parts := strings.Split(content, " from ")
							if len(parts) >= 2 {
								importPart := parts[0]  // Get the part before "from"
								// Remove "import" keyword
								importPart = strings.TrimPrefix(importPart, "import")
								importPart = strings.TrimSpace(importPart)
								
								// Handle cases like "React, { Component }" - split by comma
								subParts := strings.Split(importPart, ",")
								for _, subPart := range subParts {
									subPart = strings.TrimSpace(subPart)
									// If it contains braces, extract from there too
									if strings.Contains(subPart, "{") && strings.Contains(subPart, "}") {
										start := strings.Index(subPart, "{")
										end := strings.Index(subPart, "}")
										if start != -1 && end != -1 && end > start {
											destructured := subPart[start+1:end]
											innerItems := strings.Split(destructured, ",")
											for _, item := range innerItems {
												item = strings.TrimSpace(item)
												item = strings.Trim(item, "{}* ")
												if item != "" {
													importedItems[item] = true
												}
											}
										}
									} else {
										// Handle default/named import (not in braces)
										subPart = strings.Trim(subPart, "{}* ")
										if subPart != "" {
											importedItems[subPart] = true
										}
									}
								}
							}
						} else if strings.Contains(content, " from ") {
							// Handle default imports like: import React from 'react'
							parts := strings.Split(content, " from ")
							if len(parts) >= 2 {
								importPart := parts[0]  // Get the part before "from"
								// Remove "import" keyword
								importPart = strings.TrimPrefix(importPart, "import")
								importPart = strings.TrimSpace(importPart)
								
								// Extract just the default import name (before " from ")
								defaultImport := strings.TrimSpace(importPart)
								if defaultImport != "" && !strings.Contains(defaultImport, "{") {
									// Remove any trailing commas if present
									defaultImport = strings.TrimRight(defaultImport, ", ")
									if defaultImport != "" {
										importedItems[defaultImport] = true
									}
								}
							}
						}
					}
				}
			}
		}

		// Now check for usage of common types/hooks without imports across all hunks
		for _, h := range f.Hunks {
			for _, ln := range h.Lines {
				if ln.Kind != diff.LineAdd {
					continue
				}
				
				content := ln.Content
				
				// Check for React hooks usage without import
				for _, hook := range slp033ReactHooks {
					// Use word boundary checking to avoid partial matches like "useState" in "usePreviousState"
					if containsWholeWord(content, hook) && !importedItems[hook] {
						out = append(out, Finding{
							RuleID:   r.ID(),
							Severity: r.DefaultSeverity(),
							File:     f.Path,
							Line:     ln.NewLineNo,
							Message:  "React hook " + hook + " used without import - add import { " + hook + " } from 'react'",
							Snippet:  strings.TrimSpace(content),
						})
						break
					}
				}
				
				// Check for common types usage without import
				for _, typ := range slp033CommonTypes {
					// Use word boundary checking to avoid partial matches like "Component" in "ComponentProps"
					if containsWholeWord(content, typ) && !importedItems[typ] {
						// Only flag if it's used in a type context (not just as a variable name)
						if isTypeContext(content, typ) {
							out = append(out, Finding{
								RuleID:   r.ID(),
								Severity: r.DefaultSeverity(),
								File:     f.Path,
								Line:     ln.NewLineNo,
								Message:  "Type " + typ + " used without import - add import { " + typ + " } from 'react'",
								Snippet:  strings.TrimSpace(content),
							})
							break
						}
					}
				}
			}
		}
	}
	return out
}

// isTypeContext checks if a type name appears in a type annotation context
func isTypeContext(content, typeName string) bool {
	// Look for patterns like ": TypeName", "as TypeName", "TypeName<", etc.
	typePatterns := []string{
		":" + typeName,
		": " + typeName,
		"as " + typeName,
		typeName + "<",
		"extends " + typeName,
		"type " + typeName,
		"interface " + typeName,
	}
	
	contentLower := strings.ToLower(content)
	typeNameLower := strings.ToLower(typeName)
	
	for _, pattern := range typePatterns {
		patternLower := strings.ToLower(pattern)
		if strings.Contains(contentLower, patternLower) {
			return true
		}
	}
	
	// Check if the type appears after "extends" or "implements"
	if strings.Contains(contentLower, "extends") || strings.Contains(contentLower, "implements") {
		words := strings.Fields(contentLower)
		for i, word := range words {
			if word == "extends" || word == "implements" {
				if i+1 < len(words) && words[i+1] == typeNameLower {
					return true
				}
			}
		}
	}
	
	return false
}

// containsWholeWord checks if the needle appears as a whole word in the haystack
func containsWholeWord(haystack, needle string) bool {
	// This avoids matching "Component" inside "ComponentProps"
	haystackLower := strings.ToLower(haystack)
	needleLower := strings.ToLower(needle)
	
	// Split the haystack by word boundaries and check if needle matches any of the parts
	// Word boundaries are defined as non-alphanumeric characters
	parts := strings.FieldsFunc(haystackLower, func(r rune) bool {
		return !unicode.IsLetter(r) && !unicode.IsDigit(r) && r != '_'
	})
	
	for _, part := range parts {
		if part == needleLower {
			return true
		}
	}
	
	return false
}