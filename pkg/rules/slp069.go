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
var pascalPattern = regexp.MustCompile(`^[A-Z][a-zA-Z0-9]*$`)
var lowerCamelPattern = regexp.MustCompile(`^[a-z]+(?:[A-Z][a-zA-Z0-9]*)+$`)

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
		source, addedLines, snippets := slp069Source(f.AddedLines())
		fs := token.NewFileSet()
		file := fs.AddFile(f.Path, fs.Base(), len(source))
		var s scanner.Scanner
		s.Init(file, []byte(source), nil, scanner.ScanComments)
		for {
			pos, tok, lit := s.Scan()
			if tok == token.EOF {
				break
			}
			if tok != token.IDENT {
				continue
			}
			line := fs.Position(pos).Line
			if !addedLines[line] {
				continue
			}
			if !info.hasSnake && snakePattern.MatchString(lit) {
				info.hasSnake = true
				info.firstSnakeLine = line
				info.firstSnakeSnippet = snippets[line]
			}
			if !info.hasCamel && (pascalPattern.MatchString(lit) || lowerCamelPattern.MatchString(lit)) {
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

func slp069Source(added []diff.Line) (string, map[int]bool, map[int]string) {
	maxLine := 0
	lineContent := make(map[int]string, len(added))
	addedLines := make(map[int]bool, len(added))
	snippets := make(map[int]string, len(added))
	for _, ln := range added {
		if ln.NewLineNo > maxLine {
			maxLine = ln.NewLineNo
		}
		lineContent[ln.NewLineNo] = ln.Content
		addedLines[ln.NewLineNo] = true
		snippets[ln.NewLineNo] = strings.TrimSpace(ln.Content)
	}
	lines := make([]string, maxLine)
	for i := 1; i <= maxLine; i++ {
		lines[i-1] = lineContent[i]
	}
	return strings.Join(lines, "\n"), addedLines, snippets
}
