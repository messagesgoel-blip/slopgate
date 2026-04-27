package rules

import (
	"strings"
	"testing"
)

func assertFindings(t *testing.T, got []Finding, wantCount int, wantRule string, wantSeverity Severity) {
	t.Helper()
	if len(got) != wantCount {
		t.Fatalf("expected %d finding(s), got %d: %v", wantCount, len(got), got)
	}
	if wantCount > 0 {
		if got[0].RuleID != wantRule {
			t.Errorf("RuleID = %q, want %q", got[0].RuleID, wantRule)
		}
		if got[0].Severity != wantSeverity {
			t.Errorf("Severity = %v, want %v", got[0].Severity, wantSeverity)
		}
	}
}

func TestSLP094_FiresOnOrTrue(t *testing.T) {
	d := parseDiff(t, `diff --git a/build.sh b/build.sh
--- a/build.sh
+++ b/build.sh
@@ -1,1 +1,3 @@
+  go build ./... || true
`)
	assertFindings(t, SLP094{}.Check(d), 1, "SLP094", SeverityBlock)
}

func TestSLP094_FiresOnOrColon(t *testing.T) {
	d := parseDiff(t, `diff --git a/ci.yml b/ci.yml
--- a/ci.yml
+++ b/ci.yml
@@ -1,1 +1,3 @@
+  run: npm test || :
`)
	assertFindings(t, SLP094{}.Check(d), 1, "SLP094", SeverityBlock)
}

func TestSLP094_IgnoresNonShell(t *testing.T) {
	d := parseDiff(t, `diff --git a/main.go b/main.go
--- a/main.go
+++ b/main.go
@@ -1,1 +1,3 @@
+  x := someFunc() || true
`)
	got := SLP094{}.Check(d)
	if len(got) != 0 {
		t.Fatalf("expected 0 findings for .go file, got %d", len(got))
	}
}

func TestSLP094_IgnoresCommentedSilentFail(t *testing.T) {
	d := parseDiff(t, `diff --git a/build.sh b/build.sh
--- a/build.sh
+++ b/build.sh
@@ -1,1 +1,3 @@
+  # go build ./... || true
`)
	got := SLP094{}.Check(d)
	if len(got) != 0 {
		t.Fatalf("expected 0 findings for commented shell line, got %d", len(got))
	}
}

func TestSLP094_IgnoresWorkflowLikeYamlOutsideCILocations(t *testing.T) {
	d := parseDiff(t, `diff --git a/docs/build-workflow.yaml b/docs/build-workflow.yaml
--- a/docs/build-workflow.yaml
+++ b/docs/build-workflow.yaml
@@ -1,1 +1,3 @@
+  run: npm test || true
`)
	got := SLP094{}.Check(d)
	if len(got) != 0 {
		t.Fatalf("expected 0 findings for unrelated workflow yaml, got %d", len(got))
	}
}

func TestSLP094_IgnoresNonRunYAMLMetadata(t *testing.T) {
	d := parseDiff(t, `diff --git a/.github/workflows/ci.yml b/.github/workflows/ci.yml
--- a/.github/workflows/ci.yml
+++ b/.github/workflows/ci.yml
@@ -1,1 +1,4 @@
+  env:
+    BUILD_NOTE: "npm test || true"
`)
	got := SLP094{}.Check(d)
	if len(got) != 0 {
		t.Fatalf("expected 0 findings for non-run yaml metadata, got %d", len(got))
	}
}

func TestSLP094_FiresOnRunBlockScalarCommand(t *testing.T) {
	d := parseDiff(t, `diff --git a/.github/workflows/ci.yml b/.github/workflows/ci.yml
--- a/.github/workflows/ci.yml
+++ b/.github/workflows/ci.yml
@@ -1,3 +1,5 @@
   run: |
+    npm test || true
`)
	assertFindings(t, SLP094{}.Check(d), 1, "SLP094", SeverityBlock)
}

func TestSLP094_FiresOnRunListEntry(t *testing.T) {
	d := parseDiff(t, `diff --git a/.github/workflows/ci.yml b/.github/workflows/ci.yml
--- a/.github/workflows/ci.yml
+++ b/.github/workflows/ci.yml
@@ -1,1 +1,3 @@
+  - run: npm test || true
`)
	assertFindings(t, SLP094{}.Check(d), 1, "SLP094", SeverityBlock)
}

func TestSLP094_FiresOnChainedSilentFail(t *testing.T) {
	d := parseDiff(t, `diff --git a/build.sh b/build.sh
--- a/build.sh
+++ b/build.sh
@@ -1,1 +1,3 @@
+  go build || true&&echo done
`)
	assertFindings(t, SLP094{}.Check(d), 1, "SLP094", SeverityBlock)
}

func TestSLP094_BlockScalarSiblingKeyNotFlagged(t *testing.T) {
	d := parseDiff(t, `diff --git a/.github/workflows/ci.yml b/.github/workflows/ci.yml
--- a/.github/workflows/ci.yml
+++ b/.github/workflows/ci.yml
@@ -1,3 +1,5 @@
+  - run: |
+      npm test || true
+    if: always() || true
`)
	got := SLP094{}.Check(d)
	if len(got) != 1 {
		t.Fatalf("expected exactly 1 finding for run block command, got %d: %v", len(got), got)
	}
	if !strings.Contains(got[0].Snippet, "npm test || true") {
		t.Errorf("expected finding to target the run block command, got snippet %q", got[0].Snippet)
	}
}

func TestSLP094_IgnoresMakefileMetadata(t *testing.T) {
	d := parseDiff(t, `diff --git a/Makefile b/Makefile
--- a/Makefile
+++ b/Makefile
@@ -1,1 +1,3 @@
+VERSION := 1 || true
`)
	got := SLP094{}.Check(d)
	if len(got) != 0 {
		t.Fatalf("expected 0 findings for Makefile metadata line, got %d: %v", len(got), got)
	}
}

func TestSLP094_IgnoresInlineCommentedSilentFail(t *testing.T) {
	d := parseDiff(t, `diff --git a/build.sh b/build.sh
--- a/build.sh
+++ b/build.sh
@@ -1,1 +1,3 @@
+go build ./... # || true
`)
	got := SLP094{}.Check(d)
	if len(got) != 0 {
		t.Fatalf("expected 0 findings for inline comment containing || true, got %d: %v", len(got), got)
	}
}

func TestSLP094_Description(t *testing.T) {
	r := SLP094{}
	if r.ID() != "SLP094" {
		t.Errorf("ID = %q", r.ID())
	}
	if r.Description() == "" {
		t.Errorf("Description is empty")
	}
	if r.DefaultSeverity() != SeverityBlock {
		t.Errorf("default severity should be block")
	}
}
