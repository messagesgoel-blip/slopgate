package rules

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/messagesgoel-blip/slopgate/pkg/diff"
)

// SLP204 flags code paths where an error variable is assigned from a
// function call but the enclosing function returns success without
// checking or propagating that error.
//
// Primary pattern (high signal): an added line assigns an error (err := ...,
// const err = ..., let err = ...) and a subsequent added line returns a
// success value (nil, true, None, null, { ok: true }) without checking the error
// in between.
//
// Languages: Go, Python, Java, JS/TS.
// Scope: diff only — scans added lines within each file hunk.
type SLP204 struct{}

// ID returns the rule identifier: "SLP204".
func (SLP204) ID() string { return "SLP204" }

// DefaultSeverity returns this rule's default severity.
func (SLP204) DefaultSeverity() Severity { return SeverityBlock }

// Description returns a short description of the SLP204 rule.
func (SLP204) Description() string {
	return "error variable assigned but never checked before returning success"
}

// ---------------------------------------------------------------------------
// Regex library
// ---------------------------------------------------------------------------

// errAssignPattern matches when an error variable (name starting with "err")
// is assigned from a function call or await expression.
// Requires a trailing "(" on the RHS to ensure only function/method calls are
// matched, not plain variable assignments like "err := someVar".
var assignPattern = regexp.MustCompile(
	`(?i)(?:const|let|var)?\s*(\berr\w*\b)\s*(?::=|=)\s*(?:await\s+)?\b\w[\w.]*\s*\(`)

// successReturnPattern matches return statements returning a success
// value that would mask an unchecked error.
var successReturnPattern = regexp.MustCompile(
	`(?i)\breturn\s+(nil|true|null|None)\b`)

// inlineErrCheckPattern matches inline error checks on the same line
// as the assignment (e.g., Go if-initialization).
var inlineErrCheckPattern = regexp.MustCompile(
	`(?i)(?:if|switch)\s*\(?.*\berr\w*\b.*\{`)

// reOkSuccess matches JS/TS object returns where ok is truthy.
var reOkSuccess = regexp.MustCompile(
	`\bok:\s*(true|1)\b|['"]ok['"]:\s*(true|1)`)

// reSuccessSuccess matches JS/TS object returns where success is truthy.
var reSuccessSuccess = regexp.MustCompile(
	`\bsuccess:\s*(true|1)\b|['"]success['"]:\s*(true|1)`)

// reNilAssign matches re-assignment of an error variable to nil.
var reNilAssign = regexp.MustCompile(`\berr\w*\s*=\s*nil\b`)

// reNullAssign matches re-assignment of an error variable to null.
var reNullAssign = regexp.MustCompile(`\berr\w*\s*=\s*null\b`)

// ---------------------------------------------------------------------------
// Check
// ---------------------------------------------------------------------------

// Check implements the diff-aware SLP204 rule for unchecked errors before success returns.
func (r SLP204) Check(d *diff.Diff) []Finding {
	var out []Finding

	if d == nil {
		return out
	}

	for _, f := range d.Files {
		if f.IsDelete {
			continue
		}

		idx := strings.LastIndex(f.Path, ".")
		if idx < 0 {
			continue
		}
		ext := strings.ToLower(f.Path[idx:])
		if !slp204SupportedExt(ext) {
			continue
		}

		if isTestFile(f.Path) {
			continue
		}

		for _, h := range f.Hunks {
			// Track error variables found in added lines.
			// errName -> line number of assignment.
			pending := map[string]int{}
			// Cache compiled per-errName regexes for isErrChecked.
			errCheckCache := map[string][]*regexp.Regexp{}

			for _, ln := range h.Lines {
				if ln.Kind != diff.LineAdd {
					continue
				}
				content := strings.TrimSpace(ln.Content)

				if isSlp204Skippable(content) {
					continue
				}

				// Check for inline error checks (e.g., Go if-initialization).
				if inlineErrCheckPattern.MatchString(content) {
					// Still clear pending errors that are checked on this
					// line (e.g. `if (err)` in JS, `if (err != null)` in
					// Java) so that later success returns don't false-flag
					// them.
					slp204ClearCheckedErrors(content, pending, errCheckCache)
					continue
				}

				// Check for proper error checks and remove cleared errors from pending.
				cleared := slp204ClearCheckedErrors(content, pending, errCheckCache)
				if cleared > 0 {
					continue
				}

				// Check for success return while there are pending unchecked errors.
				if len(pending) > 0 && isSuccessReturn(content) {
					for errName, lineNo := range pending {
						out = append(out, Finding{
							RuleID:   r.ID(),
							Severity: r.DefaultSeverity(),
							File:     f.Path,
							Line:     lineNo,
							Message:  fmt.Sprintf("error %q is never checked before returning success", errName),
							Snippet:  content,
						})
					}
					// Clear pending after flagging to avoid duplicate findings per return.
					pending = map[string]int{}
					continue
				}

				// Check for error assignment.
				if m := assignPattern.FindStringSubmatch(content); len(m) > 1 {
					varName := m[1]
					if !isErrNameBlacklisted(varName, content) {
						pending[varName] = ln.NewLineNo
					}
				}
			}
		}
	}

	return out
}

