package rules

import (
	"strings"
	"testing"
)

func TestSLP002_GoTautologicalEqual(t *testing.T) {
	// assert.Equal(t, got, got) — same identifier → finding.
	d := parseDiff(t, `diff --git a/foo_test.go b/foo_test.go
--- a/foo_test.go
+++ b/foo_test.go
@@ -1,1 +1,3 @@
 package foo
+func TestFoo(t *testing.T) {
+	assert.Equal(t, got, got)
+}
`)
	got := SLP002{}.Check(d)
	if len(got) != 1 {
		t.Fatalf("expected 1 finding, got %d: %+v", len(got), got)
	}
	if !strings.Contains(got[0].Message, "got") {
		t.Errorf("message should mention identifier: %q", got[0].Message)
	}
	if got[0].File != "foo_test.go" {
		t.Errorf("file: %q", got[0].File)
	}
}

func TestSLP002_JSTautologicalToBe(t *testing.T) {
	// expect(val).toBe(val) — same identifier → finding.
	d := parseDiff(t, `diff --git a/foo.test.ts b/foo.test.ts
--- a/foo.test.ts
+++ b/foo.test.ts
@@ -1,1 +1,2 @@
 // test
+expect(val).toBe(val)
`)
	got := SLP002{}.Check(d)
	if len(got) != 1 {
		t.Fatalf("expected 1 finding, got %d: %+v", len(got), got)
	}
	if !strings.Contains(got[0].Message, "val") {
		t.Errorf("message should mention identifier: %q", got[0].Message)
	}
}

func TestSLP002_PythonTautologicalAssertEqual(t *testing.T) {
	// self.assertEqual(x, x) — same identifier → finding.
	d := parseDiff(t, `diff --git a/test_foo.py b/test_foo.py
--- a/test_foo.py
+++ b/test_foo.py
@@ -1,1 +1,2 @@
 # test
+    self.assertEqual(x, x)
`)
	got := SLP002{}.Check(d)
	if len(got) != 1 {
		t.Fatalf("expected 1 finding, got %d: %+v", len(got), got)
	}
	if !strings.Contains(got[0].Message, "x") {
		t.Errorf("message should mention identifier: %q", got[0].Message)
	}
}

func TestSLP002_GoDifferentIdentifiers_NoFinding(t *testing.T) {
	// assert.Equal(t, got, want) — different identifiers → no finding.
	d := parseDiff(t, `diff --git a/foo_test.go b/foo_test.go
--- a/foo_test.go
+++ b/foo_test.go
@@ -1,1 +1,3 @@
 package foo
+func TestFoo(t *testing.T) {
+	assert.Equal(t, got, want)
+}
`)
	got := SLP002{}.Check(d)
	if len(got) != 0 {
		t.Fatalf("expected 0 findings, got %d: %+v", len(got), got)
	}
}

func TestSLP002_GoTrueWithTrueLiteral(t *testing.T) {
	// assert.True(t, true) — tautological → finding.
	d := parseDiff(t, `diff --git a/foo_test.go b/foo_test.go
--- a/foo_test.go
+++ b/foo_test.go
@@ -1,1 +1,2 @@
 package foo
+	assert.True(t, true)
`)
	got := SLP002{}.Check(d)
	if len(got) != 1 {
		t.Fatalf("expected 1 finding, got %d: %+v", len(got), got)
	}
	if !strings.Contains(got[0].Message, "tautological") {
		t.Errorf("message should mention tautological: %q", got[0].Message)
	}
}

func TestSLP002_NonTestFile_NoFinding(t *testing.T) {
	// A tautological assertion in a non-test file should not fire.
	d := parseDiff(t, `diff --git a/foo.go b/foo.go
--- a/foo.go
+++ b/foo.go
@@ -1,1 +1,2 @@
 package foo
+	assert.Equal(t, got, got)
`)
	got := SLP002{}.Check(d)
	if len(got) != 0 {
		t.Fatalf("expected 0 findings in non-test file, got %d", len(got))
	}
}

