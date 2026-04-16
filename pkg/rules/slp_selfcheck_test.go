package rules

import "testing"

// TestSLP005_IgnoresGoTestFilesWithJSFixtureStrings covers the self-
// reference case: slopgate's own SLP005 tests contain Go raw string
// literals holding JS-shaped test fixtures (`it.only(...)` etc.). A
// naive implementation treats those string contents as real `.only`
// usage; the correct behavior is to skip Go files entirely since Go's
// testing package has no `.only`.
func TestSLP005_IgnoresGoTestFilesWithJSFixtureStrings(t *testing.T) {
	d := parseDiff(t, `diff --git a/pkg/rules/slp005_test.go b/pkg/rules/slp005_test.go
--- a/pkg/rules/slp005_test.go
+++ b/pkg/rules/slp005_test.go
@@ -1,2 +1,6 @@
 package rules
+
+func TestFixture(t *testing.T) {
+	d := parseDiff(t, `+"`"+`+  it.only("runs just this", () => expect(1).toBe(1));`+"`"+`)
+}
`)
	got := SLP005{}.Check(d)
	if len(got) != 0 {
		t.Fatalf("expected 0 findings on Go test file with JS fixture string, got %d: %+v", len(got), got)
	}
}

// TestSLP012_IgnoresProseThatMentionsTODO covers the self-reference
// case: slopgate's own slp012.go has doc comments describing the rule
// that contain the word "TODO" in prose. A real TODO comment puts
// the marker first; prose mentions it mid-sentence.
func TestSLP012_IgnoresProseThatMentionsTODO(t *testing.T) {
	d := parseDiff(t, `diff --git a/a.go b/a.go
--- a/a.go
+++ b/a.go
@@ -1,1 +1,5 @@
 package a
+// This function detects TODO / FIXME / HACK / XXX comments and flags
+// them as slop. TODO markers in backlog docs are legitimate; TODO
+// markers in freshly generated code are a tell that an AI agent
+// stopped before finishing the job.
`)
	got := SLP012{}.Check(d)
	if len(got) != 0 {
		t.Fatalf("expected 0 findings on prose that mentions TODO, got %d: %+v", len(got), got)
	}
}

// TestSLP012_StillFiresOnRealMarkerAtCommentStart keeps the happy path
// covered after the tightening in slp012Pattern.
func TestSLP012_StillFiresOnRealMarkerAtCommentStart(t *testing.T) {
	d := parseDiff(t, `diff --git a/a.go b/a.go
--- a/a.go
+++ b/a.go
@@ -1,1 +1,3 @@
 package a
+// TODO: implement this
+// FIXME: broken in edge case
`)
	got := SLP012{}.Check(d)
	if len(got) != 2 {
		t.Fatalf("expected 2 findings, got %d", len(got))
	}
}
