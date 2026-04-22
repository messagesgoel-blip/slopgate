package rules

import (
	"strings"

	"github.com/messagesgoel-blip/slopgate/pkg/diff"
)

// SLP066 flags concurrent map access without mutex protection in Go files.
type SLP066 struct{}

func (SLP066) ID() string                { return "SLP066" }
func (SLP066) DefaultSeverity() Severity { return SeverityBlock }
func (SLP066) Description() string {
	return "concurrent map access without mutex protection"
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
			if strings.Contains(ln.Content, "map[") {
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
