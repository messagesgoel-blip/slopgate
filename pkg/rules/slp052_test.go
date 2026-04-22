package rules

import (
	"strings"
	"testing"
)

func TestSLP052_FiresOnProdDeleteWithTestAdd(t *testing.T) {
	d := parseDiff(t, `diff --git a/a/foo.go b/a/foo.go
--- a/a/foo.go
+++ b/a/foo.go
@@ -1,5 +1,2 @@
 package a

-func OldFeature() {
-	return 42
-}

diff --git a/a/foo_test.go b/a/foo_test.go
--- a/a/foo_test.go
+++ b/a/foo_test.go
@@ -1,2 +1,5 @@
 package a

+func TestNewThing(t *testing.T) {
+	NewThing()
+}
`)
	got := SLP052{}.Check(d)
	if len(got) != 1 {
		t.Fatalf("expected 1 finding, got %d: %+v", len(got), got)
	}
}

func TestSLP052_IgnoresWhenNoTestChanges(t *testing.T) {
	d := parseDiff(t, `diff --git a/a/foo.go b/a/foo.go
--- a/a/foo.go
+++ b/a/foo.go
@@ -1,5 +1,2 @@
 package a

-func OldFeature() {
-	return 42
-}
`)
	got := SLP052{}.Check(d)
	if len(got) != 0 {
		t.Fatalf("expected 0 findings, got %d: %+v", len(got), got)
	}
}

func TestSLP052_IgnoresSmallDelete(t *testing.T) {
	d := parseDiff(t, `diff --git a/a/foo.go b/a/foo.go
--- a/a/foo.go
+++ b/a/foo.go
@@ -1,3 +1,2 @@
 package a

-func OldFeature() {}

diff --git a/a/foo_test.go b/a/foo_test.go
--- a/a/foo_test.go
+++ b/a/foo_test.go
@@ -1,2 +1,3 @@
 package a

+func TestNewThing(t *testing.T) {}
`)
	got := SLP052{}.Check(d)
	if len(got) != 0 {
		t.Fatalf("expected 0 findings for <3 deleted lines, got %d: %+v", len(got), got)
	}
}

func TestSLP052_Description(t *testing.T) {
	r := SLP052{}
	if r.ID() != "SLP052" {
		t.Errorf("ID = %q", r.ID())
	}
	if !strings.Contains(r.Description(), "production code") {
		t.Errorf("description should mention production code")
	}
}
