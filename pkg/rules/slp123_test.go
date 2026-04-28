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
	// Include an offset clause to trigger the offset check, then verify tiebreaker prevents firing
	d := parseDiff(t, `diff --git a/app/src/lib/files.ts b/app/src/lib/files.ts
--- a/app/src/lib/files.ts
+++ b/app/src/lib/files.ts
@@ -1,1 +1,3 @@
+query = query.offset(10)
+files = files.orderBy('created_at', 'desc').orderBy('id', 'desc')
`)
	got := SLP123{}.Check(d)
	if len(got) != 0 {
		t.Fatalf("expected 0 findings for JS orderBy chain with id tiebreaker, got %d", len(got))
	}
}

func TestSLP123_NoFalsePositive_MultilineSqlOrderById(t *testing.T) {
	// Simulate multiline SQL with OFFSET and split ORDER BY across lines to test tiebreaker detection
	d := parseDiff(t, `diff --git a/api/src/routes/activity.js b/api/src/routes/activity.js
--- a/api/src/routes/activity.js
+++ b/api/src/routes/activity.js
@@ -1,1 +1,4 @@
+const offset = 10
+const query = "ORDER BY created_at DESC, " +
+              "id DESC OFFSET $1"
`)
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
