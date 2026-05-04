package rules

import "testing"

func TestSLP145_FiresOnExtremeTimeoutWithoutComment(t *testing.T) {
	d := parseDiff(t, `diff --git a/api.js b/api.js
--- a/api.js
+++ b/api.js
@@ -1,2 +1,3 @@
 import express from 'express';
+const timeout = 500;
 const app = express();
 `)
	got := SLP145{}.Check(d)
	if len(got) != 1 {
		t.Fatalf("expected 1 finding (short timeout no comment), got %d: %+v", len(got), got)
	}
}

func TestSLP145_AllowsExtremeTimeoutWithComment(t *testing.T) {
	d := parseDiff(t, `diff --git a/api.js b/api.js
--- a/api.js
+++ b/api.js
@@ -20,3 +20,4 @@
-const timeout = 500;
+// 500ms: quick health check, failures handled by retry
+const timeout = 500;
 `)
	got := SLP145{}.Check(d)
	if len(got) != 0 {
		t.Fatalf("expected 0 findings (has comment), got %d", len(got))
	}
}

func TestSLP145_AllowsModerateTimeout(t *testing.T) {
	d := parseDiff(t, `diff --git a/api.js b/api.js
--- a/api.js
+++ b/api.js
@@ -20,3 +20,4 @@
 const app = express();
+const timeout = 10000;
 const server = app.listen(3000);
 `)
	got := SLP145{}.Check(d)
	if len(got) != 0 {
		t.Fatalf("expected 0 findings (10s is moderate), got %d", len(got))
	}
}
