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
	if len(got) != 1 {
		t.Fatalf("expected 1 finding, got %d: %+v", len(got), got)
	}
	if !strings.Contains(got[0].Message, "Helper") {
		t.Errorf("message should mention Helper: %q", got[0].Message)
	}
	if !strings.Contains(got[0].Message, "pkg/a/a.go") {
		t.Errorf("message should mention pkg/a/a.go: %q", got[0].Message)
	}
	if !strings.Contains(got[0].Message, "pkg/b/b.go") {
		t.Errorf("message should mention pkg/b/b.go: %q", got[0].Message)
	}
	files := map[string]bool{}
	for _, f := range got {
		files[f.File] = true
	}
	if !files["pkg/a/a.go"] {
		t.Errorf("expected finding for pkg/a/a.go")
	}
}

func TestSLP046_IgnoresSamePackage(t *testing.T) {
	d := parseDiff(t, `diff --git a/pkg/a/helper.go b/pkg/a/helper.go
--- a/pkg/a/helper.go
+++ b/pkg/a/helper.go
@@ -1,1 +1,3 @@
 package a
+
+func Helper() {}
diff --git a/pkg/a/local.go b/pkg/a/local.go
--- a/pkg/a/local.go
+++ b/pkg/a/local.go
@@ -1,1 +1,3 @@
 package a
+
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

func TestSLP046_FiresWhenCallerChangeIsInExistingFunction(t *testing.T) {
	d := parseDiff(t, `diff --git a/pkg/a/helper.go b/pkg/a/helper.go
--- a/pkg/a/helper.go
+++ b/pkg/a/helper.go
@@ -1,1 +1,3 @@
 package a
+
+func Helper() {}
diff --git a/pkg/b/caller.go b/pkg/b/caller.go
--- a/pkg/b/caller.go
+++ b/pkg/b/caller.go
@@ -1,3 +1,4 @@
 package b
 func Caller() {
+	Helper()
 }
`)
	got := SLP046{}.Check(d)
	if len(got) != 1 {
		t.Fatalf("expected 1 finding for cross-file call from existing function body, got %d: %+v", len(got), got)
	}
}

func TestSLP046_FiresOnQualifiedCrossPackageCall(t *testing.T) {
	d := parseDiff(t, `diff --git a/pkg/a/helper.go b/pkg/a/helper.go
--- a/pkg/a/helper.go
+++ b/pkg/a/helper.go
@@ -1,1 +1,3 @@
 package a
+
+func Helper() {}
diff --git a/pkg/b/caller.go b/pkg/b/caller.go
--- a/pkg/b/caller.go
+++ b/pkg/b/caller.go
@@ -1,3 +1,4 @@
 package b
 func Caller() {
+	a.Helper()
 }
`)
	got := SLP046{}.Check(d)
	if len(got) != 1 {
		t.Fatalf("expected 1 finding for qualified cross-package call, got %d: %+v", len(got), got)
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
