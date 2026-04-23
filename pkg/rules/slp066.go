package rules

import (
	"regexp"
	"strings"

	"github.com/messagesgoel-blip/slopgate/pkg/diff"
)

// SLP066 flags concurrent map access without mutex protection in Go files.
//
// Heuristic: if the diff contains goroutines or WaitGroup usage and also
// contains map index/read/write operations, flag each indexed map identifier
// unless a sync.Mutex/sync.RWMutex or sync.Map guard is associated with it.
// "Associated" means the mutex variable name contains the map variable name
// (e.g., "cacheMu" guards "cache") or a mutex declaration appears in the same
// block as the map declaration (within 5 lines).
//
// This is intentionally coarse — precisely matching mutex guards to specific
// map variables requires full AST analysis which is out of scope for
// diff-based linting.
type SLP066 struct{}

func (SLP066) ID() string                { return "SLP066" }
func (SLP066) DefaultSeverity() Severity { return SeverityBlock }
func (SLP066) Description() string {
	return "concurrent map access without mutex protection"
}

var mapDeclPatterns = []*regexp.Regexp{
	regexp.MustCompile(`\bvar\s+(\w+)\s+map\[`),
	regexp.MustCompile(`\bvar\s+(\w+)\s*=\s*(?:make\s*\(\s*)?map\[`),
	regexp.MustCompile(`\b(\w+)\s*(?::=|=)\s*(?:make\s*\(\s*)?map\[`),
}

// mapIndexRe extracts the identifier being indexed: `ident[`.
var mapIndexRe = regexp.MustCompile(`\b([a-zA-Z_]\w*)\s*\[`)

// mutexDeclRe matches mutex declarations, capturing the variable name.
var mutexDeclRe = regexp.MustCompile(`\b(\w+)\s+sync\.(?:Mutex|RWMutex)\b`)
var syncMapDeclRe = regexp.MustCompile(`(?:var\s+(\w+)\s+sync\.Map\b|(\w+)\s*(?::=|=)\s*sync\.Map(?:\{\})?)`)
var slp066GoroutineRe = regexp.MustCompile(`\bgo\s+(?:func\s*\(|[A-Za-z_]\w*(?:\.\w+)?\s*\()`)

// indexedMapIdents returns identifier names used in map index expressions,
// limited to names that are declared as maps in the visible hunk context.
func indexedMapIdents(line string, knownMaps map[string]bool) []string {
	var names []string
	for _, m := range mapIndexRe.FindAllStringSubmatchIndex(line, -1) {
		name := line[m[2]:m[3]]
		if name == "map" || !knownMaps[name] {
			continue
		}
		names = append(names, name)
	}
	return names
}

func slp066DeclaredMaps(lines []diff.Line) map[string]bool {
	out := make(map[string]bool)
	for _, ln := range lines {
		for _, re := range mapDeclPatterns {
			if m := re.FindStringSubmatch(ln.Content); m != nil {
				for _, name := range m[1:] {
					if name != "" {
						out[name] = true
					}
				}
			}
		}
	}
	return out
}

// guardedByMutex reports whether mapName is guarded by a mutex in the added lines.
// It checks:
//  1. A mutex variable whose name contains the map name (e.g., cacheMu guards cache).
//  2. A mutex variable declared within 5 lines of the map declaration (proximity heuristic).
//  3. sync.Map usage (inherently safe).
func guardedByMutex(mapName string, lines []diff.Line, syncMapNames map[string]bool) bool {
	if syncMapNames[mapName] {
		return true
	}
	// Find declaration line index for the map variable.
	mapDeclIdx := -1
	for i, ln := range lines {
		for _, re := range mapDeclPatterns {
			m := re.FindStringSubmatch(ln.Content)
			if m == nil {
				continue
			}
			for _, declared := range m[1:] {
				if declared == mapName {
					mapDeclIdx = i
					break
				}
			}
			if mapDeclIdx >= 0 {
				break
			}
		}
		if mapDeclIdx >= 0 {
			break
		}
	}

	// Collect mutex variable names and their declaration line indices.
	type mutexInfo struct {
		name string
		idx  int
	}
	var mutexes []mutexInfo
	for i, ln := range lines {
		if m := mutexDeclRe.FindStringSubmatch(ln.Content); m != nil {
			mutexes = append(mutexes, mutexInfo{name: m[1], idx: i})
		}
	}

	for _, mu := range mutexes {
		// Guard by naming convention: mutex name contains map name (e.g., cacheMu, muCache).
		lowerMu := strings.ToLower(mu.name)
		lowerMap := strings.ToLower(mapName)
		if strings.Contains(lowerMu, lowerMap) {
			return true
		}
		if len(lowerMu) >= 3 && strings.Contains(lowerMap, lowerMu) {
			return true
		}
		// Guard by proximity: mutex declared within 5 lines of map declaration.
		if mapDeclIdx >= 0 {
			dist := mu.idx - mapDeclIdx
			if dist < 0 {
				dist = -dist
			}
			if dist <= 5 {
				return true
			}
		}
	}
	return false
}

func (r SLP066) Check(d *diff.Diff) []Finding {
	var out []Finding
	for _, f := range d.Files {
		if f.IsDelete || !strings.HasSuffix(f.Path, ".go") {
			continue
		}
		added := f.AddedLines()
		var visible []diff.Line
		for _, h := range f.Hunks {
			for _, ln := range h.Lines {
				if ln.Kind != diff.LineDelete {
					visible = append(visible, ln)
				}
			}
		}
		knownMaps := slp066DeclaredMaps(visible)
		syncMapNames := make(map[string]bool)
		for _, ln := range visible {
			if m := syncMapDeclRe.FindStringSubmatch(ln.Content); m != nil {
				name := m[1]
				if name == "" {
					name = m[2]
				}
				if name != "" {
					syncMapNames[name] = true
				}
			}
		}
		hasConcurrent := false
		for _, ln := range added {
			clean := stripCommentAndStrings(ln.Content)
			if slp066GoroutineRe.MatchString(clean) || strings.Contains(clean, "sync.WaitGroup") {
				hasConcurrent = true
				break
			}
		}
		if !hasConcurrent {
			continue
		}
		// Track which map identifiers have already been reported to avoid duplicates.
		reported := make(map[string]bool)
		for _, ln := range added {
			for _, mapIdent := range indexedMapIdents(stripCommentAndStrings(ln.Content), knownMaps) {
				if reported[mapIdent] {
					continue
				}
				if guardedByMutex(mapIdent, visible, syncMapNames) {
					continue
				}
				reported[mapIdent] = true
				out = append(out, Finding{
					RuleID:   r.ID(),
					Severity: r.DefaultSeverity(),
					File:     f.Path,
					Line:     ln.NewLineNo,
					Message:  "map " + mapIdent + " accessed concurrently without mutex — add sync.Mutex or use sync.Map",
					Snippet:  strings.TrimSpace(ln.Content),
				})
			}
		}
	}
	return out
}
