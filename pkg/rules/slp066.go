package rules

import (
	"strings"

	"github.com/messagesgoel-blip/slopgate/pkg/diff"
)

// SLP066 flags concurrent map access without mutex protection in Go files.
//
// Heuristic: if the diff contains goroutines or WaitGroup usage and also
// contains map index/read/write operations, flag unless a sync.Mutex /
// sync.RWMutex / sync.Map is also present. This is intentionally coarse —
// precisely matching mutex guards to specific map variables requires full
// AST analysis which is out of scope for diff-based linting.
type SLP066 struct{}

func (SLP066) ID() string                { return "SLP066" }
func (SLP066) DefaultSeverity() Severity { return SeverityBlock }
func (SLP066) Description() string {
	return "concurrent map access without mutex protection"
}

// hasMapIndexOp detects map read/write operations like m[key] or m[key] = value.
func hasMapIndexOp(line string) bool {
	// Simple heuristic: line contains both '[' and ']' and also "map[" or an
	// identifier followed by '[' (e.g., "m[key]").
	if !strings.Contains(line, "[") || !strings.Contains(line, "]") {
		return false
	}
	// Explicit map literal or index op.
	if strings.Contains(line, "map[") {
		return true
	}
	// Look for identifier[
	for i := 0; i+1 < len(line); i++ {
		if line[i+1] == '[' {
			c := line[i]
			if (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || c == '_' {
				return true
			}
		}
	}
	return false
}

func (r SLP066) Check(d *diff.Diff) []Finding {
	var out []Finding
	for _, f := range d.Files {
		if f.IsDelete || !strings.HasSuffix(f.Path, ".go") {
			continue
		}
		added := f.AddedLines()
		var hasConcurrent, hasMutex bool
		for _, ln := range added {
			if strings.Contains(ln.Content, "go ") || strings.Contains(ln.Content, "sync.WaitGroup") {
				hasConcurrent = true
			}
			if strings.Contains(ln.Content, "sync.Mutex") ||
				strings.Contains(ln.Content, "sync.RWMutex") ||
				strings.Contains(ln.Content, "sync.Map") {
				hasMutex = true
			}
		}
		if !hasConcurrent || hasMutex {
			continue
		}
		for _, ln := range added {
			if hasMapIndexOp(ln.Content) {
				out = append(out, Finding{
					RuleID:   r.ID(),
					Severity: r.DefaultSeverity(),
					File:     f.Path,
					Line:     ln.NewLineNo,
					Message:  "map accessed concurrently without mutex — add sync.Mutex or use sync.Map",
					Snippet:  strings.TrimSpace(ln.Content),
				})
			}
		}
	}
	return out
}
