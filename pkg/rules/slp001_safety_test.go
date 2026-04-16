package rules

import "testing"

func TestSLP001_IgnoresNilSafeTestName(t *testing.T) {
	d := parseDiff(t, `diff --git a/a/foo_test.go b/a/foo_test.go
--- a/a/foo_test.go
+++ b/a/foo_test.go
@@ -1,1 +1,5 @@
 package a
+func TestNotify_NilSafe(t *testing.T) {
+	var n *Notifier
+	n.Notify("hello")
+}
`)
	got := SLP001{}.Check(d)
	if len(got) != 0 {
		t.Fatalf("expected 0 findings for NilSafe test, got %d", len(got))
	}
}

func TestSLP001_IgnoresNoPanicTestName(t *testing.T) {
	d := parseDiff(t, `diff --git a/a/foo_test.go b/a/foo_test.go
--- a/a/foo_test.go
+++ b/a/foo_test.go
@@ -1,1 +1,5 @@
 package a
+func TestCompact_NoPanic(t *testing.T) {
+	Compact(nil)
+	Compact(make([]byte, 0))
+}
`)
	got := SLP001{}.Check(d)
	if len(got) != 0 {
		t.Fatalf("expected 0 findings for NoPanic test, got %d", len(got))
	}
}

func TestSLP001_DefaultSeverityIsWarn(t *testing.T) {
	r := SLP001{}
	if r.DefaultSeverity() != SeverityWarn {
		t.Errorf("SLP001 default severity should be warn, got %v", r.DefaultSeverity())
	}
}
