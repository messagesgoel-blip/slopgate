package rules

import (
	"strings"
	"testing"

	"github.com/messagesgoel-blip/slopgate/pkg/diff"
)

// parseDiff204 parses a unified diff string into a Diff for SLP204 tests.
func parseDiff204(t *testing.T, s string) *diff.Diff {
	t.Helper()
	d, err := diff.Parse(strings.NewReader(s))
	if err != nil {
		t.Fatalf("diff parse: %v", err)
	}
	return d
}

// TestSLP204_UncheckedErrorBeforeReturn tests detection of error variables
// assigned from function calls but never checked before returning success.
func TestSLP204_UncheckedErrorBeforeReturn(t *testing.T) {
	tests := []struct {
		name string
		diff string
		want int
	}{
		// Go patterns
		{
			name: "go_err_assigned_return_nil",
			diff: `diff --git a/handler.go b/handler.go
--- a/handler.go
+++ b/handler.go
@@ -1,4 +1,5 @@
 func process(db *sql.DB) error {
-	fmt.Println("ok")
+	err := db.Exec("INSERT INTO users (name) VALUES ('alice')")
+	return nil
 }`,
			want: 1,
		},
		{
			name: "go_err_assigned_return_true",
			diff: `diff --git a/processor.go b/processor.go
--- a/processor.go
+++ b/processor.go
@@ -1,4 +1,5 @@
 func process() bool {
-	return false
+	err := json.Unmarshal(data, &v)
+	return true
 }`,
			want: 1,
		},
		{
			name: "go_err_checked_not_flagged",
			diff: `diff --git a/handler.go b/handler.go
--- a/handler.go
+++ b/handler.go
@@ -1,4 +1,7 @@
 func process(db *sql.DB) error {
-	fmt.Println("ok")
+	err := db.Exec("INSERT INTO users (name) VALUES ('alice')")
+	if err != nil {
+		return err
+	}
+	return nil
 }`,
			want: 0,
		},
		{
			name: "go_err_checked_with_if_not_nil",
			diff: `diff --git a/handler.go b/handler.go
--- a/handler.go
+++ b/handler.go
@@ -1,4 +1,6 @@
 func process() error {
-	fmt.Println("ok")
+	err := doSomething()
+	if err != nil {
+		fmt.Println("failed:", err)
 	}
+	return nil
 }`,
			want: 0,
		},
		{
			name: "go_err_returned_directly",
			diff: `diff --git a/handler.go b/handler.go
--- a/handler.go
+++ b/handler.go
@@ -1,4 +1,5 @@
 func process() error {
-	fmt.Println("ok")
+	return doSomething()
 }`,
			want: 0,
		},
		// JS/TS patterns
		{
			name: "ts_err_assigned_return_success",
			diff: `diff --git a/handler.ts b/handler.ts
--- a/handler.ts
+++ b/handler.ts
@@ -1,3 +1,5 @@
 async function process(): Promise<{ ok: boolean }> {
-	console.log("start")
+	const err = await dbInsert("users", { name: "alice" })
+	return { ok: true }
 }`,
			want: 1,
		},
		{
			name: "js_err_assigned_return_null",
			diff: `diff --git a/handler.js b/handler.js
--- a/handler.js
+++ b/handler.js
@@ -1,4 +1,5 @@
 function process(req, res) {
-	res.send("ok")
+	const err = doAsyncWork()
+	return null
 }`,
			want: 1,
		},
		{
			name: "js_err_caught_not_flagged",
			diff: `diff --git a/handler.js b/handler.js
--- a/handler.js
+++ b/handler.js
@@ -1,4 +1,7 @@
 function process() {
-	console.log("ok")
+	const err = doSomething()
+	if (err) {
+		console.error(err)
 	}
+	return null
 }`,
			want: 0,
		},
		// Python patterns
		{
			name: "python_err_assigned_return_none",
			diff: `diff --git a/handler.py b/handler.py
--- a/handler.py
+++ b/handler.py
@@ -1,4 +1,5 @@
 def process():
-    print("ok")
+    err = do_something()
+    return None`,
			want: 1,
		},
		{
			name: "python_err_assigned_return_true",
			diff: `diff --git a/handler.py b/handler.py
--- a/handler.py
+++ b/handler.py
@@ -1,4 +1,5 @@
 def process():
-    print("ok")
+    err = do_something()
+    return True`,
			want: 1,
		},
		{
			name: "python_err_checked_not_flagged",
			diff: `diff --git a/handler.py b/handler.py
--- a/handler.py
+++ b/handler.py
@@ -1,4 +1,7 @@
 def process():
-    print("ok")
+    err = do_something()
+    if err is not None:
+        raise ValueError(err)
+    return True`,
			want: 0,
		},
		{
			name: "python_err_returned_not_flagged",
			diff: `diff --git a/handler.py b/handler.py
--- a/handler.py
+++ b/handler.py
@@ -1,4 +1,5 @@
 def process():
-    print("ok")
+    err = do_something()
+    return err`,
			want: 0,
		},
		// Java patterns
		{
			name: "java_err_assigned_return_true",
			diff: `diff --git a/Handler.java b/Handler.java
--- a/Handler.java
+++ b/Handler.java
@@ -1,4 +1,5 @@
 boolean process() {
-    return false;
+    Exception err = doSomething();
+    return true;
 }`,
			want: 1,
		},
		{
			name: "java_err_checked_not_flagged",
			diff: `diff --git a/Handler.java b/Handler.java
--- a/Handler.java
+++ b/Handler.java
@@ -1,4 +1,7 @@
 boolean process() {
-    return false;
+    Exception err = doSomething();
+    if (err != null) {
+        log.error("failed", err);
+    }
+    return true;
 }`,
			want: 0,
		},
		// Test files not flagged
		{
			name: "test_file_not_flagged",
			diff: `diff --git a/handler_test.go b/handler_test.go
--- a/handler_test.go
+++ b/handler_test.go
@@ -1,4 +1,5 @@
 // TestProcess validates SLP204's comment-line filtering.
func TestProcess(t *testing.T) {
-	t.Parallel()
+	err := doSomething()
+	_ = err
+	require.NoError(t, err)
 }`,
			want: 0,
		},
		// Comment lines not flagged
		{
			name: "comment_line_not_flagged",
			diff: `diff --git a/handler.go b/handler.go
--- a/handler.go
+++ b/handler.go
@@ -1,3 +1,4 @@
 func process() {
-	fmt.Println("ok")
+	// err := doSomething()
 }`,
			want: 0,
		},
		// Multiple unchecked errors in same hunk
		{
			name: "multiple_unchecked_errors_flagged",
			diff: `diff --git a/handler.go b/handler.go
--- a/handler.go
+++ b/handler.go
@@ -1,4 +1,6 @@
 func process() error {
-	fmt.Println("ok")
+	err1 := step1()
+	err2 := step2()
+	return nil
 }`,
			want: 2,
		},
		// err reassignment to nil should not trigger
		{
			name: "go_err_reassigned_nil_not_flagged",
			diff: `diff --git a/handler.go b/handler.go
--- a/handler.go
+++ b/handler.go
@@ -1,4 +1,6 @@
 func process() error {
-	err := doSomething()
-	return err
+	err = doOtherThing()
+	if err == nil {
 		return nil
+	}
+	return err
 }`,
			want: 0,
		},
		// Return success object
		{
			name: "ts_return_success_object",
			diff: `diff --git a/handler.ts b/handler.ts
--- a/handler.ts
+++ b/handler.ts
@@ -1,3 +1,5 @@
 function process(): { ok: boolean } {
-	return { ok: false }
+	const err = doSomething()
+	return { ok: true }
 }`,
			want: 1,
		},
		// Bug fix: inlineErrCheckPattern must match Go if-init without parens.
		// The `if err := f(); err != nil {` pattern has `err` but no outer
		// paren wrapping the entire condition. Previously this was silently
		// skipped, leaving err in pending and producing a false positive on
		// the later `return nil`.
		{
			name: "go_if_init_err_checked_no_false_positive",
			diff: `diff --git a/handler.go b/handler.go
--- a/handler.go
+++ b/handler.go
@@ -1,3 +1,5 @@
 func process() error {
-	fmt.Println("ok")
+	if err := doSomething(); err != nil {
+		return err
+	}
+	return nil
 }`,
			want: 0,
		},
		// Bug fix: isErrNameBlacklisted must not treat "!=" as "=" via substring match.
		// `err := someFunc(x != nil)` contains "= nil" as a substring inside "!=",
		// but the error IS captured and must not be blacklisted.
		{
			name: "go_err_assigned_with_neq_nil_in_args_not_blacklisted",
			diff: `diff --git a/handler.go b/handler.go
--- a/handler.go
+++ b/handler.go
@@ -1,4 +1,5 @@
 func process() error {
-	fmt.Println("ok")
+	err := doSomething(x != nil)
+	return nil
 }`,
			want: 1,
		},
		{
			name: "python_err_assigned_with_neq_none_in_args_not_blacklisted",
			diff: `diff --git a/handler.py b/handler.py
--- a/handler.py
+++ b/handler.py
@@ -1,4 +1,5 @@
 def process():
-    print("ok")
+    err = do_something(x is not None)
+    return None`,
			want: 1,
		},
		// err assigned from a for-range clause is not a function-call
		// capture and must not be flagged.
		{
			name: "go_err_from_range_not_flagged",
			diff: `diff --git a/handler.go b/handler.go
--- a/handler.go
+++ b/handler.go
@@ -1,3 +1,4 @@
 func process() {
-	fmt.Println("ok")
+	for _, err := range pending { return nil }
 }`,
			want: 0,
		},
		// err assigned from a plain variable (not a function call) must
		// not be flagged.
		{
			name: "go_err_assigned_non_call_not_flagged",
			diff: `diff --git a/handler.go b/handler.go
--- a/handler.go
+++ b/handler.go
@@ -1,4 +1,5 @@
 func process() error {
-	fmt.Println("ok")
+	err := someVar
+	return nil
 }`,
			want: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			d := parseDiff204(t, tt.diff)
			r := SLP204{}
			got := r.Check(d)
			if len(got) != tt.want {
				t.Fatalf("got %d findings, want %d; findings: %v", len(got), tt.want, got)
			}
		})
	}
}

// TestSLP204_IDAndDescription verifies SLP204's ID and default severity.
func TestSLP204_IDAndDescription(t *testing.T) {
	var r SLP204
	if r.ID() != "SLP204" {
		t.Errorf("ID() = %q, want SLP204", r.ID())
	}
	if r.DefaultSeverity() != SeverityBlock {
		t.Errorf("DefaultSeverity() = %v, want block", r.DefaultSeverity())
	}
	if !strings.Contains(r.Description(), "error") || !strings.Contains(r.Description(), "checked") {
		t.Errorf("Description() should mention error/checked: %q", r.Description())
	}
}
