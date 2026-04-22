package rules

import (
	"strings"
	"testing"
)

func TestSLP046_FiresOnCrossPackageCall(t *testing.T) {
	d := parseDiff(t, `diff --git a/pkg/a/a.go b/pkg/a/a.go
--- a/pkg/a/a.go
+++ b/pkg/a/a.go
@@ -1,1 +1,4 @@
 package a
+
+func Helper() {
+}
+func Local() { Helper() }
diff --git a/pkg/b/b.go b/pkg/b/b.go
--- a/pkg/b/b.go
+++ b/pkg/b/b.go
@@ -1,1 +1,3 @@
 package b
+
+func Caller() { Helper() }
`)
	got := SLP046{}.Check(d)
	if len(got) != 2 {
		t.Fatalf("expected 2 findings, got %d: %+v", len(got), got)
	}
	files := map[string]bool{}
	for _, f := range got {
		files[f.File] = true
		if !strings.Contains(f.Message, "Helper") {
			t.Errorf("message should mention Helper: %q", f.Message)
		}
	}
	if !files["pkg/a/a.go"] {
		t.Errorf("expected finding for pkg/a/a.go")
	}
	if !files["pkg/b/b.go"] {
		t.Errorf("expected finding for pkg/b/b.go")
	}
}

func TestSLP046_IgnoresSamePackage(t *testing.T) {
	d := parseDiff(t, `diff --git a/pkg/a/a.go b/pkg/a/a.go
--- a/pkg/a/a.go
+++ b/pkg/a/a.go
@@ -1,1 +1,4 @@
 package a
+
+func Helper() {}
+func Local() { Helper() }
`)
	got := SLP046{}.Check(d)
	if len(got) != 0 {
		t.Fatalf("expected 0 findings for same-package call, got %d: %+v", len(got), got)
	}
}

func TestSLP046_IgnoresNoCalls(t *testing.T) {
	d := parseDiff(t, `diff --git a/pkg/a/a.go b/pkg/a/a.go
--- a/pkg/a/a.go
+++ b/pkg/a/a.go
@@ -1,1 +1,3 @@
 package a
+
+func Helper() {}
`)
	got := SLP046{}.Check(d)
	if len(got) != 0 {
		t.Fatalf("expected 0 findings when no cross-file calls, got %d: %+v", len(got), got)
	}
}

func TestSLP046_Description(t *testing.T) {
	r := SLP046{}
	if r.ID() != "SLP046" {
		t.Errorf("ID = %q", r.ID())
	}
	if r.Description() == "" {
		t.Errorf("Description is empty")
	}
	if r.DefaultSeverity() != SeverityWarn {
		t.Errorf("default severity should be warn")
	}
}
