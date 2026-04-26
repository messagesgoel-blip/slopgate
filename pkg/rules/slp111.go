package rules

import (
	"strings"

	"github.com/messagesgoel-blip/slopgate/pkg/diff"
)

// SLP111 flags binary or executable files committed to the repository.
// This catches a common AI slop pattern where agents commit compiled
// outputs, binaries, or object files.
type SLP111 struct{}

func (SLP111) ID() string                { return "SLP111" }
func (SLP111) DefaultSeverity() Severity { return SeverityBlock }
func (SLP111) Description() string {
	return "binary file committed — add to .gitignore and remove from tracking"
}

var slp111BinaryExtensions = map[string]bool{
	".exe": true, ".dll": true, ".so": true, ".dylib": true,
	".bin": true, ".wasm": true, ".o": true, ".a": true,
	".class": true, ".pyc": true, ".pyo": true,
	".jar": true, ".war": true, ".ear": true,
	".apk": true, ".ipa": true,
	".zip": true, ".tar": true, ".gz": true, ".bz2": true,
	".7z": true, ".rar": true, ".pdb": true, ".ds_store": true,
}

var slp111SourceExtensions = map[string]bool{
	".go": true, ".py": true, ".js": true, ".ts": true,
	".jsx": true, ".tsx": true, ".java": true, ".kt": true,
	".rs": true, ".c": true, ".cpp": true, ".cc": true,
	".cxx": true, ".h": true, ".hpp": true, ".rb": true,
	".php": true, ".swift": true, ".scala": true, ".cs": true,
	".fs": true, ".fsx": true, ".elm": true, ".hs": true,
	".clj": true, ".cljs": true, ".ex": true, ".exs": true,
	".erl": true,
}

func (r SLP111) Check(d *diff.Diff) []Finding {
	var out []Finding

	for _, f := range d.Files {
		if f.IsDelete || isDocFile(f.Path) {
			continue
		}

		ext := strings.ToLower(f.Path)
		if dot := strings.LastIndex(ext, "."); dot >= 0 {
			ext = ext[dot:]
		} else {
			ext = ""
		}

		if slp111BinaryExtensions[ext] {
			out = append(out, Finding{
				RuleID:   r.ID(),
				Severity: r.DefaultSeverity(),
				File:     f.Path,
				Line:     0,
				Message:  "binary file '" + f.Path + "' committed — add to .gitignore and git rm --cached",
				Snippet:  f.Path,
			})
			continue
		}

		if f.IsNew && ext == "" {
			knownNonSource := map[string]bool{
				".md": true, ".txt": true, ".json": true, ".yaml": true, ".yml": true,
				".toml": true, ".xml": true, ".csv": true, ".svg": true, ".html": true,
				".css": true, ".scss": true, ".less": true, ".graphql": true, ".proto": true,
				".sql": true, ".sh": true, ".bash": true, ".Makefile": true,
			}
			if !knownNonSource[ext] && !strings.Contains(f.Path, ".") {
				out = append(out, Finding{
					RuleID:   r.ID(),
					Severity: r.DefaultSeverity(),
					File:     f.Path,
					Line:     0,
					Message:  "extensionless file committed — may be a binary. Verify and add to .gitignore if built",
					Snippet:  f.Path,
				})
			}
		}
	}
	return out
}
