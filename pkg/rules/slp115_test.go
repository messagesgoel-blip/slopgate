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
