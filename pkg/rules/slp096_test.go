package rules

import "testing"

func TestSLP096_FiresOnShellWithoutSetE(t *testing.T) {
	d := parseDiff(t, `diff --git a/deploy.sh b/deploy.sh
new file mode 100755
--- /dev/null
+++ b/deploy.sh
@@ -0,0 +1,5 @@
+#!/bin/bash
+echo "deploying..."
+./deploy
+echo "done"
`)
	got := SLP096{}.Check(d)
	if len(got) == 0 {
		t.Fatal("expected findings for missing set -e")
	}
}

func TestSLP096_NoFireOnShellWithSetE(t *testing.T) {
	d := parseDiff(t, `diff --git a/deploy.sh b/deploy.sh
new file mode 100755
--- /dev/null
+++ b/deploy.sh
@@ -0,0 +1,5 @@
+#!/bin/bash
+set -euo pipefail
+echo "deploying..."
+./deploy
`)
	got := SLP096{}.Check(d)
	if len(got) != 0 {
		t.Fatalf("expected 0 findings for script with set -e, got %d", len(got))
	}
}

func TestSLP096_IgnoresExistingFiles(t *testing.T) {
	d := parseDiff(t, `diff --git a/deploy.sh b/deploy.sh
--- a/deploy.sh
+++ b/deploy.sh
@@ -1,1 +1,3 @@
+echo "new line"
`)
	got := SLP096{}.Check(d)
	if len(got) != 0 {
		t.Fatalf("expected 0 findings for existing file, got %d", len(got))
	}
}

func TestSLP096_Description(t *testing.T) {
	r := SLP096{}
	if r.ID() != "SLP096" {
		t.Errorf("ID = %q", r.ID())
	}
	if r.Description() == "" {
		t.Errorf("Description is empty")
	}
	if r.DefaultSeverity() != SeverityWarn {
		t.Errorf("default severity should be warn")
	}
}
