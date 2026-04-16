package rules

import "testing"

func TestSLP001_FiresOnGoTestWithNoAssertion(t *testing.T) {
	d := parseDiff(t, `diff --git a/a/foo_test.go b/a/foo_test.go
--- a/a/foo_test.go
+++ b/a/foo_test.go
@@ -1,2 +1,6 @@
 package a

+func TestFoo(t *testing.T) {
+	Foo()
+	Bar()
+}
`)
	got := SLP001{}.Check(d)
	if len(got) != 1 {
		t.Fatalf("expected 1 finding, got %d: %+v", len(got), got)
	}
	if got[0].File != "a/foo_test.go" {
		t.Errorf("file = %q", got[0].File)
	}
}

func TestSLP001_IgnoresGoTestWithTError(t *testing.T) {
	d := parseDiff(t, `diff --git a/a/foo_test.go b/a/foo_test.go
--- a/a/foo_test.go
+++ b/a/foo_test.go
@@ -1,2 +1,6 @@
 package a

+func TestFoo(t *testing.T) {
+	if got := Foo(); got != 42 {
+		t.Errorf("Foo() = %d, want 42", got)
+	}
+}
`)
	got := SLP001{}.Check(d)
	if len(got) != 0 {
		t.Fatalf("expected 0 findings, got %d: %+v", len(got), got)
	}
}

func TestSLP001_IgnoresGoTestWithTFatal(t *testing.T) {
	d := parseDiff(t, `diff --git a/a/foo_test.go b/a/foo_test.go
--- a/a/foo_test.go
+++ b/a/foo_test.go
@@ -1,2 +1,5 @@
 package a

+func TestFoo(t *testing.T) {
+	_, err := Foo()
+	if err != nil { t.Fatal(err) }
+}
`)
	got := SLP001{}.Check(d)
	if len(got) != 0 {
		t.Fatalf("expected 0 findings (t.Fatal is an assertion), got %d", len(got))
	}
}

func TestSLP001_IgnoresGoTestWithAssertLibrary(t *testing.T) {
	d := parseDiff(t, `diff --git a/a/foo_test.go b/a/foo_test.go
--- a/a/foo_test.go
+++ b/a/foo_test.go
@@ -1,2 +1,5 @@
 package a

+func TestFoo(t *testing.T) {
+	assert.Equal(t, 42, Foo())
+}
`)
	got := SLP001{}.Check(d)
	if len(got) != 0 {
		t.Fatalf("expected 0 findings (assert.Equal), got %d", len(got))
	}
}

func TestSLP001_IgnoresNonTestFiles(t *testing.T) {
	// A function named Test* outside _test.go is production code, not a test.
	d := parseDiff(t, `diff --git a/a/foo.go b/a/foo.go
--- a/a/foo.go
+++ b/a/foo.go
@@ -1,2 +1,5 @@
 package a

+func TestableHelper(x int) int {
+	return x * 2
+}
`)
	got := SLP001{}.Check(d)
	if len(got) != 0 {
		t.Fatalf("expected 0 findings in non-_test.go, got %d", len(got))
	}
}

func TestSLP001_IgnoresTestHelperFunctions(t *testing.T) {
	// Helper functions in test files that are not named Test* should
	// not fire. Only actual test entrypoints are checked.
	d := parseDiff(t, `diff --git a/a/foo_test.go b/a/foo_test.go
--- a/a/foo_test.go
+++ b/a/foo_test.go
@@ -1,2 +1,5 @@
 package a

+func makeFixture() *Thing {
+	return &Thing{}
+}
`)
	got := SLP001{}.Check(d)
	if len(got) != 0 {
		t.Fatalf("expected 0 findings for helper, got %d", len(got))
	}
}

func TestSLP001_FiresWhenTestCallsFunctionButIgnoresResult(t *testing.T) {
	// Classic AI slop: the test calls the function but does not
	// assert on the return value.
	d := parseDiff(t, `diff --git a/a/foo_test.go b/a/foo_test.go
--- a/a/foo_test.go
+++ b/a/foo_test.go
@@ -1,2 +1,5 @@
 package a

+func TestReturnsValue(t *testing.T) {
+	_ = ComputeSomething(1, 2, 3)
+}
`)
	got := SLP001{}.Check(d)
	if len(got) != 1 {
		t.Fatalf("expected 1 finding, got %d", len(got))
	}
}

func TestSLP001_IgnoresSkippedScaffoldTest(t *testing.T) {
	// Real false positive: a test that unconditionally skips is an
	// intentional scaffold, not slop. The skip itself signals "this
	// isn't asserting anything yet".
	d := parseDiff(t, `diff --git a/a/drill_test.go b/a/drill_test.go
--- a/a/drill_test.go
+++ b/a/drill_test.go
@@ -1,2 +1,7 @@
 package a

+func TestDrill_Scaffold(t *testing.T) {
+	t.Skip("drill harness not yet wired — scaffold only")
+
+	DoThing()
+}
`)
	got := SLP001{}.Check(d)
	if len(got) != 0 {
		t.Fatalf("expected 0 findings on skipped scaffold, got %d: %+v", len(got), got)
	}
}

func TestSLP001_FiresEvenIfSkipIsInConditionalBranch(t *testing.T) {
	// t.Skip inside an if is NOT an exemption — the test can still run
	// along the other branch and must assert then.
	d := parseDiff(t, `diff --git a/a/foo_test.go b/a/foo_test.go
--- a/a/foo_test.go
+++ b/a/foo_test.go
@@ -1,2 +1,7 @@
 package a

+func TestFoo(t *testing.T) {
+	if os.Getenv("CI") != "" {
+		t.Skip("skip in CI")
+	}
+	_ = DoThing()
+}
`)
	got := SLP001{}.Check(d)
	if len(got) != 1 {
		t.Fatalf("expected 1 finding (conditional skip), got %d: %+v", len(got), got)
	}
}

func TestSLP001_Description(t *testing.T) {
	r := SLP001{}
	if r.ID() != "SLP001" {
		t.Errorf("ID = %q", r.ID())
	}
}
