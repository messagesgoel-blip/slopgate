package rules

import (
	"testing"
)

func TestSLP141_FiresOnAsyncUseEffectWithoutGuard(t *testing.T) {
	d := parseDiff(t, `
diff --git a/app.js b/app.js
--- a/app.js
+++ b/app.js
@@ -1,5 +1,9 @@
+useEffect(() => {
+  const loadData = async () => {
+    const res = await fetch('/api/data');
+    setData(res.json());
+  };
+  loadData();
+}, []);
`)
	got := SLP141{}.Check(d)
	if len(got) != 1 {
		t.Fatalf("expected 1 finding, got %d", len(got))
	}
	if got[0].RuleID != "SLP141" {
		t.Errorf("expected SLP141, got %s", got[0].RuleID)
	}
}

func TestSLP141_IgnoresUseEffectWithLoadingGuard(t *testing.T) {
	d := parseDiff(t, `
diff --git a/app.js b/app.js
--- a/app.js
+++ b/app.js
@@ -1,5 +1,9 @@
+useEffect(() => {
+  if (isLoading) return;
+  fetch('/api/data');
+}, [isLoading]);
`)
	got := SLP141{}.Check(d)
	if len(got) != 0 {
		t.Fatalf("expected 0 findings, got %d: %+v", len(got), got)
	}
}

func TestSLP141_IgnoresUseEffectWithAbortController(t *testing.T) {
	d := parseDiff(t, `
diff --git a/app.js b/app.js
--- a/app.js
+++ b/app.js
@@ -1,5 +1,11 @@
+useEffect(() => {
+  const controller = new AbortController();
+  fetch('/api/data', { signal: controller.signal });
+  return () => controller.abort();
+}, []);
`)
	got := SLP141{}.Check(d)
	if len(got) != 0 {
		t.Fatalf("expected 0 findings, got %d: %+v", len(got), got)
	}
}
