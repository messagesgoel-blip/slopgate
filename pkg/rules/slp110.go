package rules

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/messagesgoel-blip/slopgate/pkg/diff"
)

// SLP110 flags new files added in the same diff that have highly similar
// import structures, suggesting copy-paste file duplication.
type SLP110 struct{}

func (SLP110) ID() string                { return "SLP110" }
func (SLP110) DefaultSeverity() Severity { return SeverityWarn }
func (SLP110) Description() string {
	return "new file appears duplicated from existing file — verify changes are intentional"
}

var slp110ImportLike = []string{
	"\"", "'",
	"require(", "from ", "#include",
	"namespace ", "module ",
}

func slp110ExtractImports(file diff.File) []string {
	var imports []string
	for _, ln := range file.AddedLines() {
		content := strings.TrimSpace(ln.Content)
		for _, pat := range slp110ImportLike {
			if strings.Contains(content, pat) {
				imports = append(imports, strings.ToLower(strings.TrimSpace(content)))
				break
			}
		}
	}
	return imports
}

func slp110Jaccard(a, b []string) float64 {
	if len(a) == 0 && len(b) == 0 {
		return 0
	}
	setA := make(map[string]bool)
	for _, s := range a {
		setA[s] = true
	}
	intersection := 0
	seen := make(map[string]bool)
	for _, s := range b {
		if setA[s] && !seen[s] {
			intersection++
			seen[s] = true
		}
	}
	union := len(setA)
	for _, s := range b {
		if !setA[s] {
			union++
			setA[s] = true
		}
	}
	if union == 0 {
		return 0
	}
	return float64(intersection) / float64(union)
}

func (r SLP110) Check(d *diff.Diff) []Finding {
	var out []Finding

	type fileIdx struct {
		file    diff.File
		imports []string
	}
	dirFiles := make(map[string][]fileIdx)

	for _, f := range d.Files {
		if f.IsDelete || isDocFile(f.Path) {
			continue
		}
		dir := filepath.Dir(f.Path)
		imports := slp110ExtractImports(f)
		if len(imports) == 0 {
			continue
		}
		dirFiles[dir] = append(dirFiles[dir], fileIdx{f, imports})
	}

	for dir, files := range dirFiles {
		if len(files) < 2 {
			continue
		}
		for i := 0; i < len(files); i++ {
			for j := i + 1; j < len(files); j++ {
				similarity := slp110Jaccard(files[i].imports, files[j].imports)
				if similarity > 0.6 {
					for _, ln := range files[j].file.AddedLines() {
						trimmed := strings.TrimSpace(ln.Content)
						if strings.HasPrefix(trimmed, "package ") || strings.HasPrefix(trimmed, "import (") {
							out = append(out, Finding{
								RuleID:   r.ID(),
								Severity: r.DefaultSeverity(),
								File:     files[j].file.Path,
								Line:     ln.NewLineNo,
								Message:  fmt.Sprintf("file %.0f%% similar imports to %s in %s — possible copy-paste duplication", similarity*100, files[i].file.Path, dir),
								Snippet:  trimmed,
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
