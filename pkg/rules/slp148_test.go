package rules

import "testing"

func TestSLP148_FiresOnInconsistentNaming(t *testing.T) {
	d := parseDiff(t, `diff --git a/service1.js b/service1.js
--- a/service1.js
+++ b/service1.js
@@ -1,2 +1,3 @@
 module.exports = {};
+export const userId = user.id;
diff --git a/service2.js b/service2.js
--- a/service2.js
+++ b/service2.js
@@ -1,2 +1,3 @@
 module.exports = {};
+export const userID = formatUser(req.user);
 `)
	got := SLP148{}.Check(d)
	if len(got) != 1 {
		t.Fatalf("expected 1 finding (userId vs userID), got %d: %+v", len(got), got)
	}
}

func TestSLP148_AllowsConsistentNaming(t *testing.T) {
	d := parseDiff(t, `diff --git a/service1.js b/service1.js
--- a/service1.js
+++ b/service1.js
@@ -1,2 +1,3 @@
 module.exports = {};
+export const userId = user.id;
diff --git a/service2.js b/service2.js
--- a/service2.js
+++ b/service2.js
@@ -1,2 +1,3 @@
 module.exports = {};
+export const userId = formatUser(req.user);
 `)
	got := SLP148{}.Check(d)
	if len(got) != 0 {
		t.Fatalf("expected 0 findings (consistent userId), got %d", len(got))
	}
}