// slp204ClearCheckedErrors removes error variables from pending that are
// properly checked on this line. Returns the number of cleared errors.
func slp204ClearCheckedErrors(content string, pending map[string]int, cache map[string][]*regexp.Regexp) int {
	cleared := 0
	for en := range pending {
		if isErrChecked(en, content, cache) {
			delete(pending, en)
			cleared++
		}
	}
	return cleared
}

// isErrChecked returns true if the line shows the error variable is properly
// checked/handled. Compiled regexes are cached per errName.
func isErrChecked(errName, content string, cache map[string][]*regexp.Regexp) bool {
	reList, ok := cache[errName]
	if !ok {
		reList = []*regexp.Regexp{
			regexp.MustCompile(fmt.Sprintf(`\b%s\s*[!=]==?\s*nil`, errName)),      // Go nil check
			regexp.MustCompile(fmt.Sprintf(`\b%s\s+is\s+(not\s+)?None`, errName)), // Python None
			regexp.MustCompile(fmt.Sprintf(`\bif\s*\(?\s*!?%s\s*\)?`, errName)),   // Python/JS truthy
			regexp.MustCompile(fmt.Sprintf(`\breturn\s+%s\b`, errName)),           // return propagation
			regexp.MustCompile(fmt.Sprintf(`\b%s\s*!=\s*null`, errName)),          // Java null
			regexp.MustCompile(fmt.Sprintf(`\b(?:raise|throw)\s+%s\b`, errName)),  // raise/throw
			regexp.MustCompile(fmt.Sprintf(`\bswitch\s+%s\s*\{`, errName)),        // Go switch on err
		}
		cache[errName] = reList
	}
	for _, re := range reList {
		if re.MatchString(content) {
			return true
		}
	}
	return false
}

// isSuccessReturn returns true if the line returns a success value that would
// mask an unchecked error.
func isSuccessReturn(content string) bool {
	if content == "" {
		return false
	}
	// Strip string/char literals so "return true" inside string
	// content (e.g. log messages, SQL) doesn't cause a false positive.
	clean := stripStringLiterals(content)

	// Simple success values: nil, true, null, None
	// Reject multi-value returns like "return nil, err" where a comma
	// follows the value — those are error-propagating, not success returns.
	if m := successReturnPattern.FindStringSubmatchIndex(clean); m != nil && len(m) > 2 {
		return !strings.HasPrefix(content[m[1]:], ",")
	}
	// JS/TS object returns: return { ok: true } or return { success: true }
	// Require truthy values — "success: false" / "ok: false" are NOT success returns.
	lower := strings.ToLower(clean)
	if strings.Contains(lower, "return") && strings.Contains(lower, "{") {
		// Match "ok: true/1" or "\"ok\"/\"'ok'": true/1
		if reOkSuccess.MatchString(lower) {
			return true
		}
		// Match "success: true/1" or "\"success\"": true/1
		if reSuccessSuccess.MatchString(lower) {
			return true
		}
	}
	return false
}

// slp204SupportedExt returns true for languages SLP204 covers.
func slp204SupportedExt(ext string) bool {
	switch ext {
	case ".go", ".js", ".ts", ".tsx", ".jsx", ".py", ".java", ".kt":
		return true
	}
	return false
}

// isSlp204Skippable returns true for lines that should never be flagged.
func isSlp204Skippable(content string) bool {
	trimmed := strings.TrimLeft(content, " \t")
	for _, pat := range slp203SkipLinePatterns {
		if pat.MatchString(trimmed) {
			return true
		}
	}
	return false
}

// isErrNameBlacklisted excludes assignments that are not actual error
// captures — e.g., re-assigning inside an existing guard block.
func isErrNameBlacklisted(varName, content string) bool {
	// Skip if this is a reassignment to nil/null inside a guard.
	// e.g. } else { err = nil } — not a new error capture.
	clean := stripStringLiterals(content)
	if reNilAssign.MatchString(clean) || reNullAssign.MatchString(clean) {
		return true
	}
	return false
}
