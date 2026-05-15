package rules

import (
	"strings"
	"testing"
)

func TestSLP207_TransactionWithoutRollback(t *testing.T) {
	tests := []struct {
		name string
		diff string
		want int
	}{
		{
			name: "go_tx_begin_without_rollback_on_error",
			diff: `diff --git a/handler.go b/handler.go
--- a/handler.go
+++ b/handler.go
@@ -1,6 +1,10 @@
 func process(ctx context.Context, db *sql.DB) error {
-	err := doThing(ctx)
+	tx, err := db.Begin(ctx)
+	if err != nil {
+		return err
+	}
+	_, err = tx.ExecContext(ctx, "INSERT INTO users (name) VALUES ($1)", "bob")
+	if err != nil {
+		return err
+	}
+	return tx.Commit()
 }`,
			want: 1,
		},
		{
			name: "go_tx_with_explicit_rollback_no_flag",
			diff: `diff --git a/handler.go b/handler.go
--- a/handler.go
+++ b/handler.go
@@ -1,6 +1,12 @@
 func process(ctx context.Context, db *sql.DB) error {
-	err := doThing(ctx)
+	tx, err := db.Begin(ctx)
+	if err != nil {
+		return err
+	}
+	defer tx.Rollback()
+	_, err = tx.ExecContext(ctx, "INSERT INTO users (name) VALUES ($1)", "bob")
+	if err != nil {
+		return err
+	}
+	return tx.Commit()
 }`,
			want: 0,
		},
		{
			name: "go_tx_with_commit_no_flag",
			diff: `diff --git a/handler.go b/handler.go
--- a/handler.go
+++ b/handler.go
@@ -1,4 +1,7 @@
 func process(ctx context.Context, db *sql.DB) error {
+	tx, err := db.Begin(ctx)
+	if err != nil {
+		return err
+	}
+	return tx.Commit()
 }`,
			want: 0,
		},
		{
			name: "python_begin_without_rollback",
			diff: `diff --git a/handler.py b/handler.py
--- a/handler.py
+++ b/handler.py
@@ -1,4 +1,9 @@
 def process(conn):
-    pass
+    cursor = conn.cursor()
+    cursor.execute("BEGIN")
+    try:
+        cursor.execute("INSERT INTO users (name) VALUES (%s)", ["bob"])
+    except Exception as e:
+        return None
+    conn.commit()
 `,
			want: 1,
		},
		{
			name: "python_begin_with_rollback_no_flag",
			diff: `diff --git a/handler.py b/handler.py
--- a/handler.py
+++ b/handler.py
@@ -1,4 +1,10 @@
 def process(conn):
-    pass
+    cursor = conn.cursor()
+    cursor.execute("BEGIN")
+    try:
+        cursor.execute("INSERT INTO users (name) VALUES (%s)", ["bob"])
+    except Exception as e:
+        conn.rollback()
+        return None
+    conn.commit()
 `,
			want: 0,
		},
		{
			name: "java_setautocommit_false_without_rollback",
			diff: `diff --git a/Handler.java b/Handler.java
--- a/Handler.java
+++ b/Handler.java
@@ -1,6 +1,13 @@
 void process(Connection conn) throws SQLException {
-    doThing();
+    conn.setAutoCommit(false);
+    try {
+        PreparedStatement ps = conn.prepareStatement("INSERT INTO users (name) VALUES (?)");
+        ps.setString(1, "bob");
+        ps.executeUpdate();
+        conn.commit();
+    } catch (SQLException e) {
+        throw e;
+    }
 }`,
			want: 1,
		},
		{
			name: "bare_sql_begin_without_rollback",
			diff: `diff --git a/migration.sql b/migration.sql
--- a/migration.sql
+++ b/migration.sql
@@ -1 +1,5 @@
 BEGIN;
+INSERT INTO users (name) VALUES ('bob');
+-- missing ROLLBACK on error path
+COMMIT;
`,
			want: 0, // has COMMIT so happy path is covered
		},
		{
			name: "test_file_not_flagged",
			diff: `diff --git a/handler_test.go b/handler_test.go
--- a/handler_test.go
+++ b/handler_test.go
@@ -1,3 +1,8 @@
 func TestProcess(t *testing.T) {
-    db := setupDB()
+    tx, err := db.Begin()
+    if err != nil {
+        return err
+    }
+    return tx.Commit()
 }`,
			want: 0,
		},
		{
			name: "comment_not_flagged",
			diff: `diff --git a/handler.go b/handler.go
--- a/handler.go
+++ b/handler.go
@@ -1,3 +1,5 @@
 func process() {
-    fmt.Println("ok")
+    // tx, err := db.Begin(ctx)
+    return nil
+    // return err
 }`,
			want: 0,
		},
		{
			name: "no_false_positive_on_new_code",
			diff: `diff --git a/handler.go b/handler.go
--- a/handler.go
+++ b/handler.go
@@ -1,2 +1,3 @@
 func process() {
-	fmt.Println("ok")
+	fmt.Println("hello world")
 }`,
			want: 0,
		},
		{
			name: "go_db_begin_without_ctx",
			diff: `diff --git a/handler.go b/handler.go
--- a/handler.go
+++ b/handler.go
@@ -1,5 +1,8 @@
 func process(db *sql.DB) error {
-	return db.Exec("DELETE FROM users")
+	err := db.Begin()
+	if err != nil {
+		return err
+	}
+	return nil
 }`,
			want: 1,
		},
		{
			name: "python_connection_begin_without_rollback",
			diff: `diff --git a/handler.py b/handler.py
--- a/handler.py
+++ b/handler.py
@@ -1,4 +1,9 @@
 def process(conn):
-    pass
+    conn.begin()
+    try:
+        conn.execute("INSERT INTO users (name) VALUES (%s)", ["bob"])
+    except:
+        return None
+    conn.commit()
 `,
			want: 1,
		},
		{
			name: "js_sequelize_transaction_without_rollback",
			diff: `diff --git a/handler.js b/handler.js
--- a/handler.js
+++ b/handler.js
@@ -1,5 +1,12 @@
 async function process(db) {
-    return res.status(200).json({ ok: true });
+    const t = await db.transaction();
+    try {
+        await db.query("INSERT INTO users (name) VALUES ('bob')", { transaction: t });
+        return res.status(201).json({ ok: true });
+    } catch (err) {
+        return res.status(500).json({ error: err.message });
+    }
+    await t.commit();
 }`,
			want: 1,
		},
		{
			name: "multiple_begins_one_finding_per_begin",
			diff: `diff --git a/handler.go b/handler.go
--- a/handler.go
+++ b/handler.go
@@ -1,4 +1,14 @@
 func process(db *sql.DB) error {
-	return nil
+	tx1, err := db.Begin()
+	if err != nil {
+		return err
+	}
+	tx2, err := db.Begin()
+	if err != nil {
+		return err
+	}
+	return nil
 }`,
			want: 2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			d := parseDiff(t, tt.diff)
			r := SLP207{}
			got := r.Check(d)
			if len(got) != tt.want {
				t.Fatalf("got %d findings, want %d; findings: %v", len(got), tt.want, got)
			}
		})
	}
}

func TestSLP207_IDAndDescription(t *testing.T) {
	var r SLP207
	if r.ID() != "SLP207" {
		t.Errorf("ID() = %q, want SLP207", r.ID())
	}
	if r.DefaultSeverity() != SeverityBlock {
		t.Errorf("DefaultSeverity() = %v, want block", r.DefaultSeverity())
	}
	desc := r.Description()
	if !strings.Contains(desc, "transaction") || !strings.Contains(desc, "rollback") {
		t.Errorf("Description() should mention transaction/rollback: %q", desc)
	}
}
