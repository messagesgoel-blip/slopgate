package rules

import (
	"path"
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

var knownExtensionless = map[string]bool{
	"Makefile": true, "Dockerfile": true, "LICENSE": true, "README": true,
	"CHANGELOG": true, "CONTRIBUTORS": true, "NOTICE": true, "AUTHORS": true,
	"Vagrantfile": true, "Procfile": true, "Rakefile": true, "Gemfile": true,
	"Jenkinsfile": true, "VERSION": true, "go.mod": false, "go.sum": false,
}

func (r SLP111) Check(d *diff.Diff) []Finding {
	var out []Finding

	for _, f := range d.Files {
		if f.IsDelete || isDocFile(f.Path) {
			continue
		}

		ext := ""
		if pe := path.Ext(f.Path); pe != "" {
			ext = strings.ToLower(pe)
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
			base := path.Base(f.Path)
			if !knownExtensionless[base] {
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
