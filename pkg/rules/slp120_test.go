package rules

import "testing"

func TestSLP120_FiresOnDiscard(t *testing.T) {
	d := parseDiff(t, `diff --git a/process.go b/process.go
--- a/process.go
+++ b/process.go
@@ -1,1 +1,3 @@
 package main
+
+_ = doSomething()
`)
	got := SLP120{}.Check(d)
	if len(got) == 0 {
		t.Fatal("expected findings for _ = discard pattern")
	}
}

func TestSLP120_NoFireOnNormalAssignment(t *testing.T) {
	d := parseDiff(t, `diff --git a/process.go b/process.go
--- a/process.go
+++ b/process.go
@@ -1,1 +1,3 @@
 package main
+
+result := doSomething()
`)
	got := SLP120{}.Check(d)
	if len(got) != 0 {
		t.Fatalf("expected 0 findings for normal assignment, got %d", len(got))
	}
}

func TestSLP120_NoFireOnNonGoFile(t *testing.T) {
	d := parseDiff(t, `diff --git a/test.py b/test.py
--- a/test.py
+++ b/test.py
@@ -1,1 +1,3 @@
+_ = some_func()
`)
	got := SLP120{}.Check(d)
	if len(got) != 0 {
		t.Fatalf("expected 0 findings for .py file, got %d", len(got))
	}
}

func TestSLP120_Description(t *testing.T) {
	r := SLP120{}
	if r.ID() != "SLP120" {
		t.Errorf("ID = %q", r.ID())
	}
	if r.Description() == "" {
		t.Errorf("Description is empty")
	}
	if r.DefaultSeverity() != SeverityWarn {
		t.Errorf("default severity should be warn")
	}
}
