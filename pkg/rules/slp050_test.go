package rules

import (
	"strings"
	"testing"
)

func TestSLP050_FiresOnPointerParam(t *testing.T) {
	d := parseDiff(t, `diff --git a/a.go b/a.go
--- a/a.go
+++ b/a.go
@@ -1,1 +1,4 @@
 package a
+
+func Do(x *int) int {
+	return *x
+}
`)
	got := SLP050{}.Check(d)
	if len(got) != 1 {
		t.Fatalf("expected 1 finding, got %d: %+v", len(got), got)
	}
	if !strings.Contains(got[0].Message, "Do") {
		t.Errorf("message should mention Do: %q", got[0].Message)
	}
}

func TestSLP050_IgnoresWhenValidated(t *testing.T) {
	d := parseDiff(t, `diff --git a/a.go b/a.go
--- a/a.go
+++ b/a.go
@@ -1,1 +1,5 @@
 package a
+
+func Do(x *int) int {
+	if x == nil { return 0 }
+	return *x
+}
`)
	got := SLP050{}.Check(d)
	if len(got) != 0 {
		t.Fatalf("expected 0 findings, got %d: %+v", len(got), got)
	}
}

func TestSLP050_FiresOnSlice(t *testing.T) {
	d := parseDiff(t, `diff --git a/a.go b/a.go
--- a/a.go
+++ b/a.go
@@ -1,1 +1,4 @@
 package a
+
+func Do(x []int) int {
+	return x[0]
+}
`)
	got := SLP050{}.Check(d)
	if len(got) != 1 {
		t.Fatalf("expected 1 finding, got %d: %+v", len(got), got)
	}
}

func TestSLP050_FiresOnString(t *testing.T) {
	d := parseDiff(t, `diff --git a/a.go b/a.go
--- a/a.go
+++ b/a.go
@@ -1,1 +1,4 @@
 package a
+
+func Do(s string) string {
+	return s + "!"
+}
`)
	got := SLP050{}.Check(d)
	if len(got) != 1 {
		t.Fatalf("expected 1 finding, got %d: %+v", len(got), got)
	}
}

func TestSLP050_IgnoresValueType(t *testing.T) {
	d := parseDiff(t, `diff --git a/a.go b/a.go
--- a/a.go
+++ b/a.go
@@ -1,1 +1,4 @@
 package a
+
+func Do(x int) int {
+	return x + 1
+}
`)
	got := SLP050{}.Check(d)
	if len(got) != 0 {
		t.Fatalf("expected 0 findings for value type, got %d: %+v", len(got), got)
	}
}

func TestSLP050_Description(t *testing.T) {
	r := SLP050{}
	if r.ID() != "SLP050" {
		t.Errorf("ID = %q", r.ID())
	}
	if r.Description() == "" {
		t.Errorf("Description is empty")
	}
	if r.DefaultSeverity() != SeverityWarn {
		t.Errorf("default severity should be warn")
	}
}
