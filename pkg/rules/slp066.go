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

// mapDeclRe matches var declarations of map types: `var name = map[...]...` or `name := map[...]...`.
var mapDeclRe = regexp.MustCompile(`(?:var\s+(\w+)\s+=\s+map\[|(\w+)\s+(?::=|=)\s+map\[)`)

// mapIndexRe extracts the identifier being indexed: `ident[`.
var mapIndexRe = regexp.MustCompile(`\b([a-zA-Z_]\w*)\s*\[`)

// mutexDeclRe matches mutex declarations, capturing the variable name.
var mutexDeclRe = regexp.MustCompile(`\b(\w+)\s+sync\.(?:Mutex|RWMutex)\b`)

// indexedMapIdents returns the set of identifier names used in map index
// expressions (e.g., "m" from "m[key]"), excluding slice/array type patterns
// and map literals.
func indexedMapIdents(line string) []string {
	var names []string
	for _, m := range mapIndexRe.FindAllStringSubmatchIndex(line, -1) {
		name := line[m[2]:m[3]]
		// Skip "map" keyword itself (map[string]int type literals).
		if name == "map" {
			continue
		}
		names = append(names, name)
	}
	return names
}

// guardedByMutex reports whether mapName is guarded by a mutex in the added lines.
// It checks:
//  1. A mutex variable whose name contains the map name (e.g., cacheMu guards cache).
//  2. A mutex variable declared within 5 lines of the map declaration (proximity heuristic).
//  3. sync.Map usage (inherently safe).
func guardedByMutex(mapName string, added []diff.Line) bool {
	// Find declaration line index for the map variable.
	mapDeclIdx := -1
	for i, ln := range added {
		m := mapDeclRe.FindStringSubmatch(ln.Content)
		if m == nil {
			continue
		}
		declared := m[1]
		if declared == "" {
			declared = m[2]
		}
		if declared == mapName {
			mapDeclIdx = i
			break
		}
	}

	// Collect mutex variable names and their declaration line indices.
	type mutexInfo struct {
		name string
		idx  int
	}
	var mutexes []mutexInfo
	hasSyncMap := false
	for i, ln := range added {
		if strings.Contains(ln.Content, "sync.Map") {
			hasSyncMap = true
		}
		if m := mutexDeclRe.FindStringSubmatch(ln.Content); m != nil {
			mutexes = append(mutexes, mutexInfo{name: m[1], idx: i})
		}
	}

	if hasSyncMap {
		return true
	}

	for _, mu := range mutexes {
		// Guard by naming convention: mutex name contains map name (e.g., cacheMu, muCache).
		lowerMu := strings.ToLower(mu.name)
		lowerMap := strings.ToLower(mapName)
		if strings.Contains(lowerMu, lowerMap) || strings.Contains(lowerMap, lowerMu) {
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
		hasConcurrent := false
		for _, ln := range added {
			if strings.Contains(ln.Content, "go ") || strings.Contains(ln.Content, "sync.WaitGroup") {
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
			for _, mapIdent := range indexedMapIdents(ln.Content) {
				if reported[mapIdent] {
					continue
				}
				if guardedByMutex(mapIdent, added) {
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
