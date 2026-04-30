package rules

import (
	"os"
	"path"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/messagesgoel-blip/slopgate/pkg/diff"
)

// SLP051 flags bare function calls in added code that may be undefined.
// We look for identifier(arg...) patterns and skip builtins, keywords,
// and method calls (which come from imported packages).
type SLP051 struct{}

func (SLP051) ID() string                { return "SLP051" }
func (SLP051) DefaultSeverity() Severity { return SeverityBlock }
func (SLP051) Description() string {
	return "call to potentially undefined function — implement or import it"
}

// goKeywords are Go keywords, predeclared identifiers, builtins, and type
// names that may legally appear before '(' without implying an undefined call.
var goKeywords = map[string]bool{
	"if": true, "for": true, "switch": true, "select": true,
	"return": true, "defer": true, "go": true, "panic": true,
	"recover": true, "print": true, "println": true,
	"import": true, "var": true, "const": true, "type": true,
	"new": true, "make": true, "len": true, "cap": true,
	"append": true, "copy": true, "delete": true, "close": true,
	"complex": true, "real": true, "imag": true,
	"min": true, "max": true, "clear": true,
	"func": true,
	"bool": true, "byte": true, "rune": true,
	"string": true, "int": true, "int8": true, "int16": true, "int32": true, "int64": true,
	"uint": true, "uint8": true, "uint16": true, "uint32": true, "uint64": true, "uintptr": true,
	"float32": true, "float64": true, "complex64": true, "complex128": true,
	"error": true, "any": true,
	"true": true, "false": true, "nil": true,
}

// undefinedCallPattern matches a bare identifier followed immediately by '('.
var undefinedCallPattern = regexp.MustCompile(`\b([a-zA-Z_]\w*)\s*\(`)

func (r SLP051) Check(d *diff.Diff) []Finding {
	var out []Finding
	packageSymbols := slp051PackageSymbols(d)
	for _, f := range d.Files {
		if f.IsDelete || !isGoFile(f.Path) {
			continue
		}
		localFuncs := slp051LocalSymbols(f)
		for name := range packageSymbols[slp051PackageDir(f.Path)] {
			localFuncs[name] = true
		}
		for _, h := range f.Hunks {
			for _, ln := range h.Lines {
				if ln.Kind != diff.LineAdd {
					continue
				}
				content := strings.TrimSpace(stripCommentAndStrings(ln.Content))
				if content == "" {
					continue
				}
				// Find all bare calls.
				for _, m := range undefinedCallPattern.FindAllStringSubmatchIndex(content, -1) {
					name := content[m[2]:m[3]]
					if goKeywords[name] || isSafetyTestName(name) {
						continue
					}
					// Skip method calls: check the character immediately before the match start.
					if m[2] > 0 && content[m[2]-1] == '.' {
						continue
					}
					// Skip if there is a local func definition for this name somewhere in the file.
					if localFuncs[name] {
						continue
					}
					out = append(out, Finding{
						RuleID:   r.ID(),
						Severity: r.DefaultSeverity(),
						File:     f.Path,
						Line:     ln.NewLineNo,
						Message:  "call to undefined function " + name + " — implement or import it",
						Snippet:  strings.TrimSpace(ln.Content),
					})
				}
			}
		}
	}
	return out
}

// funcDefPattern matches a function/method definition.
var funcDefPattern = regexp.MustCompile(`^func\s+(?:\([^)]+\)\s+)?([a-zA-Z_]\w*)(?:\s*\[[^\]]+\])?\s*\(`)

var typeDefPattern = regexp.MustCompile(`^type\s+([a-zA-Z_]\w*)\b`)
var typeBlockStartPattern = regexp.MustCompile(`^type\s*\(`)
var typeBlockSymbolPattern = regexp.MustCompile(`^([a-zA-Z_]\w*)\b`)

func slp051PackageDir(filePath string) string {
	normalized := strings.ReplaceAll(filePath, "\\", "/")
	dir := path.Dir(normalized)
	if dir == "." {
		return ""
	}
	return dir
}

func slp051AddSymbolFromLine(localSymbols map[string]bool, line string) {
	trimmed := strings.TrimSpace(line)
	if m := funcDefPattern.FindStringSubmatch(trimmed); len(m) > 1 {
		localSymbols[m[1]] = true
	}
	if m := typeDefPattern.FindStringSubmatch(trimmed); len(m) > 1 {
		localSymbols[m[1]] = true
	}
}

func slp051LocalSymbols(f diff.File) map[string]bool {
	localSymbols := make(map[string]bool)
	var lines []string
	for _, h := range f.Hunks {
		for _, ln := range h.Lines {
			if ln.Kind == diff.LineDelete {
				continue
			}
			lines = append(lines, ln.Content)
		}
	}
	slp051CollectSymbolsFromLines(localSymbols, lines)
	return localSymbols
}

func slp051CollectSymbolsFromText(localSymbols map[string]bool, content string) {
	slp051CollectSymbolsFromLines(localSymbols, strings.Split(content, "\n"))
}

