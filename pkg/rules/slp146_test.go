package rules

import "testing"

func TestSLP146_FiresOnAsyncMapWithoutAwait(t *testing.T) {
	d := parseDiff(t, `diff --git a/processor.js b/processor.js
--- a/processor.js
+++ b/processor.js
@@ -10,2 +10,3 @@
 processItems();
+items.map(async item => process(item));
 `)
	got := SLP146{}.Check(d)
	if len(got) != 1 {
		t.Fatalf("expected 1 finding (async map), got %d: %+v", len(got), got)
	}
}

func TestSLP146_AllowsPromiseAll(t *testing.T) {
	d := parseDiff(t, `diff --git a/processor.js b/processor.js
--- a/processor.js
+++ b/processor.js
@@ -10,2 +10,4 @@
 processItems();
-await Promise.all(items.map(item => process(item)));
+await Promise.all(items.map(async item => process(item)));
 `)
	got := SLP146{}.Check(d)
	if len(got) != 0 {
		t.Fatalf("expected 0 findings (Promise.all wrapper), got %d", len(got))
	}
}
