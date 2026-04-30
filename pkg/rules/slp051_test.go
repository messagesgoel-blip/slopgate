package rules

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestSLP051_FiresOnUndefinedCall(t *testing.T) {
	d := parseDiff(t, `diff --git a/a/foo.go b/a/foo.go
--- a/a/foo.go
+++ b/a/foo.go
@@ -1,2 +1,4 @@
 package a

+func Run() {
+	undefinedHelper()
+}
`)
	got := SLP051{}.Check(d)
	if len(got) != 1 {
		t.Fatalf("expected 1 finding, got %d: %+v", len(got), got)
	}
	if !strings.Contains(got[0].Message, "undefinedHelper") {
		t.Errorf("expected message to mention undefinedHelper, got %q", got[0].Message)
	}
}

func TestSLP051_IgnoresDefinedCall(t *testing.T) {
	d := parseDiff(t, `diff --git a/a/foo.go b/a/foo.go
--- a/a/foo.go
+++ b/a/foo.go
@@ -1,2 +1,8 @@
 package a

+func helper() {}
+
+func Run() {
+	helper()
+}
`)
	got := SLP051{}.Check(d)
	if len(got) != 0 {
		t.Fatalf("expected 0 findings, got %d: %+v", len(got), got)
	}
}

func TestSLP051_IgnoresBuiltins(t *testing.T) {
	d := parseDiff(t, `diff --git a/a/foo.go b/a/foo.go
--- a/a/foo.go
+++ b/a/foo.go
@@ -1,2 +1,5 @@
 package a

+func Run() {
+	make([]int, 10)
+}
`)
	got := SLP051{}.Check(d)
	if len(got) != 0 {
		t.Fatalf("expected 0 findings for builtins, got %d: %+v", len(got), got)
	}
}

func TestSLP051_IgnoresMethodCalls(t *testing.T) {
	d := parseDiff(t, `diff --git a/a/foo.go b/a/foo.go
--- a/a/foo.go
+++ b/a/foo.go
@@ -1,2 +1,5 @@
 package a

+func Run() {
+	srv.DoThing()
+}
`)
	got := SLP051{}.Check(d)
	if len(got) != 0 {
		t.Fatalf("expected 0 findings for method calls, got %d: %+v", len(got), got)
	}
}

func TestSLP051_IgnoresCallsInsideCommentsAndStrings(t *testing.T) {
	d := parseDiff(t, `diff --git a/a/foo.go b/a/foo.go
--- a/a/foo.go
+++ b/a/foo.go
@@ -1,2 +1,6 @@
 package a

+func Run() {
+	// helper()
+	println("helper()")
+}
`)
	got := SLP051{}.Check(d)
	if len(got) != 0 {
		t.Fatalf("expected 0 findings for comments/strings, got %d: %+v", len(got), got)
	}
}

func TestSLP051_IgnoresGenericLocalFunctions(t *testing.T) {
	d := parseDiff(t, `diff --git a/a/foo.go b/a/foo.go
--- a/a/foo.go
+++ b/a/foo.go
@@ -1,2 +1,7 @@
 package a

+func helper[T any](v T) T { return v }
+
+func Run() {
+	helper(1)
+}
`)
	got := SLP051{}.Check(d)
	if len(got) != 0 {
		t.Fatalf("expected 0 findings for generic local function, got %d: %+v", len(got), got)
	}
}

func TestSLP051_IgnoresPredeclaredTypeConversions(t *testing.T) {
	d := parseDiff(t, `diff --git a/a/foo.go b/a/foo.go
--- a/a/foo.go
+++ b/a/foo.go
@@ -1,2 +1,5 @@
 package a

+func Run(v []byte) {
+	_ = string(v)
+}
`)
	got := SLP051{}.Check(d)
	if len(got) != 0 {
		t.Fatalf("expected 0 findings for string conversion, got %d: %+v", len(got), got)
	}
}

func TestSLP051_IgnoresLocalTypeConversions(t *testing.T) {
	d := parseDiff(t, `diff --git a/a/foo.go b/a/foo.go
--- a/a/foo.go
+++ b/a/foo.go
@@ -1,2 +1,7 @@
 package a

+type Status string
+
+func Run(v string) {
+	_ = Status(v)
+}
`)
	got := SLP051{}.Check(d)
	if len(got) != 0 {
		t.Fatalf("expected 0 findings for local type conversion, got %d: %+v", len(got), got)
	}
}

