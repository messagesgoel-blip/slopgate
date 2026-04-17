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
	return ext == ".js" || ext == ".ts" || ext == ".tsx" || ext == ".jsx" || ext == ".mjs" || ext == ".cjs"
}

// isPythonFile reports whether path is a Python file (.py, .pyi, .pyw).
func isPythonFile(path string) bool {
	ext := strings.ToLower(filepath.Ext(path))
	return ext == ".py" || ext == ".pyi" || ext == ".pyw"
}

// isJavaFile reports whether path is a Java or Kotlin file.
func isJavaFile(path string) bool {
	ext := strings.ToLower(filepath.Ext(path))
	return ext == ".java" || ext == ".kt"
}

// isRustFile reports whether path ends with .rs (case-insensitive, matching isJavaFile behaviour).
func isRustFile(path string) bool {
	return strings.ToLower(filepath.Ext(path)) == ".rs"
}

// isJavaTestFile reports whether path is a Java test file.
// Convention: file name contains "Test" (JUnit) or file lives under
// src/test/ (Maven/Gradle convention).
func isJavaTestFile(path string) bool {
	if !isJavaFile(path) {
		return false
	}
	lower := strings.ToLower(path)
	// Strip the extension so "FooTest.java" → "footest", not "footest.java".
	base := strings.ToLower(strings.TrimSuffix(filepath.Base(path), filepath.Ext(path)))
	return strings.HasPrefix(base, "test") || strings.HasSuffix(base, "test") || strings.HasSuffix(base, "tests") ||
		strings.Contains(lower, "/test/") ||
		strings.Contains(lower, "\\test\\")
}

// isRustTestFile reports whether path is a Rust test file.
// Rust tests live in *_test.rs modules or under tests/ directories.
func isRustTestFile(path string) bool {
	if !isRustFile(path) {
		return false
	}
	return strings.HasSuffix(path, "_test.rs") ||
		strings.Contains(path, "/tests/") ||
		strings.Contains(path, "\\tests\\")
}
