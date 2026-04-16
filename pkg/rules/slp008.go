package rules

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/messagesgoel-blip/slopgate/pkg/diff"
)

// SLP008 flags error handlers that log the error but then silently return
// without recovery — the error is acknowledged but never acted on.
//
// Rationale: AI code generators frequently produce error-handling blocks
// that log.Printf/slog.Error the error and then return nil (or bare return).
// This swallows the error: the caller has no idea anything went wrong, and
// no recovery has been attempted. The correct pattern is either to return
// the error to the caller (so they can decide what to do) or to handle it
// locally. Logging-and-returning-nil is the worst of both worlds.
type SLP008 struct{}

func (SLP008) ID() string                { return "SLP008" }
func (SLP008) DefaultSeverity() Severity { return SeverityWarn }
func (SLP008) Description() string {
	return "error logged but silently returned without recovery"
}

// slp008LogPatterns matches logging calls that acknowledge an error.
var slp008LogPatterns = []*regexp.Regexp{
	// Go: log.Printf, log.Println, log.Print, slog.Error, slog.Warn, slog.Info
	regexp.MustCompile(`\blog\.(Printf|Println|Print|Fatalf|Panicf)\s*\(`),
	regexp.MustCompile(`\bslog\.(Error|Warn|Info)\s*\(`),
	// JS/TS: console.error, console.warn
	regexp.MustCompile(`\bconsole\.(error|warn)\s*\(`),
	// Python: logging.error, logging.warning, logger.error, log.error, etc.
	regexp.MustCompile(`\b(logging|logger|log)\.(error|warning|critical|exception)\s*\(`),
}

// slp008SilentReturnPatterns matches return statements that silently
// swallow the error — returning nil, nothing, undefined, or None.
var slp008SilentReturnPatterns = []*regexp.Regexp{
	// Go: return nil, bare return (return)
	regexp.MustCompile(`\breturn\s+nil\b`),
	regexp.MustCompile(`^\s*return\s*$`),
	// Go: return nil, err is NOT silent (returns the error) — handled below.
	// Go: multiple return with nil first: return nil, nil, etc.
	regexp.MustCompile(`\breturn\s+nil\s*,`),
	// JS/TS: return, return undefined, return null
	regexp.MustCompile(`\breturn\s+(undefined|null)\b`),
	// Python: return None, bare return
	regexp.MustCompile(`\breturn\s+None\b`),
	regexp.MustCompile(`^\s*return\s*$`),
}

// slp008ErrorReturnPattern detects a return that actually propagates an
// error value back to the caller — this is NOT a finding.
var slp008ErrorReturnPatterns = []*regexp.Regexp{
	// Go: return err, return fmt.Errorf(...), return errors.Wrap/New(...)
	// return <anything>, err  (multi-return where last is error)
	regexp.MustCompile(`\breturn\s+.*\berr\b`),
	regexp.MustCompile(`\breturn\s+fmt\.Errorf\s*\(`),
	regexp.MustCompile(`\breturn\s+errors\.(Wrap|Wrapf|New)\s*\(`),
	// Go: return ..., err (multi-return propagating error)
	regexp.MustCompile(`\breturn\s+.*,\s*err\b`),
	// JS/TS: return err, throw err, return new Error
	regexp.MustCompile(`\breturn\s+err\b`),
	regexp.MustCompile(`\bthrow\b`),
	// Python: raise, return err
	regexp.MustCompile(`\braise\b`),
}

// isSilentReturn checks whether a return statement silently swallows
// the error (returns nil/nothing) vs propagating it.
func isSilentReturn(line string) bool {
	trimmed := strings.TrimSpace(line)
	if trimmed == "" {
		return false
	}

	// First, check if this is a return that propagates an error.
	// If so, it is NOT a silent return.
	for _, p := range slp008ErrorReturnPatterns {
		if p.MatchString(trimmed) {
			return false
		}
	}

	// Check if it's a silent return pattern.
	for _, p := range slp008SilentReturnPatterns {
		if p.MatchString(trimmed) {
			return true
		}
	}
	return false
}

// isLoggingCall checks whether the line contains an error-logging call.
func isLoggingCall(line string) bool {
	stripped := stripCommentAndStrings(line)
	if stripped == "" {
		return false
	}
	for _, p := range slp008LogPatterns {
		if p.MatchString(stripped) {
			return true
		}
	}
	return false
}

func (r SLP008) Check(d *diff.Diff) []Finding {
	var out []Finding
	for _, f := range d.Files {
		if f.IsDelete || isSuppressedDebugFile(f.Path) {
			continue
		}
		lines := f.AddedLines()
		for i, ln := range lines {
			if !isLoggingCall(ln.Content) {
				continue
			}
			// Look for a silent return on the same line or the next added line.
			// Same line: e.g.  log.Printf(...); return nil
			if sameLineReturn(ln.Content) {
				out = append(out, Finding{
					RuleID:   r.ID(),
					Severity: r.DefaultSeverity(),
					File:     f.Path,
					Line:     ln.NewLineNo,
					Message:  fmt.Sprintf("error logged then silently returned — propagate the error or handle it"),
					Snippet:  strings.TrimSpace(ln.Content),
				})
				continue
			}
			// Next added line: must be the immediately following added line
			// with no substantive code in between.
			if i+1 < len(lines) {
				next := lines[i+1]
				// The next line must be adjacent in the new file (off by 1)
				// to ensure no intervening lines from context/delete were skipped.
				if next.NewLineNo == ln.NewLineNo+1 && isSilentReturn(next.Content) {
					out = append(out, Finding{
						RuleID:   r.ID(),
						Severity: r.DefaultSeverity(),
						File:     f.Path,
						Line:     next.NewLineNo,
						Message:  fmt.Sprintf("error logged then silently returned — propagate the error or handle it"),
						Snippet:  strings.TrimSpace(next.Content),
					})
				}
			}
		}
	}
	return out
}

// sameLineReturn checks whether a line that contains a logging call also
// contains a silent return on the same line (e.g. after a semicolon).
func sameLineReturn(line string) bool {
	// Split on semicolons to check for a return after the log call.
	parts := strings.Split(line, ";")
	if len(parts) < 2 {
		return false
	}
	// Check all parts after the first for a silent return.
	for _, part := range parts[1:] {
		if isSilentReturn(part) {
			return true
		}
	}
	return false
}
