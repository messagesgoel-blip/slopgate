package rules

import (
	"strings"

	"github.com/messagesgoel-blip/slopgate/pkg/diff"
)

// SLP112 flags generated files committed without their corresponding source
// files. This catches common AI slop patterns like committing .pb.go, .min.js,
// or .generated.ts files without the .proto, .tsx source in the same commit.
type SLP112 struct{}

func (SLP112) ID() string                { return "SLP112" }
func (SLP112) DefaultSeverity() Severity { return SeverityWarn }
func (SLP112) Description() string {
	return "generated file committed without corresponding source — commit the source file too"
}

var slp112GeneratedSuffixes = []string{
	".pb.go", ".pb.gw.go", "_generated.go", "_generated.ts", "_generated.js",
	".min.js", ".min.css", ".grpc.go",
	".pb.d.ts", ".pb.js",
}

var slp112KnownSourceMappings = map[string]string{
	".pb.go":        ".proto",
	".pb.gw.go":     ".proto",
	".grpc.go":      ".proto",
	"_generated.go": ".proto",
	"_generated.ts": ".proto",
	"_generated.js": ".proto",
	".min.js":       ".js",
	".min.css":      ".css",
	".pb.d.ts":      ".proto",
	".pb.js":        ".proto",
}

func (r SLP112) Check(d *diff.Diff) []Finding {
	var out []Finding
	allFiles := make(map[string]bool)
	for _, f := range d.Files {
		allFiles[f.Path] = true
	}

	for _, f := range d.Files {
		if f.IsDelete || isDocFile(f.Path) || !f.IsNew {
			continue
		}

		for _, suffix := range slp112GeneratedSuffixes {
			if strings.HasSuffix(f.Path, suffix) {
				sourceExt, ok := slp112KnownSourceMappings[suffix]
				if !ok {
					continue
				}
				base := strings.TrimSuffix(f.Path, suffix)
				sourceFile := base + sourceExt
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
