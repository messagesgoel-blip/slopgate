package rules

import (
	"regexp"
	"strings"

	"github.com/messagesgoel-blip/slopgate/pkg/diff"
)

// SLP034 flags potential state management anti-patterns in React components
// and other stateful implementations.
//
// Pattern: Complex state update patterns that could lead to race conditions
// or inconsistent state.
//
// Rationale: Improper state management can lead to race conditions, 
// inconsistent UI states, and difficult-to-debug issues.
type SLP034 struct{}

func (SLP034) ID() string                { return "SLP034" }
func (SLP034) DefaultSeverity() Severity { return SeverityWarn }
func (SLP034) Description() string {
	return "potential state management anti-pattern detected"
}

// slp034AntiPatterns matches problematic state management patterns.
var slp034AntiPatterns = []*regexp.Regexp{
	// Multiple state updates in sequence using stale values
	regexp.MustCompile(`(?s)(set\w+\([^)]*\);\s*){2,}`),
	// Direct mutation of state objects
	regexp.MustCompile(`(?i)(\w+)\[(\w+)\]\s*=\s*`),
	// Async operations without proper dependency arrays in useEffect
	regexp.MustCompile(`(?i)useEffect\s*\(\s*\(\s*\)\s*=>\s*{\s*async\s+function|\(async\s*\(.*\)\s*=>\s*{.*fetch|axios\.get`),
	// State updates based on previous state without functional updates
	regexp.MustCompile(`(?i)(const|let)\s+(\w+)\s*=\s*\w+;\s*\w+\s*=\s*\w+\s*[+\-*/]\s*`),
	// Multiple sequential state updates that should be batched
	regexp.MustCompile(`(?s)set\w+\([^)]*\);\s*set\w+\([^)]*\);\s*set\w+\([^)]*\)`),
	// Missing dependency in useEffect that references state
	regexp.MustCompile(`(?i)useEffect\s*\(\s*\(\s*\)\s*=>\s*{.*}(,\s*\[[^\]]*\])?\s*\)`),
}

// slp034SequentialUpdates matches multiple state updates in sequence.
var slp034SequentialUpdates = regexp.MustCompile(`(?i)set\w+\([^)]*\);\s*set\w+\([^)]*\)`)

func (r SLP034) Check(d *diff.Diff) []Finding {
	var out []Finding
	for _, f := range d.Files {
		if f.IsDelete {
			continue
		}
		
		// Only check JavaScript/TypeScript files
		lowerPath := strings.ToLower(f.Path)
		if !strings.HasSuffix(lowerPath, ".ts") && 
		   !strings.HasSuffix(lowerPath, ".tsx") && 
		   !strings.HasSuffix(lowerPath, ".js") &&
		   !strings.HasSuffix(lowerPath, ".jsx") {
			continue
		}

		for _, h := range f.Hunks {
			// Combine all added lines in the hunk to check for multi-line patterns
			var allAddedContent []string
			for _, ln := range h.Lines {
				if ln.Kind == diff.LineAdd {
					allAddedContent = append(allAddedContent, ln.Content)
				}
			}
			
			fullContent := strings.Join(allAddedContent, "\n")
			
			for _, ln := range h.Lines {
				if ln.Kind != diff.LineAdd {
					continue
				}
				
				content := ln.Content
				// Check for anti-patterns
				for _, pattern := range slp034AntiPatterns {
					if pattern.MatchString(content) {
						out = append(out, Finding{
							RuleID:   r.ID(),
							Severity: r.DefaultSeverity(),
							File:     f.Path,
							Line:     ln.NewLineNo,
							Message:  "potential state management anti-pattern detected - review for race conditions or inefficiencies",
							Snippet:  strings.TrimSpace(content),
						})
						break
					}
				}
				
				// Check for sequential state updates that should be combined
				if slp034SequentialUpdates.MatchString(content) {
					// Look for multiple setState calls in the same function scope
					// This requires more context, so we'll check the hunk for multiple set* calls
					setStateCount := 0
					for _, hLn := range h.Lines {
						if hLn.Kind == diff.LineAdd && regexp.MustCompile(`(?i)set\w+\s*\(`).MatchString(hLn.Content) {
							setStateCount++
						}
					}
					
					if setStateCount >= 2 {
						out = append(out, Finding{
							RuleID:   r.ID(),
							Severity: r.DefaultSeverity(),
							File:     f.Path,
							Line:     ln.NewLineNo,
							Message:  "multiple state updates detected - consider batching or using functional updates",
							Snippet:  strings.TrimSpace(content),
						})
						break // Only report once per hunk to avoid duplicates
					}
				}
			}
			
			// Also check the full hunk content for multi-line patterns
			if len(allAddedContent) > 0 {
				for _, pattern := range slp034AntiPatterns {
					if pattern.MatchString(fullContent) {
						// Report the first line of the hunk if we found a multi-line pattern
						for _, ln := range h.Lines {
							if ln.Kind == diff.LineAdd {
								// Don't add duplicate findings
								duplicate := false
								for _, existing := range out {
									if existing.File == f.Path && existing.Line == ln.NewLineNo {
										duplicate = true
										break
									}
								}
								if !duplicate {
									out = append(out, Finding{
										RuleID:   r.ID(),
										Severity: r.DefaultSeverity(),
										File:     f.Path,
										Line:     ln.NewLineNo,
										Message:  "complex state management pattern detected - review for potential issues",
										Snippet:  strings.TrimSpace(allAddedContent[0]),
									})
								}
								break
							}
						}
						break
					}
				}
			}
		}
	}
	return out
}