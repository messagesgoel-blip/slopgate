package rules

import "testing"

// checkSLP097 runs the diff through SLP097{}.Check and asserts the expected finding count.
// For positive cases (wantCount > 0) it also validates RuleID and Severity on each finding.
func checkSLP097(t *testing.T, diffStr string, wantCount int) []Finding {
	t.Helper()
	d := parseDiff(t, diffStr)
	got := SLP097{}.Check(d)
	if len(got) != wantCount {
		t.Fatalf("expected %d findings, got %d: %v", wantCount, len(got), got)
	}
	for _, f := range got {
		if f.RuleID != "SLP097" || f.Severity != SeverityWarn {
			t.Errorf("unexpected finding metadata: RuleID=%q Severity=%v", f.RuleID, f.Severity)
		}
	}
	return got
}

func TestSLP097_FiresOnDataDestructureWithoutOkCheck(t *testing.T) {
	checkSLP097(t, `diff --git a/api.ts b/api.ts
--- a/api.ts
+++ b/api.ts
@@ -1,1 +1,3 @@
+  const { data } = await api.get();
`, 1)
}

func TestSLP097_FiresOnFetchWithoutOkCheck(t *testing.T) {
	checkSLP097(t, `diff --git a/api.ts b/api.ts
--- a/api.ts
+++ b/api.ts
@@ -1,1 +1,3 @@
+  fetch("/api/items").then(res => res.json())
`, 1)
}

func TestSLP097_IgnoresMismatchedJsonReceiver(t *testing.T) {
	checkSLP097(t, `diff --git a/api.ts b/api.ts
--- a/api.ts
+++ b/api.ts
@@ -1,1 +1,3 @@
+  fetch("/api/items").then(res => response.json())
`, 0)
}

func TestSLP097_IgnoresPrefetchHelper(t *testing.T) {
	checkSLP097(t, `diff --git a/api.ts b/api.ts
--- a/api.ts
+++ b/api.ts
@@ -1,1 +1,3 @@
+  prefetch("/api/items").then(res => res.json())
`, 0)
}

func TestSLP097_IgnoresTestFiles(t *testing.T) {
	checkSLP097(t, `diff --git a/api.test.ts b/api.test.ts
--- a/api.test.ts
+++ b/api.test.ts
@@ -1,1 +1,3 @@
+  const { data } = await api.get();
`, 0)
}

func TestSLP097_IgnoresNonJSTS(t *testing.T) {
	checkSLP097(t, `diff --git a/api.go b/api.go
--- a/api.go
+++ b/api.go
@@ -1,1 +1,3 @@
+  const { data } = await api.get();  // nonsense in Go
`, 0)
}

func TestSLP097_FiresOnFetchWithTypedParam(t *testing.T) {
	checkSLP097(t, `diff --git a/api.ts b/api.ts
--- a/api.ts
+++ b/api.ts
@@ -1,1 +1,3 @@
+  fetch("/api/items").then((res: Response) => res.json())
`, 1)
}

func TestSLP097_FiresOnNestedFetch(t *testing.T) {
	checkSLP097(t, `diff --git a/api.ts b/api.ts
--- a/api.ts
+++ b/api.ts
@@ -1,1 +1,3 @@
+  fetch("/a").then(res => fetch("/b").then(r => r.json()))
`, 1)
}

func TestSLP097_Description(t *testing.T) {
	r := SLP097{}
	if r.ID() != "SLP097" {
		t.Errorf("ID = %q", r.ID())
	}
	if r.Description() == "" {
		t.Errorf("Description is empty")
	}
	if r.DefaultSeverity() != SeverityWarn {
		t.Errorf("default severity should be warn")
	}
}
