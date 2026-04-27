package rules

import (
	"path/filepath"
	"strings"

	"github.com/messagesgoel-blip/slopgate/pkg/diff"
)

type SLP113 struct{}

func (SLP113) ID() string                { return "SLP113" }
func (SLP113) DefaultSeverity() Severity { return SeverityWarn }
func (SLP113) Description() string {
	return "source file changed without corresponding test update — update tests or add a test file"
}

var slp113SourceExtMap = map[string]string{
	".go":   "_test.go",
	".js":   ".test.js",
	".ts":   ".test.ts",
	".tsx":  ".test.tsx",
	".jsx":  ".test.jsx",
	".py":   "_test.py",
	".java": "Test.java",
	".kt":   "Test.kt",
}

func slp113IsTestFile(path string) bool {
	base := filepath.Base(path)
	ext := filepath.Ext(path)
	testSuffix, ok := slp113SourceExtMap[ext]
	if ok && strings.HasSuffix(base, testSuffix) {
		return true
	}
	if strings.Contains(path, ".test.") || strings.Contains(path, "_test.") {
		return true
	}
	dir := filepath.Dir(path)
	testDir := filepath.ToSlash(filepath.Join(dir, "testdata"))
	if strings.HasPrefix(path, testDir+"/") {
		return true
	}
	return false
}

func slp113SourceExts() map[string]string {
	return slp113SourceExtMap
}

func slp113TestPath(dir, base, testSuffix string) string {
	if dir == "." || dir == "" {
		return base + testSuffix
	}
	return dir + "/" + base + testSuffix
}

func slp113HasTestFile(sourcePath string, allFiles map[string]bool) bool {
	ext := filepath.Ext(sourcePath)
	testSuffix, ok := slp113SourceExtMap[ext]
	if !ok {
		return true
	}

	dir := filepath.Dir(sourcePath)
	base := strings.TrimSuffix(filepath.Base(sourcePath), ext)

	if strings.HasSuffix(testSuffix, ext) {
		testName := slp113TestPath(dir, base, testSuffix)
		if allFiles[testName] {
			return true
		}
	}

	testName := slp113TestPath(dir, base, testSuffix)
	if allFiles[testName] {
		return true
	}

	testDir := filepath.ToSlash(filepath.Join(dir, "testdata"))
	for f := range allFiles {
		if strings.HasPrefix(f, testDir+"/") && strings.HasPrefix(filepath.Base(f), base) {
			return true
		}
	}

	return false
}

func (r SLP113) Check(d *diff.Diff) []Finding {
	var out []Finding
	allFiles := make(map[string]bool)
	sourceFiles := make(map[string]bool)

	for _, f := range d.Files {
		if !f.IsDelete {
			allFiles[f.Path] = true
			ext := filepath.Ext(f.Path)
			if _, ok := slp113SourceExtMap[ext]; ok {
				if !slp113IsTestFile(f.Path) {
					sourceFiles[f.Path] = true
				}
			}
		}
	}

	for _, f := range d.Files {
		if f.IsDelete || isDocFile(f.Path) {
			continue
		}

		ext := filepath.Ext(f.Path)
		if _, ok := slp113SourceExtMap[ext]; !ok {
			continue
		}
		if slp113IsTestFile(f.Path) {
			continue
		}

		if !slp113HasTestFile(f.Path, allFiles) {
			dir := filepath.Dir(f.Path)
			base := strings.TrimSuffix(filepath.Base(f.Path), ext)
			testSuffix, _ := slp113SourceExtMap[ext]
			var expectedTestFile string
			switch ext {
			case ".go":
				expectedTestFile = slp113TestPath(dir, base, "_test.go")
			case ".py":
				expectedTestFile = slp113TestPath(dir, base, "_test.py")
			case ".java":
				expectedTestFile = slp113TestPath(dir, base, "Test.java")
			case ".kt":
				expectedTestFile = slp113TestPath(dir, base, "Test.kt")
			default:
				expectedTestFile = slp113TestPath(dir, base, testSuffix)
			}
			out = append(out, Finding{
				RuleID:   r.ID(),
				Severity: r.DefaultSeverity(),
				File:     f.Path,
				Line:     1,
				Message:  "source file changed without corresponding test file — expected " + expectedTestFile,
				Snippet:  f.Path,
			})
		}
	}

	return out
}