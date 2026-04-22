package rules

import (
	"path/filepath"
	"regexp"
	"strings"

	"github.com/messagesgoel-blip/slopgate/pkg/diff"
)

// SLP069 flags mixed naming conventions (snake_case and CamelCase) in the same package.
type SLP069 struct{}

func (SLP069) ID() string                { return "SLP069" }
func (SLP069) DefaultSeverity() Severity { return SeverityInfo }
func (SLP069) Description() string {
	return "mixed naming conventions (snake_case and CamelCase) in the same package"
}

var snakePattern = regexp.MustCompile(`[a-z]+_[a-z_]+`)
var camelPattern = regexp.MustCompile(`[A-Z][a-zA-Z0-9]+`)

func (r SLP069) Check(d *diff.Diff) []Finding {
	var out []Finding

	type fileInfo struct {
		path              string
		hasSnake          bool
		hasCamel          bool
		firstSnakeLine    int
		firstSnakeSnippet string
	}

	byDir := make(map[string][]*fileInfo)

	for _, f := range d.Files {
		if f.IsDelete || !strings.HasSuffix(f.Path, ".go") || strings.HasSuffix(f.Path, "_test.go") {
			continue
		}
		if len(f.AddedLines()) == 0 {
			continue
		}
		dir := filepath.Dir(f.Path)
		info := &fileInfo{path: f.Path}
		for _, ln := range f.AddedLines() {
			if !info.hasSnake && snakePattern.MatchString(ln.Content) {
				info.hasSnake = true
				info.firstSnakeLine = ln.NewLineNo
				info.firstSnakeSnippet = strings.TrimSpace(ln.Content)
			}
			if !info.hasCamel && camelPattern.MatchString(ln.Content) {
				info.hasCamel = true
			}
			if info.hasSnake && info.hasCamel {
				break
			}
		}
		byDir[dir] = append(byDir[dir], info)
	}

	for _, files := range byDir {
		for _, fi := range files {
			if !fi.hasSnake {
				continue
			}
			otherCamel := false
			for _, other := range files {
				if other.path == fi.path {
					continue
				}
				if other.hasCamel {
					otherCamel = true
					break
				}
			}
			if otherCamel {
				out = append(out, Finding{
					RuleID:   r.ID(),
					Severity: r.DefaultSeverity(),
					File:     fi.path,
					Line:     fi.firstSnakeLine,
					Message:  "mixed naming conventions in package — use Go-standard camelCase/PascalCase",
					Snippet:  fi.firstSnakeSnippet,
				})
			}
		}
	}

	return out
}
