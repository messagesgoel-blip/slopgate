package rules

import (
	"strings"

	"github.com/messagesgoel-blip/slopgate/pkg/diff"
)

// SLP112 flags generated files committed without their corresponding source
// files. This catches common AI slop patterns like committing .pb.go, .min.js,
// or _generated.ts files without the .proto, .js source in the same commit.
type SLP112 struct{}

func (SLP112) ID() string                { return "SLP112" }
func (SLP112) DefaultSeverity() Severity { return SeverityWarn }
func (SLP112) Description() string {
	return "generated file committed without corresponding source — commit the source file too"
}

var slp112SuffixMappings = []struct {
	generated string
	source    string
}{
	{generated: ".pb.go", source: ".proto"},
	{generated: ".pb.gw.go", source: ".proto"},
	{generated: "_generated.go", source: ".proto"},
	{generated: "_generated.ts", source: ".proto"},
	{generated: "_generated.js", source: ".proto"},
	{generated: ".min.js", source: ".js"},
	{generated: ".min.css", source: ".css"},
	{generated: ".grpc.go", source: ".proto"},
	{generated: ".pb.d.ts", source: ".proto"},
	{generated: ".pb.js", source: ".proto"},
}

func (r SLP112) Check(d *diff.Diff) []Finding {
	var out []Finding
	allFiles := make(map[string]bool)
	for _, f := range d.Files {
		if !f.IsDelete {
			allFiles[f.Path] = true
		}
	}

	for _, f := range d.Files {
		if f.IsDelete || isDocFile(f.Path) {
			continue
		}

		for _, mapping := range slp112SuffixMappings {
			if strings.HasSuffix(f.Path, mapping.generated) {
				base := strings.TrimSuffix(f.Path, mapping.generated)
				sourceFile := base + mapping.source
				if !allFiles[sourceFile] {
					out = append(out, Finding{
						RuleID:   r.ID(),
						Severity: r.DefaultSeverity(),
						File:     f.Path,
						Line:     0,
						Message:  "generated file '" + f.Path + "' committed without source '" + sourceFile + "' — commit both together",
						Snippet:  f.Path,
					})
				}
				break
			}
		}
	}
	return out
}
