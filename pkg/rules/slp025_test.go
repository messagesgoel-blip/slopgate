package rules

import (
	"strings"
	"testing"
)

func TestSLP025_FiresOnURLConcat(t *testing.T) {
	d := parseDiff(t, `diff --git a/routes/billing.js b/routes/billing.js
--- a/routes/billing.js
+++ b/routes/billing.js
@@ -10,3 +10,4 @@
+  const url = "`+"${APP_URL}${successPath}"+`";
`)
	got := SLP025{}.Check(d)
	if len(got) != 1 {
		t.Fatalf("expected 1 finding, got %d: %+v", len(got), got)
	}
}

func TestSLP025_FiresOnBaseUrlPath(t *testing.T) {
	d := parseDiff(t, `diff --git a/api.js b/api.js
--- a/api.js
+++ b/api.js
@@ -5,2 +5,3 @@
+  const full = "`+"${BASE_URL}${path}"+`";
`)
	got := SLP025{}.Check(d)
	if len(got) != 1 {
		t.Fatalf("expected 1 finding, got %d", len(got))
	}
}

func TestSLP025_IgnoresTestFiles(t *testing.T) {
	d := parseDiff(t, `diff --git a/routes.test.js b/routes.test.js
--- a/routes.test.js
+++ b/routes.test.js
@@ -1,2 +1,3 @@
+  const url = "`+"${APP_URL}${path}"+`";
`)
	got := SLP025{}.Check(d)
	if len(got) != 0 {
		t.Fatalf("expected 0 findings in test file, got %d", len(got))
	}
}

func TestSLP025_IgnoresNonJS(t *testing.T) {
	d := parseDiff(t, `diff --git a/config.py b/config.py
--- a/config.py
+++ b/config.py
@@ -1,2 +1,3 @@
+url = BASE_URL + path
`)
	got := SLP025{}.Check(d)
	if len(got) != 0 {
		t.Fatalf("expected 0 findings for Python, got %d", len(got))
	}
}

func TestSLP025_Description(t *testing.T) {
	r := SLP025{}
	if r.ID() != "SLP025" {
		t.Errorf("ID = %q", r.ID())
	}
	if !strings.Contains(strings.ToLower(r.Description()), "validation") {
		t.Errorf("description should mention validation: %q", r.Description())
	}
}
