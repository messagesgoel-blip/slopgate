package rules

import "testing"

func TestSLP123_FiresOnOffsetWithMutableTimeOrdering(t *testing.T) {
	d := parseDiff(t, `diff --git a/api/src/routes/family.js b/api/src/routes/family.js
--- a/api/src/routes/family.js
+++ b/api/src/routes/family.js
@@ -1,1 +1,3 @@
+const q = "SELECT * FROM family_activity ORDER BY created_at DESC LIMIT $1 OFFSET $2"
`)
	got := SLP123{}.Check(d)
	if len(got) == 0 {
		t.Fatal("expected finding for offset pagination with mutable ordering and no cursor/tiebreaker")
	}
}

func TestSLP123_NoFireWithCursorKeysetSignal(t *testing.T) {
	d := parseDiff(t, `diff --git a/api/src/routes/family.js b/api/src/routes/family.js
--- a/api/src/routes/family.js
+++ b/api/src/routes/family.js
@@ -1,1 +1,3 @@
+const q = "SELECT * FROM family_activity WHERE (created_at, id) < ($1, $2) ORDER BY created_at DESC, id DESC LIMIT $3 OFFSET $4"
`)
	got := SLP123{}.Check(d)
	if len(got) != 0 {
		t.Fatalf("expected 0 findings when cursor/tiebreaker signals exist, got %d", len(got))
	}
}

func TestSLP123_NoFalsePositive_JSOrderByChainWithId(t *testing.T) {
	d := parseDiff(t, `diff --git a/app/src/lib/files.ts b/app/src/lib/files.ts
--- a/app/src/lib/files.ts
+++ b/app/src/lib/files.ts
@@ -1,1 +1,3 @@
+files = files.orderBy('created_at', 'desc').orderBy('id', 'desc')
`)
	got := SLP123{}.Check(d)
	if len(got) != 0 {
		t.Fatalf("expected 0 findings for JS orderBy chain with id tiebreaker, got %d", len(got))
	}
}

func TestSLP123_NoFalsePositive_MultilineSqlOrderById(t *testing.T) {
	// Test that ORDER BY with id on separate line is still recognized (using \n in string)
	d := parseDiff(t, "diff --git a/api/src/routes/activity.js b/api/src/routes/activity.js\n--- a/api/src/routes/activity.js\n+++ b/api/src/routes/activity.js\n@@ -1,1 +1,3 @@\n+const query = \"ORDER BY created_at DESC, id DESC\"\n")
	got := SLP123{}.Check(d)
	if len(got) != 0 {
		t.Fatalf("expected 0 findings for SQL with id tiebreaker on same line, got %d", len(got))
	}
}

func TestSLP123_Description(t *testing.T) {
	r := SLP123{}
	if r.ID() != "SLP123" {
		t.Errorf("ID = %q", r.ID())
	}
	if r.Description() == "" {
		t.Errorf("Description is empty")
	}
	if r.DefaultSeverity() != SeverityWarn {
		t.Errorf("default severity should be warn")
	}
}
