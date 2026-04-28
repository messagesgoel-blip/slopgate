package rules

import "testing"

func TestSLP121_FiresWithoutAccessGuard(t *testing.T) {
	d := parseDiff(t, `diff --git a/api/src/routes/providers.js b/api/src/routes/providers.js
--- a/api/src/routes/providers.js
+++ b/api/src/routes/providers.js
@@ -1,1 +1,5 @@
 router.patch('/providers/:id/share', async (req, res) => {
+  await db.update('shares', payload)
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
+  await db.update('shares', payload)
+  return res.json({ ok: true })
 })
`)
	got := SLP121{}.Check(d)
	if len(got) != 0 {
		t.Fatalf("expected 0 findings when access guard is present, got %d", len(got))
	}
}

func TestSLP121_WordBoundary_NoFalsePositiveOnSubstringIds(t *testing.T) {
	// tenant_id, member_id, access_token etc should NOT trigger — they're not standalone keywords
	// This test exercises the sensitive-mutation regex path by including a mutation verb
	d := parseDiff(t, `diff --git a/api/src/models/user.go b/api/src/models/user.go
--- a/api/src/models/user.go
+++ b/api/src/models/user.go
@@ -1,1 +1,3 @@
+	await db.update('tenant_id', payload)
`)
	got := SLP121{}.Check(d)
	if len(got) != 0 {
		t.Fatalf("expected 0 findings for tenant_id column definition, got %d", len(got))
	}
}

func TestSLP121_WordBoundary_FiresOnStandaloneKeywords(t *testing.T) {
	// standalone "tenant" in a mutation context SHOULD trigger
	d := parseDiff(t, `diff --git a/api/src/routes/tenant.go b/api/src/routes/tenant.go
--- a/api/src/routes/tenant.go
+++ b/api/src/routes/tenant.go
@@ -1,1 +1,5 @@
 router.patch('/tenant/:id', async (req, res) => {
+  await db.update('tenants', payload)
+  return res.json({ ok: true })
 })
`)
	got := SLP121{}.Check(d)
	if len(got) == 0 {
		t.Fatal("expected finding for standalone 'tenant' keyword without guard")
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
