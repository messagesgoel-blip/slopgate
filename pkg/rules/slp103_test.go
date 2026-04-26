package rules

import "testing"

func TestSLP103_FiresOnGoDuration(t *testing.T) {
	d := parseDiff(t, `diff --git a/server.go b/server.go
--- a/server.go
+++ b/server.go
@@ -1,1 +1,3 @@
+  ctx, cancel := context.WithTimeout(ctx, time.Second * 30)
`)
	got := SLP103{}.Check(d)
	if len(got) == 0 {
		t.Fatal("expected findings for hardcoded duration")
	}
}

func TestSLP103_FiresOnJSTimeout(t *testing.T) {
	d := parseDiff(t, `diff --git a/api.ts b/api.ts
--- a/api.ts
+++ b/api.ts
@@ -1,1 +1,3 @@
+  setTimeout(() => fetchData(), 5000);
`)
	got := SLP103{}.Check(d)
	if len(got) == 0 {
		t.Fatal("expected findings for setTimeout")
	}
}

func TestSLP103_IgnoresTestFiles(t *testing.T) {
	d := parseDiff(t, `diff --git a/server.test.go b/server.test.go
--- a/server.test.go
+++ b/server.test.go
@@ -1,1 +1,3 @@
+  ctx, cancel := context.WithTimeout(ctx, time.Second * 30)
`)
	got := SLP103{}.Check(d)
	if len(got) != 0 {
		t.Fatalf("expected 0 findings for test file, got %d", len(got))
	}
}

func TestSLP103_Description(t *testing.T) {
	r := SLP103{}
	if r.ID() != "SLP103" {
		t.Errorf("ID = %q", r.ID())
	}
	if r.Description() == "" {
		t.Errorf("Description is empty")
	}
	if r.DefaultSeverity() != SeverityInfo {
		t.Errorf("default severity should be info")
	}
}
