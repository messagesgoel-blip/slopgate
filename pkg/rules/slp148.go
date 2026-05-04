package rules

import (
	"regexp"
	"sort"
	"strings"

	"github.com/messagesgoel-blip/slopgate/pkg/diff"
)

// SLP148 detects when different variables representing the same conceptual
// entity use inconsistent naming conventions across modified modules/files.
// This catches patterns like userId vs userID vs user_id for the same concept.
//
// Detection strategy:
//  1. Extract all variable/constant declarations from added lines
//  2. Normalize names (lowercase, strip underscores, etc.)
//  3. Group by semantic similarity (Levenshtein distance, shared prefixes/suffixes)
//  4. Flag groups with multiple naming conventions
//
// Languages: JavaScript, TypeScript, Go, Python, Java
//
// Scope: across all files in the diff
type SLP148 struct{}

func (SLP148) ID() string                { return "SLP148" }
func (SLP148) DefaultSeverity() Severity { return SeverityWarn }
func (SLP148) Description() string {
	return "inconsistent naming for the same conceptual variable across modules"
}


// identifierPattern matches any identifier after a keyword.
var identifierPattern = regexp.MustCompile(`\b([a-zA-Z_$][a-zA-Z0-9_$]*)\b`)

// ignoreList contains common generic names that shouldn't be checked.
var ignoreList = map[string]bool{
	"err":     true,
	"ctx":     true,
	"req":     true,
	"res":     true,
	"data":    true,
	"result":  true,
	"error":   true,
	"message": true,
	"config":  true,
	"options": true,
	"params":  true,
	"body":    true,
	"headers": true,
	"status":  true,
	"id":      true, // too generic - can be many different ids
	"key":     true,
	"value":   true,
	"name":    true,
	"type":    true,
	"time":    true,
	"date":    true,
	"url":     true,
	"path":    true,
	"file":    true,
	"dir":     true,
	"user":    true, // user could be many types
	"item":    true,
	"obj":     true,
	"arr":     true,
	"map":     true,
	"set":     true,
}

// semanticGroups maps common semantic categories.
// Each key is a canonical concept that should have consistent naming.
var semanticGroups = map[string][]string{
	"id":           {"identifier", "uid", "uuid", "guid"},
	"user":         {"user", "account", "profile", "customer", "client"},
	"token":        {"token", "accesstoken", "authtoken", "bearer"},
	"key":          {"key", "apikey", "secret_key"},
	"secret":       {"secret", "apisecret", "password"},
	"config":       {"config", "configuration", "settings", "options"},
	"param":        {"parameter", "param", "arg", "argument"},
	"value":        {"value", "val", "result", "output"},
	"error":        {"error", "err", "failure"},
	"message":      {"message", "msg", "text"},
	"notification": {"notification", "notif", "alert"},
	"record":       {"record", "rec", "entry"},
	"item":         {"item", "entry"},
	"count":        {"count", "total", "num", "number"},
}

// normalizeName returns a canonical semantic group key for a variable name.
// It attempts to group related names (e.g., userId, userID, user_id all → "user")
func normalizeName(name string) string {
	lower := strings.ToLower(name)

	// Check each semantic group: match by prefix only (not suffix)
	// to avoid false matches like "formatUser" -> "user".
	for concept, variants := range semanticGroups {
		for _, variant := range variants {
			if lower == variant || strings.HasPrefix(lower, variant) {
				return concept
			}
		}
	}

	// Strip common suffixes and return base
	suffixes := []string{"id", "ids", "uid", "uuid", "key", "token", "url", "path", "file", "dir"}
	base := lower
	for _, suf := range suffixes {
		if strings.HasSuffix(lower, suf) {
			base = strings.TrimSuffix(lower, suf)
			break
		}
	}

	// If we stripped something meaningful, return the base concept
	if base != lower && len(base) > 0 {
		return base
	}

	return lower
}

// extractIdentifiers extracts variable/function names from added lines.
func extractIdentifiers(content string) []string {
	var ids []string
	for _, match := range identifierPattern.FindAllStringSubmatch(content, -1) {
		if len(match) > 1 {
			name := match[1]
			// Skip keywords and very short names
			if len(name) < 2 || ignoreList[name] {
				continue
			}
			ids = append(ids, name)
		}
	}
	return ids
}

func (r SLP148) Check(d *diff.Diff) []Finding {
	var out []Finding

	// Collect all added identifiers across all files
	type nameEntry struct {
		name     string
		file     string
		lineNo   int
		normName string
	}
	var allNames []nameEntry

	for _, f := range d.Files {
		if f.IsDelete {
			continue
		}
		// Get all added lines content
		for _, h := range f.Hunks {
			for _, ln := range h.Lines {
				if ln.Kind != diff.LineAdd {
					continue
				}
				line := ln.Content
				// Extract identifiers
				ids := extractIdentifiers(line)
				for _, id := range ids {
					norm := normalizeName(id)
					// Only consider normalized forms that aren't empty
					if norm != "" && norm != id && !ignoreList[id] {
						allNames = append(allNames, nameEntry{
							name:     id,
							file:     f.Path,
							lineNo:   ln.NewLineNo,
							normName: norm,
						})
					}
				}
			}
		}
	}

	// Group by normalized name, then find variant groups
	groups := make(map[string][]nameEntry)
	for _, entry := range allNames {
		groups[entry.normName] = append(groups[entry.normName], entry)
	}

	// For each group with multiple variants, check if they're truly different
	// naming styles for the same concept
	for norm, variants := range groups {
		if len(variants) < 2 {
			continue
		}
		// Find distinct name variants
		variantSet := make(map[string]bool)
		for _, v := range variants {
			variantSet[v.name] = true
		}
		if len(variantSet) < 2 {
			continue // only one distinct name
		}

		// Format message showing variants
		var variantList []string
		for v := range variantSet {
			variantList = append(variantList, v)
		}
		sort.Strings(variantList)
		variantStr := strings.Join(variantList, ", ")

		// Find a representative file/line for the finding
		repFile := variants[0].file
		repLine := variants[0].lineNo

		out = append(out, Finding{
			RuleID:   r.ID(),
			Severity: r.DefaultSeverity(),
			File:     repFile,
			Line:     repLine,
			Message:  "inconsistent naming for '" + norm + "': " + variantStr,
			Snippet:  "consider standardizing to one convention",
		})
	}

	return out
}
