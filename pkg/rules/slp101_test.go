package rules

import "testing"

func TestSLP101_FiresOnFeatureFlagCheck(t *testing.T) {
	d := parseDiff(t, `diff --git a/feature.ts b/feature.ts
--- a/feature.ts
+++ b/feature.ts
@@ -1,1 +1,3 @@
+  if (featureFlag('new-ui')) {
`)
	got := SLP101{}.Check(d)
	if len(got) == 0 {
		t.Fatal("expected findings for feature flag check")
	}
}

func TestSLP101_FiresOnEmptyElseBranch(t *testing.T) {
	d := parseDiff(t, `diff --git a/feature.ts b/feature.ts
--- a/feature.ts
+++ b/feature.ts
@@ -1,1 +1,3 @@
+  } else {}
`)
	got := SLP101{}.Check(d)
	if len(got) == 0 {
		t.Fatal("expected findings for empty else")
	}
}

func TestSLP101_IgnoresTestFiles(t *testing.T) {
	d := parseDiff(t, `diff --git a/feature.test.ts b/feature.test.ts
--- a/feature.test.ts
+++ b/feature.test.ts
@@ -1,1 +1,3 @@
+  if (featureFlag('new-ui')) {
`)
	got := SLP101{}.Check(d)
	if len(got) != 0 {
		t.Fatalf("expected 0 findings for test file, got %d", len(got))
	}
}

func TestSLP101_IgnoresCommentedOutFlagCheck(t *testing.T) {
	d := parseDiff(t, `diff --git a/feature.ts b/feature.ts
--- a/feature.ts
+++ b/feature.ts
@@ -1,1 +1,3 @@
+  // if featureFlag('new-ui') {
`)
	got := SLP101{}.Check(d)
	if len(got) != 0 {
		t.Fatalf("expected 0 findings for commented-out flag check, got %d", len(got))
	}
}

func TestSLP101_IgnoresBlockCommentedOutFlagCheck(t *testing.T) {
	d := parseDiff(t, `diff --git a/feature.ts b/feature.ts
--- a/feature.ts
+++ b/feature.ts
@@ -1,1 +1,5 @@
+  /*
+   * if featureFlag('new-ui') {
+   */
`)
	got := SLP101{}.Check(d)
	if len(got) != 0 {
		t.Fatalf("expected 0 findings for block-commented flag check, got %d", len(got))
	}
}

func TestSLP101_Description(t *testing.T) {
	r := SLP101{}
	if r.ID() != "SLP101" {
		t.Errorf("ID = %q", r.ID())
	}
	if r.Description() == "" {
		t.Errorf("Description is empty")
	}
	if r.DefaultSeverity() != SeverityWarn {
		t.Errorf("default severity should be warn")
	}
}
