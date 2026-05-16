package rules

import "testing"

func TestSLP010_FiresOnExistingTestWithNoAssertionInAddedLines(t *testing.T) {
	// Existing test TestFoo (signature is a context line), added line
	// `result := Foo()` with no assertion.
	d := parseDiff(t, `diff --git a/a/foo_test.go b/a/foo_test.go
--- a/a/foo_test.go
+++ b/a/foo_test.go
@@ -1,4 +1,5 @@
 package a
 func TestFoo(t *testing.T) {
+	result := Foo()
 }
`)
	got := SLP010{}.Check(d)
	if len(got) != 1 {
		t.Fatalf("expected 1 finding, got %d: %+v", len(got), got)
	}
	if got[0].File != "a/foo_test.go" {
		t.Errorf("file = %q", got[0].File)
	}
	if !containsStr(got[0].Message, "TestFoo") {
		t.Errorf("message should mention TestFoo: %q", got[0].Message)
	}
}

func TestSLP010_NoFindingWhenAddedLineHasAssertion(t *testing.T) {
	// Existing test with added line containing t.Errorf — has assertion.
	d := parseDiff(t, `diff --git a/a/foo_test.go b/a/foo_test.go
--- a/a/foo_test.go
+++ b/a/foo_test.go
@@ -1,4 +1,6 @@
 package a
 func TestFoo(t *testing.T) {
+	got := Foo()
+	t.Errorf("bad: %d", got)
 }
`)
	got := SLP010{}.Check(d)
	if len(got) != 0 {
		t.Fatalf("expected 0 findings (has assertion), got %d: %+v", len(got), got)
	}
}

func TestSLP010_NoFindingForEntirelyNewTest(t *testing.T) {
	// Entirely new test function (signature on added line) — SLP001 territory, not SLP010.
	d := parseDiff(t, `diff --git a/a/foo_test.go b/a/foo_test.go
--- a/a/foo_test.go
+++ b/a/foo_test.go
@@ -1,1 +1,4 @@
 package a
+func TestBar(t *testing.T) {
+	Bar()
+}
`)
	got := SLP010{}.Check(d)
	if len(got) != 0 {
		t.Fatalf("expected 0 findings for entirely new test (SLP001 territory), got %d: %+v", len(got), got)
	}
}

func TestSLP010_NoFindingWhenAddedLinesIncludeSetupAndAssertion(t *testing.T) {
	// Existing test with added lines that include both setup AND assertion.
	d := parseDiff(t, `diff --git a/a/foo_test.go b/a/foo_test.go
--- a/a/foo_test.go
+++ b/a/foo_test.go
@@ -1,4 +1,7 @@
 package a
 func TestFoo(t *testing.T) {
+	result := Foo()
+	if result != 42 {
+		t.Errorf("Foo() = %d, want 42", result)
+	}
 }
`)
	got := SLP010{}.Check(d)
	if len(got) != 0 {
		t.Fatalf("expected 0 findings (has assertion), got %d: %+v", len(got), got)
	}
}

func TestSLP010_FiresWithRequireLibrary(t *testing.T) {
	// No require.* or assert.* among added lines → finding.
	d := parseDiff(t, `diff --git a/a/foo_test.go b/a/foo_test.go
--- a/a/foo_test.go
+++ b/a/foo_test.go
@@ -1,4 +1,5 @@
 package a
 func TestFoo(t *testing.T) {
+	result := Foo()
 }
`)
	got := SLP010{}.Check(d)
	if len(got) != 1 {
		t.Fatalf("expected 1 finding, got %d: %+v", len(got), got)
	}
}

func TestSLP010_NoFindingWithRequireLib(t *testing.T) {
	// Added line has require.Equal → no finding.
	d := parseDiff(t, `diff --git a/a/foo_test.go b/a/foo_test.go
--- a/a/foo_test.go
+++ b/a/foo_test.go
@@ -1,4 +1,6 @@
 package a
 func TestFoo(t *testing.T) {
+	result := Foo()
+	require.Equal(t, 42, result)
 }
`)
	got := SLP010{}.Check(d)
	if len(got) != 0 {
		t.Fatalf("expected 0 findings (require.Equal), got %d: %+v", len(got), got)
	}
}

func TestSLP010_IgnoresNonTestFiles(t *testing.T) {
	// A function named Test* in a non-_test.go file is not a test.
	d := parseDiff(t, `diff --git a/a/foo.go b/a/foo.go
--- a/a/foo.go
+++ b/a/foo.go
@@ -1,3 +1,4 @@
 package a
 func TestHelper(x int) int {
+	return x * 2
 }
`)
	got := SLP010{}.Check(d)
	if len(got) != 0 {
		t.Fatalf("expected 0 findings in non-_test.go, got %d", len(got))
	}
}

func TestSLP010_IgnoresTopLevelSkip(t *testing.T) {
	// Added t.Skip at top level → intentional scaffold, no finding.
	d := parseDiff(t, `diff --git a/a/foo_test.go b/a/foo_test.go
--- a/a/foo_test.go
+++ b/a/foo_test.go
@@ -1,4 +1,5 @@
 package a
 func TestFoo(t *testing.T) {
+	t.Skip("not yet implemented")
 }
`)
	got := SLP010{}.Check(d)
	if len(got) != 0 {
		t.Fatalf("expected 0 findings (t.Skip scaffold), got %d: %+v", len(got), got)
	}
}

func TestSLP010_IgnoresSafetyTestName(t *testing.T) {
	// Test named with safety suffix (e.g. NoPanic) → legitimate no-assertion test.
	d := parseDiff(t, `diff --git a/a/foo_test.go b/a/foo_test.go
--- a/a/foo_test.go
+++ b/a/foo_test.go
@@ -1,4 +1,5 @@
 package a
 func TestFoo_NoPanic(t *testing.T) {
+	CallDangerousThing()
 }
`)
	got := SLP010{}.Check(d)
	if len(got) != 0 {
		t.Fatalf("expected 0 findings (safety test), got %d: %+v", len(got), got)
	}
}

func TestSLP010_MultipleAddedLinesNoAssertion(t *testing.T) {
	// Multiple added lines in existing test, none with assertion → single finding.
	d := parseDiff(t, `diff --git a/a/foo_test.go b/a/foo_test.go
--- a/a/foo_test.go
+++ b/a/foo_test.go
@@ -1,4 +1,7 @@
 package a
 func TestFoo(t *testing.T) {
+	setup := NewFixture()
+	result := Run(setup)
+	_ = result
 }
`)
	got := SLP010{}.Check(d)
	if len(got) != 1 {
		t.Fatalf("expected 1 finding, got %d: %+v", len(got), got)
	}
}

func TestSLP010_Description(t *testing.T) {
	r := SLP010{}
	if r.ID() != "SLP010" {
		t.Errorf("ID = %q", r.ID())
	}
	if r.Description() == "" {
		t.Errorf("Description is empty")
	}
	if r.DefaultSeverity() != SeverityInfo {
		t.Errorf("DefaultSeverity = %v, want SeverityInfo", r.DefaultSeverity())
	}
}

// containsStr is a small helper to avoid importing strings just for one check.
func containsStr(s, sub string) bool {
	return len(s) >= len(sub) && searchStr(s, sub)
}

func searchStr(s, sub string) bool {
	for i := 0; i <= len(s)-len(sub); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}
