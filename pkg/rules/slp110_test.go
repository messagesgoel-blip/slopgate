package rules

import "testing"

func TestSLP110_FiresOnSimilarFilesInSameDir(t *testing.T) {
	d := parseDiff(t, `diff --git a/handlers/user_handler.go b/handlers/user_handler.go
new file mode 100644
--- /dev/null
+++ b/handlers/user_handler.go
@@ -0,0 +1,5 @@
+package handlers
+
+import (
+    "database/sql"
+    "context"
+)
diff --git a/handlers/item_handler.go b/handlers/item_handler.go
new file mode 100644
--- /dev/null
+++ b/handlers/item_handler.go
@@ -0,0 +1,5 @@
+package handlers
+
+import (
+    "database/sql"
+    "context"
+)
`)
	got := SLP110{}.Check(d)
	if len(got) == 0 {
		t.Fatal("expected findings for similar files")
	}
}

func TestSLP110_NoFireOnDifferentFiles(t *testing.T) {
	d := parseDiff(t, `diff --git a/handlers/user.go b/handlers/user.go
new file mode 100644
--- /dev/null
+++ b/handlers/user.go
@@ -0,0 +1,5 @@
+package handlers
+
+import (
+    "net/http"
+)
diff --git a/handlers/db.go b/handlers/db.go
new file mode 100644
--- /dev/null
+++ b/handlers/db.go
@@ -0,0 +1,5 @@
+package handlers
+
+import (
+    "database/sql"
+)
`)
	got := SLP110{}.Check(d)
	if len(got) != 0 {
		t.Fatalf("expected 0 findings for different files, got %d", len(got))
	}
}

func TestSLP110_Description(t *testing.T) {
	r := SLP110{}
	if r.ID() != "SLP110" {
		t.Errorf("ID = %q", r.ID())
	}
	if r.Description() == "" {
		t.Errorf("Description is empty")
	}
	if r.DefaultSeverity() != SeverityWarn {
		t.Errorf("default severity should be warn")
	}
}