func TestSLP002_RequireTautologicalEqual(t *testing.T) {
	// require.Equal(t, x, x) — same identifier → finding.
	d := parseDiff(t, `diff --git a/bar_test.go b/bar_test.go
--- a/bar_test.go
+++ b/bar_test.go
@@ -1,1 +1,2 @@
 package bar
+	require.Equal(t, x, x)
`)
	got := SLP002{}.Check(d)
	if len(got) != 1 {
		t.Fatalf("expected 1 finding, got %d: %+v", len(got), got)
	}
}

func TestSLP002_JSAssertStrictEqual(t *testing.T) {
	// assert.strictEqual(x, x) — same identifier → finding.
	d := parseDiff(t, `diff --git a/app.test.js b/app.test.js
--- a/app.test.js
+++ b/app.test.js
@@ -1,1 +1,2 @@
 // test
+assert.strictEqual(x, x)
`)
	got := SLP002{}.Check(d)
	if len(got) != 1 {
		t.Fatalf("expected 1 finding, got %d: %+v", len(got), got)
	}
}

func TestSLP002_JSDifferentIdentifiers_NoFinding(t *testing.T) {
	// expect(result).toBe(expected) — different identifiers → no finding.
	d := parseDiff(t, `diff --git a/foo.test.ts b/foo.test.ts
--- a/foo.test.ts
+++ b/foo.test.ts
@@ -1,1 +1,2 @@
 // test
+expect(result).toBe(expected)
`)
	got := SLP002{}.Check(d)
	if len(got) != 0 {
		t.Fatalf("expected 0 findings, got %d: %+v", len(got), got)
	}
}

func TestSLP002_PythonDifferentIdentifiers_NoFinding(t *testing.T) {
	// self.assertEqual(actual, expected) — different identifiers → no finding.
	d := parseDiff(t, `diff --git a/test_foo.py b/test_foo.py
--- a/test_foo.py
+++ b/test_foo.py
@@ -1,1 +1,2 @@
 # test
+    self.assertEqual(actual, expected)
`)
	got := SLP002{}.Check(d)
	if len(got) != 0 {
		t.Fatalf("expected 0 findings, got %d: %+v", len(got), got)
	}
}

func TestSLP002_GoFalseWithFalseLiteral(t *testing.T) {
	// assert.False(t, false) — tautological → finding.
	d := parseDiff(t, `diff --git a/foo_test.go b/foo_test.go
--- a/foo_test.go
+++ b/foo_test.go
@@ -1,1 +1,2 @@
 package foo
+	assert.False(t, false)
`)
	got := SLP002{}.Check(d)
	if len(got) != 1 {
		t.Fatalf("expected 1 finding, got %d: %+v", len(got), got)
	}
}

func TestSLP002_GoTrueWithVariable_NoFinding(t *testing.T) {
	// assert.True(t, ok) — variable, not literal true → no finding.
	d := parseDiff(t, `diff --git a/foo_test.go b/foo_test.go
--- a/foo_test.go
+++ b/foo_test.go
@@ -1,1 +1,2 @@
 package foo
+	assert.True(t, ok)
`)
	got := SLP002{}.Check(d)
	if len(got) != 0 {
		t.Fatalf("expected 0 findings, got %d: %+v", len(got), got)
	}
}

func TestSLP002_ContextLine_NoFinding(t *testing.T) {
	// A tautological assertion on a context (unchanged) line should not fire.
	d := parseDiff(t, `diff --git a/foo_test.go b/foo_test.go
--- a/foo_test.go
+++ b/foo_test.go
@@ -1,2 +1,2 @@
 package foo
-// old
+// new
 	assert.Equal(t, got, got)
`)
	got := SLP002{}.Check(d)
	if len(got) != 0 {
		t.Fatalf("expected 0 findings for context line, got %d", len(got))
	}
}

func TestSLP002_Description(t *testing.T) {
	r := SLP002{}
	if r.ID() != "SLP002" {
		t.Errorf("ID = %q", r.ID())
	}
	if r.Description() == "" {
		t.Errorf("Description is empty")
	}
	if r.DefaultSeverity() != SeverityBlock {
		t.Errorf("default severity should be block")
	}
}
