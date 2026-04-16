package rules

import (
	"path/filepath"
	"strings"
)

// isGoFile reports whether path ends with .go.
func isGoFile(path string) bool {
	return strings.HasSuffix(path, ".go")
}

// isJSOrTSFile reports whether path is a JS or TS file.
func isJSOrTSFile(path string) bool {
	ext := strings.ToLower(filepath.Ext(path))
	return ext == ".js" || ext == ".ts" || ext == ".tsx" || ext == ".jsx" || ext == ".mjs"
}

// isPythonFile reports whether path ends with .py.
func isPythonFile(path string) bool {
	ext := strings.ToLower(filepath.Ext(path))
	return ext == ".py"
}
