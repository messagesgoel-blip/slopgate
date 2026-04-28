// Package diff provides AST analysis capabilities for Go files
// in the diff. This enables semantic rules that can track
// cross-function type information.
package diff

import (
	"fmt"
	"go/ast"
	"go/importer"
	"go/parser"
	"go/token"
	"go/types"
	"strings"
)

// FileAnalysis holds the AST and type information for a single Go file.
type FileAnalysis struct {
	Path  string
	AST   *ast.File
	Types *types.Package
	Info  *types.Info
	// Error is the error encountered during AST/type parsing, if any.
	Error error
}

// AnalysisResult holds the AST analysis for all modified Go files in the diff.
type AnalysisResult struct {
	// Files maps file path -> analysis. Only Go files that could be parsed
	// are included. Files that failed to parse have Error set.
	Files map[string]*FileAnalysis
	// GoFiles is the list of Go file paths in the diff, in order.
	GoFiles []string
}

// LoadASTAnalysis parses the added/changed Go files in the diff using
// the standard library's go/parser and go/types. This enables AST-aware
// rules that can track cross-function type information.
//
// The AnalysisResult is keyed by the new file path (Path field of each File).
//
// If a file cannot be parsed, it is included with a non-nil Error field
// so rules can handle partial failures gracefully.
func LoadASTAnalysis(d *Diff) *AnalysisResult {
	result := &AnalysisResult{
		Files: make(map[string]*FileAnalysis),
	}
	for _, f := range d.Files {
		if f.IsDelete {
			continue
		}
		if !strings.HasSuffix(f.Path, ".go") {
			continue
		}
		result.GoFiles = append(result.GoFiles, f.Path)
		fa := &FileAnalysis{Path: f.Path}
		result.Files[f.Path] = fa

		// Reconstruct the new file content from the diff.
		content, err := reconstructFile(d, f)
		if err != nil {
			fa.Error = fmt.Errorf("reconstructing file: %w", err)
			continue
		}

		// Parse the AST.
		fset := token.NewFileSet()
		fa.AST, fa.Error = parser.ParseFile(fset, f.Path, content, parser.AllErrors)
		if fa.Error != nil {
			continue
		}

		// Type-check the file.
		conf := &types.Config{
			Importer: importer.ForCompiler(fset, "source", nil),
			Error:    nil, // we handle errors per-file
		}
		fa.Info = &types.Info{
			Types: make(map[ast.Expr]types.TypeAndValue),
		}
		fa.Types, fa.Error = conf.Check(f.Path, fset, []*ast.File{fa.AST}, fa.Info)
		if fa.Error != nil {
			// Type errors are non-fatal — we still have the AST.
			// Clear the error so the rule can still use the AST.
			// We just lose type precision for this file.
			fa.Error = nil
		}
	}
	return result
}

// reconstructFile rebuilds the new version of a file from the diff.
// For new files, this concatenates all added lines (the entire new file content).
// For modified files, this returns an error — AST analysis works best on new files.
// The reconstructed content can be parsed for AST-aware rules.
func reconstructFile(_ *Diff, target File) (string, error) {
	// For new files, just concatenate all added lines.
	if target.IsNew {
		var lines []string
		for _, h := range target.Hunks {
			for _, ln := range h.Lines {
				if ln.Kind == LineAdd {
					lines = append(lines, ln.Content)
				}
			}
		}
		return strings.Join(lines, "\n") + "\n", nil
	}

	// For modified files, we can't reliably reconstruct without git.
	// AST analysis still has value for checking patterns in added hunks,
	// but we'd need git show to get the base.
	// For now, fall back to regex-only for modified files.
	return "", fmt.Errorf("modified files require git show for full reconstruction; AST analysis works best on new files")
}

// IsGoFile returns true if the path ends with .go and is not a test file
// (unless includeTests is true).
func IsGoFile(path string, includeTests bool) bool {
	if !strings.HasSuffix(path, ".go") {
		return false
	}
	if !includeTests && strings.HasSuffix(path, "_test.go") {
		return false
	}
	return true
}

// HasGoFiles returns true if the diff contains any Go files (excluding test
// files unless includeTests is true).
func HasGoFiles(d *Diff, includeTests bool) bool {
	for _, f := range d.Files {
		if IsGoFile(f.Path, includeTests) {
			return true
		}
	}
	return false
}
