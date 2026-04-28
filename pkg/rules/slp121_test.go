package rules

import "testing"

func TestSLP121_FiresWithoutAccessGuard(t *testing.T) {
	d := parseDiff(t, `diff --git a/api/src/routes/providers.js b/api/src/routes/providers.js
--- a/api/src/routes/providers.js
+++ b/api/src/routes/providers.js
@@ -1,1 +1,5 @@
 router.patch('/providers/:id/share', async (req, res) => {
+  await db.update('vps_connection_shares', payload)
+  return res.json({ ok: true })
 })
`)
	got := SLP121{}.Check(d)
	if len(got) == 0 {
		t.Fatal("expected finding when access mutation has no guard")
	}
}

func TestSLP121_NoFireWithAccessGuard(t *testing.T) {
	d := parseDiff(t, `diff --git a/api/src/routes/providers.js b/api/src/routes/providers.js
--- a/api/src/routes/providers.js
+++ b/api/src/routes/providers.js
@@ -1,1 +1,6 @@
 router.patch('/providers/:id/share', async (req, res) => {
+  await requireVpsAccess(req, res, 'admin')
+  await db.update('vps_connection_shares', payload)
+  return res.json({ ok: true })
 })
`)
	got := SLP121{}.Check(d)
	if len(got) != 0 {
		t.Fatalf("expected 0 findings when access guard is present, got %d", len(got))
	}
}

func TestSLP121_Description(t *testing.T) {
	r := SLP121{}
	if r.ID() != "SLP121" {
		t.Errorf("ID = %q", r.ID())
	}
	if r.Description() == "" {
		t.Errorf("Description is empty")
	}
	if r.DefaultSeverity() != SeverityWarn {
		t.Errorf("default severity should be warn")
	}
}
