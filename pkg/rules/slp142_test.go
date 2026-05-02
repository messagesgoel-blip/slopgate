package rules

import (
	"testing"
)

func TestSLP142_FiresOnUnsafePathJoin(t *testing.T) {
	d := parseDiff(t, `
diff --git a/pkg/rules/slp007.go b/pkg/rules/slp007.go
--- a/pkg/rules/slp007.go
+++ b/pkg/rules/slp007.go
@@ -1,5 +1,10 @@
+func slp007FileLines(d *diff.Diff, relPath string) ([]string, bool) {
+	resolved := filepath.Join(d.RepoRoot, relPath)
+	content, err := os.ReadFile(resolved)
+	if err != nil {
+		return nil, false
+	}
+	return strings.Split(string(content), "\n"), true
+}
`)
	got := SLP142{}.Check(d)
	if len(got) != 1 {
		t.Fatalf("expected 1 finding, got %d", len(got))
	}
}

func TestSLP142_IgnoresSafePathJoin(t *testing.T) {
	d := parseDiff(t, `
diff --git a/pkg/rules/slp007.go b/pkg/rules/slp007.go
--- a/pkg/rules/slp007.go
+++ b/pkg/rules/slp007.go
@@ -1,5 +1,11 @@
+func slp007FileLines(d *diff.Diff, relPath string) ([]string, bool) {
+	resolved := filepath.Join(d.RepoRoot, relPath)
+	eval, err := filepath.EvalSymlinks(resolved)
+	if err != nil {
+		return nil, false
+	}
+	content, err := os.ReadFile(eval)
+	return strings.Split(string(content), "\n"), true
+}
`)
	got := SLP142{}.Check(d)
	if len(got) != 0 {
		t.Fatalf("expected 0 findings, got %d: %+v", len(got), got)
	}
}

func TestSLP142_FiresOnRelWithoutEval(t *testing.T) {
	d := parseDiff(t, `
diff --git a/pkg/rules/slp007.go b/pkg/rules/slp007.go
--- a/pkg/rules/slp007.go
+++ b/pkg/rules/slp007.go
@@ -1,5 +1,11 @@
+func slp007FileLines(d *diff.Diff, relPath string) ([]string, bool) {
+	resolved := filepath.Join(d.RepoRoot, relPath)
+	rel, err := filepath.Rel(d.RepoRoot, resolved)
+	if err != nil || strings.HasPrefix(rel, "..") {
+		return nil, false
+	}
+	content, err := os.ReadFile(resolved)
+	return strings.Split(string(content), "\n"), true
+}
`)
	got := SLP142{}.Check(d)
	if len(got) != 1 {
		t.Fatalf("expected 1 finding (Rel without Eval), got %d: %+v", len(got), got)
	}
}
