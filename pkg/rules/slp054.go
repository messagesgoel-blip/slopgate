package rules

import (
	"path/filepath"
	"strings"

	"github.com/messagesgoel-blip/slopgate/pkg/diff"
)

// SLP054 flags Go files whose package declaration does not match the
// containing directory name, with exceptions for _test packages and
// package main in cmd/ directories.
type SLP054 struct{}

func (SLP054) ID() string                { return "SLP054" }
func (SLP054) DefaultSeverity() Severity { return SeverityWarn }
func (SLP054) Description() string {
	return "package declaration does not match directory name"
}

func (r SLP054) Check(d *diff.Diff) []Finding {
	var out []Finding
	for _, f := range d.Files {
		if f.IsDelete || !isGoFile(f.Path) {
			continue
		}
		dir := filepath.Base(filepath.Dir(f.Path))
		for _, ln := range f.AddedLines() {
			content := strings.TrimSpace(ln.Content)
			if !strings.HasPrefix(content, "package ") {
				continue
			}
			pkg := strings.TrimSpace(strings.TrimPrefix(content, "package "))
			// Strip _test suffix for test files.
			expected := dir
			if strings.HasSuffix(f.Path, "_test.go") {
				// package foo_test and package foo are both valid in foo_test.go.
				if pkg == expected+"_test" || pkg == expected {
					continue
				}
			}
			// package main is valid in cmd/ directories regardless of dir name.
			if pkg == "main" && strings.Contains(filepath.Dir(f.Path), "cmd") {
				continue
			}
			if pkg != expected {
				out = append(out, Finding{
					RuleID:   r.ID(),
					Severity: r.DefaultSeverity(),
					File:     f.Path,
					Line:     ln.NewLineNo,
					Message:  "package " + pkg + " does not match directory " + dir + " — rename package or move file",
					Snippet:  content,
				})
			}
		}
	}
	return out
}
