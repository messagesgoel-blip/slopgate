package rules

import "testing"

func TestSLP147_FiresOnUnsafeDestructuring(t *testing.T) {
	d := parseDiff(t, `diff --git a/api.js b/api.js
--- a/api.js
+++ b/api.js
@@ -15,3 +15,3 @@
 function handler(req) {
+	const {userId} = req.user;
 	process(userId);
 }
 `)
	got := SLP147{}.Check(d)
	if len(got) != 1 {
		t.Fatalf("expected 1 finding (unsafe destructuring), got %d: %+v", len(got), got)
	}
}

func TestSLP147_AllowsSafeDefault(t *testing.T) {
	d := parseDiff(t, `diff --git a/api.js b/api.js
--- a/api.js
+++ b/api.js
@@ -15,3 +15,4 @@
 function handler(req) {
+	const {userId = 'guest'} = req.user || {};
 	process(userId);
 }
 `)
	got := SLP147{}.Check(d)
	if len(got) != 0 {
		t.Fatalf("expected 0 findings (has defaults), got %d", len(got))
	}
}
