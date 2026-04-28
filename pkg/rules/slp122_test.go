package rules

import "testing"

func TestSLP122_FiresOnPollingWithoutGuard(t *testing.T) {
	d := parseDiff(t, `diff --git a/app/src/pages/OrganizationPage.tsx b/app/src/pages/OrganizationPage.tsx
--- a/app/src/pages/OrganizationPage.tsx
+++ b/app/src/pages/OrganizationPage.tsx
@@ -1,1 +1,4 @@
+const poll = async () => { await api.runTagScan(payload); setTimeout(poll, 4000) }
`)
	got := SLP122{}.Check(d)
	if len(got) == 0 {
		t.Fatal("expected finding for polling loop without cancellation/in-flight guard")
	}
}

func TestSLP122_NoFireWithAbortAndTimeoutCleanup(t *testing.T) {
	d := parseDiff(t, `diff --git a/app/src/pages/OrganizationPage.tsx b/app/src/pages/OrganizationPage.tsx
--- a/app/src/pages/OrganizationPage.tsx
+++ b/app/src/pages/OrganizationPage.tsx
@@ -1,1 +1,7 @@
+const controller = new AbortController()
+const tick = async () => { await fetch(url, { signal: controller.signal }) }
+const id = setTimeout(tick, 4000)
+return () => { clearTimeout(id); controller.abort() }
`)
	got := SLP122{}.Check(d)
	if len(got) != 0 {
		t.Fatalf("expected 0 findings when cancellation guards exist, got %d", len(got))
	}
}

func TestSLP122_Description(t *testing.T) {
	r := SLP122{}
	if r.ID() != "SLP122" {
		t.Errorf("ID = %q", r.ID())
	}
	if r.Description() == "" {
		t.Errorf("Description is empty")
	}
	if r.DefaultSeverity() != SeverityWarn {
		t.Errorf("default severity should be warn")
	}
}
