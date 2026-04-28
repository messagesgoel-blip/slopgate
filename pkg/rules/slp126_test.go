package rules

import "testing"

func TestSLP126_FiresOnReferenceWithoutIndex(t *testing.T) {
	d := parseDiff(t, `diff --git a/api/migrations/058_vps_connection_shares/migration.sql b/api/migrations/058_vps_connection_shares/migration.sql
--- a/api/migrations/058_vps_connection_shares/migration.sql
+++ b/api/migrations/058_vps_connection_shares/migration.sql
@@ -0,0 +1,5 @@
+CREATE TABLE vps_connection_shares (
+  id uuid primary key,
+  vps_connection_id uuid not null references vps_connections(id)
+);
`)
	got := SLP126{}.Check(d)
	if len(got) == 0 {
		t.Fatal("expected finding when reference column has no matching index")
	}
}

func TestSLP126_NoFireWhenIndexAdded(t *testing.T) {
	d := parseDiff(t, `diff --git a/api/migrations/058_vps_connection_shares/migration.sql b/api/migrations/058_vps_connection_shares/migration.sql
--- a/api/migrations/058_vps_connection_shares/migration.sql
+++ b/api/migrations/058_vps_connection_shares/migration.sql
@@ -0,0 +1,7 @@
+CREATE TABLE vps_connection_shares (
+  id uuid primary key,
+  vps_connection_id uuid not null references vps_connections(id)
+);
+CREATE INDEX idx_vps_connection_shares_vps_connection_id ON vps_connection_shares (vps_connection_id);
`)
	got := SLP126{}.Check(d)
	if len(got) != 0 {
		t.Fatalf("expected 0 findings when index is added, got %d", len(got))
	}
}

func TestSLP126_Description(t *testing.T) {
	r := SLP126{}
	if r.ID() != "SLP126" {
		t.Errorf("ID = %q", r.ID())
	}
	if r.Description() == "" {
		t.Errorf("Description is empty")
	}
	if r.DefaultSeverity() != SeverityWarn {
		t.Errorf("default severity should be warn")
	}
}
