package rules

import (
	"go/scanner"
	"go/token"
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

var snakePattern = regexp.MustCompile(`^[a-z]+_[a-z_]+$`)
var camelPattern = regexp.MustCompile(`^[A-Z][a-zA-Z0-9]+$`)

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
			// Use Go scanner to extract identifiers only, skipping strings/comments.
			fs := token.NewFileSet()
			file := fs.AddFile(f.Path, fs.Base(), len(ln.Content))
			var s scanner.Scanner
			s.Init(file, []byte(ln.Content), nil, scanner.ScanComments)
			for {
				_, tok, lit := s.Scan()
				if tok == token.EOF {
					break
				}
				if tok != token.IDENT {
					continue
				}
				if !info.hasSnake && snakePattern.MatchString(lit) {
					info.hasSnake = true
					info.firstSnakeLine = ln.NewLineNo
					info.firstSnakeSnippet = strings.TrimSpace(ln.Content)
				}
				if !info.hasCamel && camelPattern.MatchString(lit) {
					info.hasCamel = true
				}
				if info.hasSnake && info.hasCamel {
					break
				}
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
			// Check both other files and the same file for CamelCase.
			for _, other := range files {
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
