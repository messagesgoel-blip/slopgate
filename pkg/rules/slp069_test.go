package rules

import (
	"strings"
	"testing"
)

func TestSLP069_FiresWhenMixedNamingInPackage(t *testing.T) {
	d := parseDiff(t, `diff --git a/pkg/foo/legacy.go b/pkg/foo/legacy.go
--- a/pkg/foo/legacy.go
+++ b/pkg/foo/legacy.go
@@ -1,1 +1,3 @@
 package foo
+
+var user_name = "test"
+func fetch_data() {}
diff --git a/pkg/foo/modern.go b/pkg/foo/modern.go
--- a/pkg/foo/modern.go
+++ b/pkg/foo/modern.go
@@ -1,1 +1,2 @@
 package foo
+
+func FetchData() {}
`)
	got := SLP069{}.Check(d)
	if len(got) != 1 {
		t.Fatalf("expected 1 finding, got %d: %+v", len(got), got)
	}
	if got[0].File != "pkg/foo/legacy.go" {
		t.Errorf("file: %q", got[0].File)
	}
	if !strings.Contains(got[0].Message, "mixed naming conventions") {
		t.Errorf("message: %q", got[0].Message)
	}
}

func TestSLP069_NoFireWhenOnlySnakeCase(t *testing.T) {
	d := parseDiff(t, `diff --git a/pkg/foo/legacy.go b/pkg/foo/legacy.go
--- a/pkg/foo/legacy.go
+++ b/pkg/foo/legacy.go
@@ -1,1 +1,2 @@
 package foo
+
+var user_name = "test"
`)
	got := SLP069{}.Check(d)
	if len(got) != 0 {
		t.Fatalf("expected 0 findings, got %d: %+v", len(got), got)
	}
}

func TestSLP069_NoFireWhenOnlyCamelCase(t *testing.T) {
	d := parseDiff(t, `diff --git a/pkg/foo/modern.go b/pkg/foo/modern.go
--- a/pkg/foo/modern.go
+++ b/pkg/foo/modern.go
@@ -1,1 +1,2 @@
 package foo
+
+func FetchData() {}
`)
	got := SLP069{}.Check(d)
	if len(got) != 0 {
		t.Fatalf("expected 0 findings, got %d: %+v", len(got), got)
	}
}

func TestSLP069_NoFireInTestFiles(t *testing.T) {
	d := parseDiff(t, `diff --git a/pkg/foo/legacy_test.go b/pkg/foo/legacy_test.go
--- a/pkg/foo/legacy_test.go
+++ b/pkg/foo/legacy_test.go
@@ -1,1 +1,2 @@
 package foo
+
+var test_user_name = "test"
diff --git a/pkg/foo/modern.go b/pkg/foo/modern.go
--- a/pkg/foo/modern.go
+++ b/pkg/foo/modern.go
@@ -1,1 +1,2 @@
 package foo
+
+func FetchData() {}
`)
	got := SLP069{}.Check(d)
	if len(got) != 0 {
		t.Fatalf("expected 0 findings because test files are ignored, got %d: %+v", len(got), got)
	}
}

func TestSLP069_Meta(t *testing.T) {
	r := SLP069{}
	if r.ID() != "SLP069" {
		t.Errorf("ID = %q", r.ID())
	}
	if r.Description() == "" {
		t.Errorf("Description is empty")
	}
	if r.DefaultSeverity() != SeverityInfo {
		t.Errorf("default severity should be info")
	}
}
