package rules

import (
	"strings"
	"testing"
)

func TestSLP048_FiresOnInconsistentErrorHandling(t *testing.T) {
	d := parseDiff(t, `diff --git a/pkg/a.go b/pkg/a.go
--- a/pkg/a.go
+++ b/pkg/a.go
@@ -1,1 +1,5 @@
 package pkg
+
+func Foo() error {
+	return nil
+}
diff --git a/pkg/b.go b/pkg/b.go
--- a/pkg/b.go
+++ b/pkg/b.go
@@ -1,1 +1,5 @@
 package pkg
+
+func Bar() error {
+	if err != nil { return err }
+	return nil
+}
`)
	got := SLP048{}.Check(d)
	if len(got) != 1 {
		t.Fatalf("expected 1 finding, got %d: %+v", len(got), got)
	}
	if !strings.Contains(got[0].File, "a.go") {
		t.Errorf("expected finding for a.go, got %q", got[0].File)
	}
}

func TestSLP048_IgnoresConsistentErrorHandling(t *testing.T) {
	d := parseDiff(t, `diff --git a/pkg/a.go b/pkg/a.go
--- a/pkg/a.go
+++ b/pkg/a.go
@@ -1,1 +1,5 @@
 package pkg
+
+func Foo() error {
+	if err != nil { return err }
+	return nil
+}
diff --git a/pkg/b.go b/pkg/b.go
--- a/pkg/b.go
+++ b/pkg/b.go
@@ -1,1 +1,5 @@
 package pkg
+
+func Bar() error {
+	if err != nil { return err }
+	return nil
+}
`)
	got := SLP048{}.Check(d)
	if len(got) != 0 {
		t.Fatalf("expected 0 findings for consistent handling, got %d: %+v", len(got), got)
	}
}

func TestSLP048_IgnoresSingleFile(t *testing.T) {
	d := parseDiff(t, `diff --git a/pkg/a.go b/pkg/a.go
--- a/pkg/a.go
+++ b/pkg/a.go
@@ -1,1 +1,5 @@
 package pkg
+
+func Foo() error {
+	return nil
+}
`)
	got := SLP048{}.Check(d)
	if len(got) != 0 {
		t.Fatalf("expected 0 findings for single file, got %d: %+v", len(got), got)
	}
}

func TestSLP048_IgnoresDifferentPackagesInSameDirectory(t *testing.T) {
	d := parseDiff(t, `diff --git a/service/foo.go b/service/foo.go
--- a/service/foo.go
+++ b/service/foo.go
@@ -1,1 +1,5 @@
+package foo
+
+func A() error {
+	return nil
+}
diff --git a/service/bar.go b/service/bar.go
--- a/service/bar.go
+++ b/service/bar.go
@@ -1,1 +1,6 @@
+package bar
+
+func B() error {
+	if err != nil { return err }
+	return nil
+}
`)
	got := SLP048{}.Check(d)
	if len(got) != 0 {
		t.Fatalf("expected 0 findings for different packages in same dir, got %d: %+v", len(got), got)
	}
}

func TestSLP048_FindsMultilineGenericErrorReturn(t *testing.T) {
	d := parseDiff(t, `diff --git a/pkg/a.go b/pkg/a.go
--- a/pkg/a.go
+++ b/pkg/a.go
@@ -1,1 +1,8 @@
 package pkg
+
+func Load[T any](
+	id string,
+) (T, error) {
+	var zero T
+	return zero, nil
+}
diff --git a/pkg/b.go b/pkg/b.go
--- a/pkg/b.go
+++ b/pkg/b.go
@@ -1,1 +1,6 @@
 package pkg
+
+func Save() error {
+	if err != nil { return err }
+	return nil
+}
`)
	got := SLP048{}.Check(d)
	if len(got) != 1 {
		t.Fatalf("expected 1 finding for multiline generic error-return function, got %d: %+v", len(got), got)
	}
	if got[0].File != "pkg/a.go" {
		t.Fatalf("expected finding for pkg/a.go, got %+v", got[0])
	}
}

func TestSLP048_Description(t *testing.T) {
	r := SLP048{}
	if r.ID() != "SLP048" {
		t.Errorf("ID = %q", r.ID())
	}
	if r.Description() == "" {
		t.Errorf("Description is empty")
	}
	if r.DefaultSeverity() != SeverityWarn {
		t.Errorf("default severity should be warn")
	}
}