func TestSLP051_IgnoresPackageGroupedTypeConversions(t *testing.T) {
	cwd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	tmp := t.TempDir()
	if err := os.Chdir(tmp); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := os.Chdir(cwd); err != nil {
			t.Fatalf("restore cwd: %v", err)
		}
	})
	if err := os.MkdirAll("a", 0o750); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile("a/types.go", []byte("package a\n\ntype (\n\tStatus string\n\tCode int\n)\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	d := parseDiff(t, `diff --git a/a/foo.go b/a/foo.go
--- a/a/foo.go
+++ b/a/foo.go
@@ -1,2 +1,5 @@
 package a

+func Run(v string) {
+	_ = Status(v)
+}
`)
	got := SLP051{}.Check(d)
	if len(got) != 0 {
		t.Fatalf("expected 0 findings for package grouped type conversion, got %d: %+v", len(got), got)
	}
}

func TestSLP051_IgnoresPackageLocalHelpers(t *testing.T) {
	cwd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	tmp := t.TempDir()
	if err := os.Chdir(tmp); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := os.Chdir(cwd); err != nil {
			t.Fatalf("restore cwd: %v", err)
		}
	})
	if err := os.MkdirAll("a", 0o750); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile("a/helpers.go", []byte("package a\n\nfunc helper() {}\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	d := parseDiff(t, `diff --git a/a/foo.go b/a/foo.go
--- a/a/foo.go
+++ b/a/foo.go
@@ -1,2 +1,5 @@
 package a

+func Run() {
+	helper()
+}
`)
	got := SLP051{}.Check(d)
	if len(got) != 0 {
		t.Fatalf("expected 0 findings for package-local helper, got %d: %+v", len(got), got)
	}
}

func TestSLP051_DoesNotUseSymlinkedHelpersOutsideRepo(t *testing.T) {
	cwd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	tmp := t.TempDir()
	if err := os.Chdir(tmp); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := os.Chdir(cwd); err != nil {
			t.Fatalf("restore cwd: %v", err)
		}
	})
	if err := os.MkdirAll("a", 0o750); err != nil {
		t.Fatal(err)
	}
	outside := t.TempDir()
	outsideFile := filepath.Join(outside, "helpers.go")
	if err := os.WriteFile(outsideFile, []byte("package a\n\nfunc externalHelper() {}\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(outsideFile, filepath.Join("a", "helpers.go")); err != nil {
		t.Skipf("symlink unavailable: %v", err)
	}
	d := parseDiff(t, `diff --git a/a/foo.go b/a/foo.go
--- a/a/foo.go
+++ b/a/foo.go
@@ -1,2 +1,5 @@
 package a

+func Run() {
+	externalHelper()
+}
`)
	got := SLP051{}.Check(d)
	if len(got) != 1 {
		t.Fatalf("expected 1 finding for helper symlinked outside repo, got %d: %+v", len(got), got)
	}
}

func TestSLP051_DoesNotUseTestOnlyPackageHelpers(t *testing.T) {
	cwd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	tmp := t.TempDir()
	if err := os.Chdir(tmp); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := os.Chdir(cwd); err != nil {
			t.Fatalf("restore cwd: %v", err)
		}
	})
	if err := os.MkdirAll("a", 0o750); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile("a/helpers_test.go", []byte("package a\n\nfunc testOnlyHelper() {}\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	d := parseDiff(t, `diff --git a/a/foo.go b/a/foo.go
--- a/a/foo.go
+++ b/a/foo.go
@@ -1,2 +1,5 @@
 package a

+func Run() {
+	testOnlyHelper()
+}
`)
	got := SLP051{}.Check(d)
	if len(got) != 1 {
		t.Fatalf("expected 1 finding for test-only helper, got %d: %+v", len(got), got)
	}
}

func TestSLP051ResolvePackageDirRejectsEscapes(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "pkg"), 0o750); err != nil {
		t.Fatal(err)
	}
	if _, ok := slp051ResolvePackageDir(root, "../outside"); ok {
		t.Fatal("expected parent traversal to be rejected")
	}
	if _, ok := slp051ResolvePackageDir(root, filepath.ToSlash(filepath.Join(root, "pkg"))); ok {
		t.Fatal("expected absolute path to be rejected")
	}
	got, ok := slp051ResolvePackageDir(root, "a/../pkg")
	if !ok {
		t.Fatal("expected normalized repo-relative path to be accepted")
	}
	want, err := filepath.EvalSymlinks(filepath.Join(root, "pkg"))
	if err != nil {
		t.Fatal(err)
	}
	if got != want {
		t.Fatalf("resolved path = %q, want %q", got, want)
	}
}

func TestSLP051ResolvePackageDirAcceptsInternalSymlink(t *testing.T) {
	root := t.TempDir()
	realDir := filepath.Join(root, "realpkg")
	if err := os.MkdirAll(realDir, 0o750); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(realDir, filepath.Join(root, "pkg")); err != nil {
		t.Skipf("symlink unavailable: %v", err)
	}
	got, ok := slp051ResolvePackageDir(root, "pkg")
	if !ok {
		t.Fatal("expected internal symlink to be accepted")
	}
	want, err := filepath.EvalSymlinks(realDir)
	if err != nil {
		t.Fatal(err)
	}
	if got != want {
		t.Fatalf("resolved path = %q, want %q", got, want)
	}
}

func TestSLP051ResolvePackageDirRejectsExternalSymlink(t *testing.T) {
	root := t.TempDir()
	outside := t.TempDir()
	if err := os.Symlink(outside, filepath.Join(root, "pkg")); err != nil {
		t.Skipf("symlink unavailable: %v", err)
	}
	if _, ok := slp051ResolvePackageDir(root, "pkg"); ok {
		t.Fatal("expected symlink outside repo to be rejected")
	}
}

func TestSLP051_IgnoresGoDeclarations(t *testing.T) {
	d := parseDiff(t, `diff --git a/a/foo.go b/a/foo.go
--- a/a/foo.go
+++ b/a/foo.go
@@ -1,2 +1,7 @@
 package a

+import (
+	"fmt"
+)
+var (
+	value = fmt.Sprintf("%s", "ok")
+)
`)
	got := SLP051{}.Check(d)
	if len(got) != 0 {
		t.Fatalf("expected 0 findings for Go declarations, got %d: %+v", len(got), got)
	}
}

func TestSLP051_Meta(t *testing.T) {
	r := SLP051{}
	if r.ID() != "SLP051" {
		t.Errorf("ID = %q", r.ID())
	}
	if r.Description() == "" {
		t.Errorf("Description is empty")
	}
	if r.DefaultSeverity() != SeverityBlock {
		t.Errorf("DefaultSeverity = %v, want SeverityBlock", r.DefaultSeverity())
	}
}
