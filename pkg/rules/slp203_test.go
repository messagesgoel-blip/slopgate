package rules

import (
	"strings"
	"testing"

	"github.com/messagesgoel-blip/slopgate/pkg/diff"
)

func parseDiff203(t *testing.T, s string) *diff.Diff {
	t.Helper()
	d, err := diff.Parse(strings.NewReader(s))
	if err != nil {
		t.Fatalf("diff parse: %v", err)
	}
	return d
}

func TestSLP203_InsertWithoutConflictHandling(t *testing.T) {
	tests := []struct {
		name string
		diff string
		want int
	}{
		{
			name: "go_insert_no_conflict",
			diff: `diff --git a/handler.go b/handler.go
--- a/handler.go
+++ b/handler.go
@@ -1,3 +1,4 @@
 func process(db *sql.DB) {
-	fmt.Println("ok")
+	db.Exec("INSERT INTO users (name, email) VALUES ('alice', 'alice@example.com')")
 }`,
			want: 1,
		},
		{
			name: "go_insert_with_on_conflict_not_flagged",
			diff: `diff --git a/handler.go b/handler.go
--- a/handler.go
+++ b/handler.go
@@ -1,3 +1,4 @@
 func process(db *sql.DB) {
-	fmt.Println("ok")
+	db.Exec("INSERT INTO users (name, email) VALUES ('alice', 'alice@example.com') ON CONFLICT (email) DO UPDATE SET name = excluded.name")
 }`,
			want: 0,
		},
		{
			name: "python_insert_no_conflict",
			diff: `diff --git a/handler.py b/handler.py
--- a/handler.py
+++ b/handler.py
@@ -1,3 +1,4 @@
 def process(cur):
-    print("ok")
+    cur.execute("INSERT INTO users (name) VALUES (%s)", ("alice",))
 }`,
			want: 1,
		},
		{
			name: "python_insert_with_on_conflict_not_flagged",
			diff: `diff --git a/handler.py b/handler.py
--- a/handler.py
+++ b/handler.py
@@ -1,3 +1,4 @@
 def process(cur):
-    print("ok")
+    cur.execute("INSERT INTO users (name) VALUES (%s) ON CONFLICT (name) DO NOTHING", ("alice",))
 }`,
			want: 0,
		},
		{
			name: "test_file_not_flagged",
			diff: `diff --git a/handler_test.go b/handler_test.go
--- a/handler_test.go
+++ b/handler_test.go
@@ -1,3 +1,4 @@
 func TestInsert(t *testing.T) {
-	t.Parallel()
+	db.Exec("INSERT INTO users (name) VALUES ('test')")
 }`,
			want: 0,
		},
		{
			name: "comment_line_not_flagged",
			diff: `diff --git a/handler.go b/handler.go
--- a/handler.go
+++ b/handler.go
@@ -1,3 +1,4 @@
 func process(db *sql.DB) {
-	fmt.Println("ok")
+	// INSERT INTO users (name) VALUES ('hacker')
 }`,
			want: 0,
		},
		{
			name: "closing_brace_not_flagged",
			diff: `diff --git a/handler.go b/handler.go
--- a/handler.go
+++ b/handler.go
@@ -1,4 +1,5 @@
 func process(db *sql.DB) {
-	if err != nil {
-		fmt.Println("ok")
+	if err != nil {
+		return err
+	}
+	db.Exec("INSERT INTO users (name) VALUES ('alice')")
 }`,
			want: 1,
		},
		{
			name: "mysql_insert_with_duplicate_key_not_flagged",
			diff: `diff --git a/handler.go b/handler.go
--- a/handler.go
+++ b/handler.go
@@ -1,3 +1,4 @@
 func process(db *sql.DB) {
-	fmt.Println("ok")
+	db.Exec("INSERT INTO users (name) VALUES ('alice') ON DUPLICATE KEY UPDATE name = VALUES(name)")
 }`,
			want: 0,
		},
		{
			name: "sqlite_insert_or_replace_not_flagged",
			diff: `diff --git a/handler.go b/handler.go
--- a/handler.go
+++ b/handler.go
@@ -1,3 +1,4 @@
 func process(db *sql.DB) {
-	fmt.Println("ok")
+	db.Exec("INSERT OR REPLACE INTO users (name) VALUES ('alice')")
 }`,
			want: 0,
		},
		{
			name: "sqlite_insert_or_ignore_not_flagged",
			diff: `diff --git a/handler.go b/handler.go
--- a/handler.go
+++ b/handler.go
@@ -1,3 +1,4 @@
 func process(db *sql.DB) {
-	fmt.Println("ok")
+	db.Exec("INSERT OR IGNORE INTO users (name) VALUES ('alice')")
 }`,
			want: 0,
		},
		{
			name: "select_statement_not_flagged",
			diff: `diff --git a/handler.go b/handler.go
--- a/handler.go
+++ b/handler.go
@@ -1,3 +1,4 @@
 func process(db *sql.DB) {
-	fmt.Println("ok")
+	rows, err := db.Query("SELECT * FROM users WHERE id = 1")
+	_ = rows
+	_ = err
 }`,
			want: 0,
		},
		{
			name: "update_statement_not_flagged",
			diff: `diff --git a/handler.go b/handler.go
--- a/handler.go
+++ b/handler.go
@@ -1,3 +1,4 @@
 func process(db *sql.DB) {
-	fmt.Println("ok")
+	db.Exec("UPDATE users SET name = 'bob' WHERE id = 1")
 }`,
			want: 0,
		},
		{
			name: "go_mustexec_insert_with_conflict_not_flagged",
			diff: `diff --git a/handler.go b/handler.go
--- a/handler.go
+++ b/handler.go
@@ -1,3 +1,4 @@
 func process(db *sql.DB) {
-	fmt.Println("ok")
+	db.MustExec("INSERT INTO users (name) VALUES ('alice') ON CONFLICT DO NOTHING")
 }`,
			want: 0,
		},
		{
			name: "java_insert_no_conflict",
			diff: `diff --git a/Handler.java b/Handler.java
--- a/Handler.java
+++ b/Handler.java
@@ -1,4 +1,5 @@
 void process(Statement stmt) {
-    System.out.println("ok");
+    stmt.execute("INSERT INTO users (name) VALUES ('alice')");
 }`,
			want: 1,
		},
		{
			name: "java_insert_with_on_conflict_not_flagged",
			diff: `diff --git a/Handler.java b/Handler.java
--- a/Handler.java
+++ b/Handler.java
@@ -1,4 +1,5 @@
 void process(Statement stmt) {
-    System.out.println("ok");
+    stmt.execute("INSERT INTO users (name) VALUES ('alice') ON CONFLICT (name) DO NOTHING");
 }`,
			want: 0,
		},
		{
			name: "multiple_inserts_flagged",
			diff: `diff --git a/handler.go b/handler.go
--- a/handler.go
+++ b/handler.go
@@ -1,3 +1,5 @@
 func process(db *sql.DB) {
-	fmt.Println("ok")
+	db.Exec("INSERT INTO users (name) VALUES ('alice')")
+	db.Exec("INSERT INTO posts (title) VALUES ('hello')")
 }`,
			want: 2,
		},
		{
			name: "merge_statement_not_flagged",
			diff: `diff --git a/handler.go b/handler.go
--- a/handler.go
+++ b/handler.go
@@ -1,3 +1,4 @@
 func process(db *sql.DB) {
-	fmt.Println("ok")
+	db.Exec("MERGE INTO users USING src ON (users.id = src.id) WHEN MATCHED THEN UPDATE SET name = src.name")
 }`,
			want: 0,
		},
		{
			name: "upsert_keyword_not_flagged",
			diff: `diff --git a/handler.go b/handler.go
--- a/handler.go
+++ b/handler.go
@@ -1,3 +1,4 @@
 func process(db *sql.DB) {
-	fmt.Println("ok")
+	db.Exec("UPSERT INTO users (name) VALUES ('alice')")
 }`,
			want: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			d := parseDiff203(t, tt.diff)
			r := SLP203{}
			got := r.Check(d)
			if len(got) != tt.want {
				t.Fatalf("got %d findings, want %d; findings: %v", len(got), tt.want, got)
			}
		})
	}
}

func TestSLP203_IDAndDescription(t *testing.T) {
	var r SLP203
	if r.ID() != "SLP203" {
		t.Errorf("ID() = %q, want SLP203", r.ID())
	}
	if r.DefaultSeverity() != SeverityBlock {
		t.Errorf("DefaultSeverity() = %v, want block", r.DefaultSeverity())
	}
	if !strings.Contains(r.Description(), "INSERT") && !strings.Contains(r.Description(), "unique") {
		t.Errorf("Description() should mention INSERT/unique: %q", r.Description())
	}
}
