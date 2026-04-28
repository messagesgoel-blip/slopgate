package rules

import "testing"

func TestSLP115_FiresOnNarrowExtensionCheck(t *testing.T) {
	d := parseDiff(t, `diff --git a/config.go b/config.go
--- a/config.go
+++ b/config.go
@@ -1,1 +1,3 @@
 package main
+
+func isJSFile(path string) bool { return strings.HasSuffix(path, ".js") }
`)
	got := SLP115{}.Check(d)
	if len(got) == 0 {
		t.Fatal("expected findings for narrow extension check .js without .mjs/.cjs")
	}
}

func TestSLP115_NoFireOnBroaderExtensionCheck(t *testing.T) {
	d := parseDiff(t, `diff --git a/config.go b/config.go
--- a/config.go
+++ b/config.go
@@ -1,1 +1,3 @@
 package main
+
+func isJSFile(path string) bool { return strings.HasSuffix(path, ".js") || strings.HasSuffix(path, ".mjs") }
`)
	got := SLP115{}.Check(d)
	if len(got) != 0 {
		t.Fatalf("expected 0 findings when broader check present, got %d", len(got))
	}
}

func TestSLP115_IsExtBorder(t *testing.T) {
	borderChars := []byte{' ', '\t', '/', '"', '\'', '(', ')'}
	for _, c := range borderChars {
		if !slp115IsExtBorder(c) {
			t.Errorf("expected %q to be an extension border", c)
		}
	}
	nonBorderChars := []byte{'a', 'Z', '0', '9', '_', '.'}
	for _, c := range nonBorderChars {
		if slp115IsExtBorder(c) {
			t.Errorf("expected %q to NOT be an extension border", c)
		}
	}
}

func TestSLP115_ContainsExtTokenNoMatchOnMapSuffix(t *testing.T) {
	if slp115ContainsExtToken(".js.map", ".js") {
		t.Error("expected .js NOT to match inside .js.map")
	}
	if slp115ContainsExtToken(".css.map", ".css") {
		t.Error("expected .css NOT to match inside .css.map")
	}
}

func TestSLP115_ContainsExtTokenMatchOnPlain(t *testing.T) {
	if !slp115ContainsExtToken(".js", ".js") {
		t.Error("expected .js to match plain .js")
	}
	if !slp115ContainsExtToken(".css", ".css") {
		t.Error("expected .css to match plain .css")
	}
	if !slp115ContainsExtToken("path \".js\"", ".js") {
		t.Error("expected .js to match when in quotes")
	}
}

func TestSLP115_Description(t *testing.T) {
	r := SLP115{}
	if r.ID() != "SLP115" {
		t.Errorf("ID = %q", r.ID())
	}
	if r.Description() == "" {
		t.Errorf("Description is empty")
	}
	if r.DefaultSeverity() != SeverityInfo {
		t.Errorf("default severity should be info")
	}
}