func slp051CollectSymbolsFromLines(localSymbols map[string]bool, lines []string) {
	inTypeBlock := false
	typeBlockBraceDepth := 0
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if inTypeBlock {
			if typeBlockBraceDepth == 0 && strings.HasPrefix(trimmed, ")") {
				inTypeBlock = false
				continue
			}
			if typeBlockBraceDepth == 0 {
				if m := typeBlockSymbolPattern.FindStringSubmatch(trimmed); len(m) > 1 {
					localSymbols[m[1]] = true
				}
			}
			typeBlockBraceDepth += strings.Count(trimmed, "{") - strings.Count(trimmed, "}")
			if typeBlockBraceDepth < 0 {
				typeBlockBraceDepth = 0
			}
			continue
		}
		slp051AddSymbolFromLine(localSymbols, line)
		if typeBlockStartPattern.MatchString(trimmed) {
			inTypeBlock = true
			typeBlockBraceDepth = 0
		}
	}
}

func slp051IsProductionGoFile(path string) bool {
	return isGoFile(path) && !isTestFile(path)
}

func slp051AddDiffLocalSymbols(symbols map[string]bool, d *diff.Diff, dir string) {
	for _, f := range d.Files {
		if !f.IsDelete && slp051IsProductionGoFile(f.Path) && slp051PackageDir(f.Path) == dir {
			for name := range slp051LocalSymbols(f) {
				symbols[name] = true
			}
		}
	}
}

func slp051RepoRoot() (string, bool) {
	wd, err := os.Getwd()
	if err != nil {
		return "", false
	}
	root, err := filepath.Abs(wd)
	if err != nil {
		return "", false
	}
	evaluatedRoot, err := filepath.EvalSymlinks(root)
	if err != nil {
		return "", false
	}
	return filepath.Clean(evaluatedRoot), true
}

func slp051RepoRootForDiff(d *diff.Diff) (string, bool) {
	if d != nil && d.RepoRoot != "" {
		return slp051ResolveExistingPathInRepo(d.RepoRoot, d.RepoRoot)
	}
	return slp051RepoRoot()
}

func slp051ResolvePackageDir(repoRoot, dir string) (string, bool) {
	if repoRoot == "" {
		return "", false
	}
	evaluatedRoot, ok := slp051ResolveExistingPathInRepo(repoRoot, repoRoot)
	if !ok {
		return "", false
	}
	if filepath.IsAbs(filepath.FromSlash(dir)) {
		return "", false
	}
	cleanSlash := path.Clean(strings.ReplaceAll(dir, "\\", "/"))
	if cleanSlash == "." {
		cleanSlash = ""
	}
	if cleanSlash == ".." || strings.HasPrefix(cleanSlash, "../") {
		return "", false
	}
	target := filepath.Join(evaluatedRoot, filepath.FromSlash(cleanSlash))
	return slp051ResolveExistingPathInRepo(evaluatedRoot, target)
}

func slp051ResolveExistingPathInRepo(repoRoot, target string) (string, bool) {
	rootAbs, err := filepath.Abs(repoRoot)
	if err != nil {
		return "", false
	}
	evaluatedRoot, err := filepath.EvalSymlinks(rootAbs)
	if err != nil {
		return "", false
	}
	targetAbs, err := filepath.Abs(target)
	if err != nil {
		return "", false
	}
	evaluatedTarget, err := filepath.EvalSymlinks(targetAbs)
	if err != nil {
		return "", false
	}
	rel, err := filepath.Rel(evaluatedRoot, evaluatedTarget)
	if err != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return "", false
	}
	return filepath.Clean(evaluatedTarget), true
}

func slp051PackageSymbols(d *diff.Diff) map[string]map[string]bool {
	dirs := make(map[string]bool)
	for _, f := range d.Files {
		if !f.IsDelete && slp051IsProductionGoFile(f.Path) {
			dirs[slp051PackageDir(f.Path)] = true
		}
	}

	out := make(map[string]map[string]bool, len(dirs))
	for dir := range dirs {
		symbols := make(map[string]bool)
		slp051AddDiffLocalSymbols(symbols, d, dir)

		if repoRoot, ok := slp051RepoRootForDiff(d); ok {
			if globDir, ok := slp051ResolvePackageDir(repoRoot, dir); ok {
				matches, err := filepath.Glob(filepath.Join(globDir, "*.go"))
				if err != nil {
					out[dir] = symbols
					continue
				}
				for _, match := range matches {
					evaluatedMatch, ok := slp051ResolveExistingPathInRepo(repoRoot, match)
					if !ok {
						continue
					}
					if strings.HasSuffix(strings.ToLower(match), "_test.go") ||
						strings.HasSuffix(strings.ToLower(evaluatedMatch), "_test.go") {
						continue
					}
					content, readErr := os.ReadFile(evaluatedMatch) // #nosec G304 -- evaluated symlink target is constrained to repo root above.
					if readErr == nil {
						slp051CollectSymbolsFromText(symbols, string(content))
					}
				}
			}
		}
		out[dir] = symbols
	}
	return out
}
