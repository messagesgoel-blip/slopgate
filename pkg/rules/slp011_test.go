package rules

import "testing"

func TestSLP011_FiresOnGoTestWithOnlyAssertion(t *testing.T) {
	// Classic AI slop: test body is entirely assertion calls with no arrange.
	// The test compiles, passes, but tests nothing meaningful.
	d := parseDiff(t, `diff --git a/a/foo_test.go b/a/foo_test.go
--- a/a/foo_test.go
+++ b/a/foo_test.go
@@ -1,2 +1,5 @@
 package a

+func TestFoo(t *testing.T) {
+	assert.Equal(t, 1, 1)
+}
`)
	got := SLP011{}.Check(d)
	if len(got) != 1 {
		t.Fatalf("expected 1 finding, got %d: %+v", len(got), got)
	}
	if got[0].File != "a/foo_test.go" {
		t.Errorf("file = %q", got[0].File)
	}
	// Line 3 is the function signature (where the finding is reported)
	if got[0].Line != 3 {
		t.Errorf("line = %d, want 3", got[0].Line)
	}
}

func TestSLP011_FiresOnGoTestWithMultipleAssertionsOnly(t *testing.T) {
	// Multiple assertions but still no arrange - slop.
	d := parseDiff(t, `diff --git a/a/foo_test.go b/a/foo_test.go
--- a/a/foo_test.go
+++ b/a/foo_test.go
@@ -1,2 +1,7 @@
 package a

+func TestBar(t *testing.T) {
+	assert.Equal(t, 1, 1)
+	assert.NotNil(t, nil)
+	require.True(t, true)
+}
`)
	got := SLP011{}.Check(d)
	if len(got) != 1 {
		t.Fatalf("expected 1 finding, got %d: %+v", len(got), got)
	}
}

func TestSLP011_IgnoresGoTestWithArrange(t *testing.T) {
	// got := Foo() is the arrange; assert checks it - this is proper test structure.
	d := parseDiff(t, `diff --git a/a/foo_test.go b/a/foo_test.go
--- a/a/foo_test.go
+++ b/a/foo_test.go
@@ -1,2 +1,6 @@
 package a

+func TestFoo(t *testing.T) {
+	if got := Foo(); assert.Equal(t, 1, got) {
+	}
+}
`)
	got := SLP011{}.Check(d)
	if len(got) != 0 {
		t.Fatalf("expected 0 findings (has arrange), got %d: %+v", len(got), got)
	}
}

func TestSLP011_IgnoresGoTestWithSeparateVariableDeclaration(t *testing.T) {
	// Variable declaration before assertion is the arrange pattern.
	d := parseDiff(t, `diff --git a/a/foo_test.go b/a/foo_test.go
--- a/a/foo_test.go
+++ b/a/foo_test.go
@@ -1,2 +1,6 @@
 package a

+func TestFoo(t *testing.T) {
+	got := Foo()
+	assert.Equal(t, 1, got)
+}
`)
	got := SLP011{}.Check(d)
	if len(got) != 0 {
		t.Fatalf("expected 0 findings (has arrange), got %d: %+v", len(got), got)
	}
}

func TestSLP011_IgnoresGoTestWithTFatalError(t *testing.T) {
	// t.Error/t.Fatal are also assertions - this test has logic.
	d := parseDiff(t, `diff --git a/a/foo_test.go b/a/foo_test.go
--- a/a/foo_test.go
+++ b/a/foo_test.go
@@ -1,2 +1,5 @@
 package a

+func TestFoo(t *testing.T) {
+	if got := Foo(); got != 42 { t.Errorf("got %d", got) }
+}
`)
	got := SLP011{}.Check(d)
	if len(got) != 0 {
		t.Fatalf("expected 0 findings (has t.Error), got %d: %+v", len(got), got)
	}
}

func TestSLP011_IgnoresSafetyTestNames(t *testing.T) {
	// NilSafe/NoPanic tests are intentionally minimal.
	d := parseDiff(t, `diff --git a/a/foo_test.go b/a/foo_test.go
--- a/a/foo_test.go
+++ b/a/foo_test.go
@@ -1,2 +1,5 @@
 package a

+func TestNilSafe(t *testing.T) {
+	assert.Nil(t, nil)
+}
`)
	got := SLP011{}.Check(d)
	if len(got) != 0 {
		t.Fatalf("expected 0 findings (safety test), got %d: %+v", len(got), got)
	}
}

func TestSLP011_IgnoresNonTestFiles(t *testing.T) {
	// Functions named Test* outside _test.go are not tests.
	d := parseDiff(t, `diff --git a/a/foo.go b/a/foo.go
--- a/a/foo.go
+++ b/a/foo.go
@@ -1,2 +1,5 @@
 package a

+func TestHelper() {
+	assert.True(t, true)
+}
`)
	got := SLP011{}.Check(d)
	if len(got) != 0 {
		t.Fatalf("expected 0 findings in non-_test.go, got %d", len(got))
	}
}

func TestSLP011_Description(t *testing.T) {
	r := SLP011{}
	if r.ID() != "SLP011" {
		t.Errorf("ID = %q", r.ID())
	}
	if r.Description() == "" {
		t.Errorf("Description is empty")
	}
}
