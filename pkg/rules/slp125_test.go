package rules

import "testing"

func TestSLP125_FiresOnMutationWithoutAudit(t *testing.T) {
	d := parseDiff(t, `diff --git a/api/src/routes/providers.js b/api/src/routes/providers.js
--- a/api/src/routes/providers.js
+++ b/api/src/routes/providers.js
@@ -1,1 +1,3 @@
+await db.update('vps_connection_shares', { role: nextRole }).where({ id: shareId })
`)
	got := SLP125{}.Check(d)
	if len(got) == 0 {
		t.Fatal("expected finding when share/role mutation has no audit call")
	}
}

func TestSLP125_NoFireWithAuditCall(t *testing.T) {
	d := parseDiff(t, `diff --git a/api/src/routes/providers.js b/api/src/routes/providers.js
--- a/api/src/routes/providers.js
+++ b/api/src/routes/providers.js
@@ -1,1 +1,5 @@
+await db.update('vps_connection_shares', { role: nextRole }).where({ id: shareId })
+await appendActivity({ action: 'share_role_changed', shareId })
`)
	got := SLP125{}.Check(d)
	if len(got) != 0 {
		t.Fatalf("expected 0 findings when audit logging exists, got %d", len(got))
	}
}

func TestSLP125_Description(t *testing.T) {
	r := SLP125{}
	if r.ID() != "SLP125" {
		t.Errorf("ID = %q", r.ID())
	}
	if r.Description() == "" {
		t.Errorf("Description is empty")
	}
	if r.DefaultSeverity() != SeverityWarn {
		t.Errorf("default severity should be warn")
	}
}
