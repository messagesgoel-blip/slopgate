package rules

import "testing"

func TestSLP118_FiresOnDirectIndexAccess(t *testing.T) {
	d := parseDiff(t, `diff --git a/process.go b/process.go
--- a/process.go
+++ b/process.go
@@ -1,1 +1,3 @@
 package main
+
+var first = items[0]
`)
	got := SLP118{}.Check(d)
	if len(got) == 0 {
		t.Fatal("expected findings for index access without guard")
	}
}

func TestSLP118_NoFireOnSlicing(t *testing.T) {
	d := parseDiff(t, `diff --git a/process.go b/process.go
--- a/process.go
+++ b/process.go
@@ -1,1 +1,3 @@
 package main
+
+var subset = items[1:3]
`)
	got := SLP118{}.Check(d)
	if len(got) != 0 {
		t.Fatalf("expected 0 findings for slicing, got %d", len(got))
	}
}

func TestSLP118_NoFireOnGuardedAccess(t *testing.T) {
	d := parseDiff(t, `diff --git a/process.go b/process.go
--- a/process.go
+++ b/process.go
@@ -1,1 +1,4 @@
 package main
+
+if len(items) > 0 {
+    var first = items[0]
+}
`)
	got := SLP118{}.Check(d)
	if len(got) != 0 {
		t.Fatalf("expected 0 findings for guarded access, got %d", len(got))
	}
}

func TestSLP118_Description(t *testing.T) {
	r := SLP118{}
	if r.ID() != "SLP118" {
		t.Errorf("ID = %q", r.ID())
	}
	if r.Description() == "" {
		t.Errorf("Description is empty")
	}
	if r.DefaultSeverity() != SeverityBlock {
		t.Errorf("default severity should be block")
	}
}
