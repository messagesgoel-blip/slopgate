package rules

import (
	"strings"
	"testing"
)

func TestSLP024_FiresOnCatchWith200(t *testing.T) {
	// Single-line catch with console.error and res.status(200).
	d := parseDiff(t, `diff --git a/api/routes/billing.js b/api/routes/billing.js
--- a/api/routes/billing.js
+++ b/api/routes/billing.js
@@ -10,3 +10,4 @@
+ catch (err) { console.error(err); return res.status(200).json({ received: true }); }
`)
	got := SLP024{}.Check(d)
	if len(got) != 1 {
		t.Fatalf("expected 1 finding, got %d: %+v", len(got), got)
	}
}

func TestSLP024_FiresOnCatchWithJsonSuccess(t *testing.T) {
	d := parseDiff(t, `diff --git a/handler.js b/handler.js
--- a/handler.js
+++ b/handler.js
@@ -5,2 +5,3 @@
+ catch (e) { console.error('failed', e); res.json({ success: true }); }
`)
	got := SLP024{}.Check(d)
	if len(got) != 1 {
		t.Fatalf("expected 1 finding, got %d: %+v", len(got), got)
	}
}

func TestSLP024_FiresOnCatchWithReturnSuccess(t *testing.T) {
	d := parseDiff(t, `diff --git a/webhook.js b/webhook.js
--- a/webhook.js
+++ b/webhook.js
@@ -8,2 +8,3 @@
+ catch (err) { console.error(err); return { received: true }; }
`)
	got := SLP024{}.Check(d)
	if len(got) != 1 {
		t.Fatalf("expected 1 finding, got %d", len(got))
	}
}

func TestSLP024_IgnoresCorrect500Return(t *testing.T) {
	// Correct: returns 500, not 200.
	d := parseDiff(t, `diff --git a/api/routes/billing.js b/api/routes/billing.js
--- a/api/routes/billing.js
+++ b/api/routes/billing.js
@@ -10,3 +10,4 @@
+ catch (err) { console.error(err); return res.status(500).json({ error: 'failed' }); }
`)
	got := SLP024{}.Check(d)
	if len(got) != 0 {
		t.Fatalf("expected 0 findings for 500 return, got %d", len(got))
	}
}

func TestSLP024_IgnoresCatchWithoutErrorLog(t *testing.T) {
	d := parseDiff(t, `diff --git a/handler.js b/handler.js
--- a/handler.js
+++ b/handler.js
@@ -3,2 +3,3 @@
+ catch (e) { return res.status(200).json({ ok: true }); }
`)
	got := SLP024{}.Check(d)
	if len(got) != 0 {
		t.Fatalf("expected 0 findings without error log, got %d", len(got))
	}
}

func TestSLP024_IgnoresTestFiles(t *testing.T) {
	d := parseDiff(t, `diff --git a/handler.test.js b/handler.test.js
--- a/handler.test.js
+++ b/handler.test.js
@@ -1,2 +1,3 @@
+ catch (e) { console.error(e); return res.status(200).json({ ok: true }); }
`)
	got := SLP024{}.Check(d)
	if len(got) != 0 {
		t.Fatalf("expected 0 findings in test file, got %d", len(got))
	}
}

func TestSLP024_IgnoresGoFiles(t *testing.T) {
	d := parseDiff(t, `diff --git a/handler.go b/handler.go
--- a/handler.go
+++ b/handler.go
@@ -3,3 +3,4 @@
+ if err != nil { log.Println(err); return 200 }
`)
	got := SLP024{}.Check(d)
	if len(got) != 0 {
		t.Fatalf("expected 0 findings for Go file, got %d", len(got))
	}
}

func TestSLP024_Description(t *testing.T) {
	r := SLP024{}
	if r.ID() != "SLP024" {
		t.Errorf("ID = %q", r.ID())
	}
	if !strings.Contains(strings.ToLower(r.Description()), "catch") {
		t.Errorf("description should mention catch: %q", r.Description())
	}
}