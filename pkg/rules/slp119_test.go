package rules

import "testing"

func TestSLP119_FiresOnTrimSuffixUse(t *testing.T) {
	d := parseDiff(t, `diff --git a/process.go b/process.go
--- a/process.go
+++ b/process.go
@@ -1,1 +1,3 @@
 package main
+
+var result = strings.TrimSuffix(name, ".txt")
`)
	got := SLP119{}.Check(d)
	if len(got) == 0 {
		t.Fatal("expected findings for TrimSuffix without empty check")
	}
}

func TestSLP119_FiresOnTrimPrefixUse(t *testing.T) {
	d := parseDiff(t, `diff --git a/process.go b/process.go
--- a/process.go
+++ b/process.go
@@ -1,1 +1,3 @@
 package main
+
+var result = strings.TrimPrefix(path, "/api/")
`)
	got := SLP119{}.Check(d)
	if len(got) == 0 {
		t.Fatal("expected findings for TrimPrefix without empty check")
	}
}

func TestSLP119_NoFireOnNonTrimCode(t *testing.T) {
	d := parseDiff(t, `diff --git a/process.go b/process.go
--- a/process.go
+++ b/process.go
@@ -1,1 +1,3 @@
 package main
+
+var result = name + ".bak"
`)
	got := SLP119{}.Check(d)
	if len(got) != 0 {
		t.Fatalf("expected 0 findings for non-trim code, got %d", len(got))
	}
}

func TestSLP119_Description(t *testing.T) {
	r := SLP119{}
	if r.ID() != "SLP119" {
		t.Errorf("ID = %q", r.ID())
	}
	if r.Description() == "" {
		t.Errorf("Description is empty")
	}
	if r.DefaultSeverity() != SeverityWarn {
		t.Errorf("default severity should be warn")
	}
}
