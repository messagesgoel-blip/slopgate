package rules

import "testing"

func TestSLP124_FiresOnExternalCallWithoutValidation(t *testing.T) {
	d := parseDiff(t, `diff --git a/api/src/services/aiRouter.js b/api/src/services/aiRouter.js
--- a/api/src/services/aiRouter.js
+++ b/api/src/services/aiRouter.js
@@ -1,1 +1,3 @@
+const response = await litellm.chat.completions.create({ messages: req.body.messages })
`)
	got := SLP124{}.Check(d)
	if len(got) == 0 {
		t.Fatal("expected finding for unvalidated payload passed to external call")
	}
}

func TestSLP124_NoFireWhenValidationExists(t *testing.T) {
	d := parseDiff(t, `diff --git a/api/src/services/aiRouter.js b/api/src/services/aiRouter.js
--- a/api/src/services/aiRouter.js
+++ b/api/src/services/aiRouter.js
@@ -1,1 +1,6 @@
+if (!Array.isArray(req.body.messages) || req.body.messages.length == 0) return res.status(400).json({ ok: false })
+const response = await litellm.chat.completions.create({ messages: req.body.messages })
`)
	got := SLP124{}.Check(d)
	if len(got) != 0 {
		t.Fatalf("expected 0 findings when validation guard exists, got %d", len(got))
	}
}

func TestSLP124_Description(t *testing.T) {
	r := SLP124{}
	if r.ID() != "SLP124" {
		t.Errorf("ID = %q", r.ID())
	}
	if r.Description() == "" {
		t.Errorf("Description is empty")
	}
	if r.DefaultSeverity() != SeverityWarn {
		t.Errorf("default severity should be warn")
	}
}
