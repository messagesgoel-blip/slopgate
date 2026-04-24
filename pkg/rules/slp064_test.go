package rules

import (
	"strings"
	"testing"
)

func TestSLP064_FiresOnMockWithoutAssertion(t *testing.T) {
	d := parseDiff(t, `diff --git a/foo_test.go b/foo_test.go
--- a/foo_test.go
+++ b/foo_test.go
@@ -1,1 +1,7 @@
 package foo
+func TestFoo(t *testing.T) {
+	m := mock.NewSomething()
+	m.Do()
+	_ = m
+}
`)
	got := SLP064{}.Check(d)
	if len(got) != 1 {
		t.Fatalf("expected 1 finding, got %d: %+v", len(got), got)
	}
	if !strings.Contains(got[0].Message, "mocks present but no assertions") {
		t.Errorf("message should mention mocks: %q", got[0].Message)
	}
}

func TestSLP064_NoFireWhenAssertionPresent(t *testing.T) {
	d := parseDiff(t, `diff --git a/foo_test.go b/foo_test.go
--- a/foo_test.go
+++ b/foo_test.go
@@ -1,1 +1,7 @@
 package foo
+func TestFoo(t *testing.T) {
+	m := mock.NewSomething()
+	m.Do()
+	assert.Equal(t, 1, m.Count)
+}
`)
	got := SLP064{}.Check(d)
	if len(got) != 0 {
		t.Fatalf("expected 0 findings, got %d: %+v", len(got), got)
	}
}

func TestSLP064_NoFireWithoutMocks(t *testing.T) {
	d := parseDiff(t, `diff --git a/foo_test.go b/foo_test.go
--- a/foo_test.go
+++ b/foo_test.go
@@ -1,1 +1,6 @@
 package foo
+func TestFoo(t *testing.T) {
+	x := 1
+	assert.Equal(t, 1, x)
+}
`)
	got := SLP064{}.Check(d)
	if len(got) != 0 {
		t.Fatalf("expected 0 findings, got %d: %+v", len(got), got)
	}
}

func TestSLP064_FiresOnGomock(t *testing.T) {
	d := parseDiff(t, `diff --git a/foo_test.go b/foo_test.go
--- a/foo_test.go
+++ b/foo_test.go
@@ -1,1 +1,6 @@
 package foo
+func TestFoo(t *testing.T) {
+	ctrl := gomock.NewController(t)
+	defer ctrl.Finish()
+}
`)
	got := SLP064{}.Check(d)
	if len(got) != 1 {
		t.Fatalf("expected 1 finding, got %d: %+v", len(got), got)
	}
}

func TestSLP064_IgnoresNonTestFiles(t *testing.T) {
	d := parseDiff(t, `diff --git a/foo.go b/foo.go
--- a/foo.go
+++ b/foo.go
@@ -1,1 +1,4 @@
 package foo
+func Bar() {
+	mock.Setup()
+}
`)
	got := SLP064{}.Check(d)
	if len(got) != 0 {
		t.Fatalf("expected 0 findings, got %d: %+v", len(got), got)
	}
}

func TestSLP064_Description(t *testing.T) {
	r := SLP064{}
	if r.ID() != "SLP064" {
		t.Errorf("ID = %q", r.ID())
	}
	if r.Description() == "" {
		t.Errorf("Description is empty")
	}
	if r.DefaultSeverity() != SeverityWarn {
		t.Errorf("default severity should be warn")
	}
}
