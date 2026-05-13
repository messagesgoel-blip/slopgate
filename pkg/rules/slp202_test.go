package rules

import (
	"strings"
	"testing"
)

func TestSLP202_NilDerefOnGuardedVar(t *testing.T) {
	tests := []struct {
		name string
		diff string
		want int
	}{
		{
			name: "go_guard_removed_then_deref",
			diff: `diff --git a/handler.go b/handler.go
--- a/handler.go
+++ b/handler.go
@@ -1,7 +1,8 @@
 func process(req *Request) error {
-	if req == nil {
-		return fmt.Errorf("nil request")
-	}
+	if req == nil {
+		return nil
+	}
+	result := req.Body.Data
+	return result
 }`,
			want: 1,
		},
		{
			name: "go_chained_nil_deref",
			diff: `diff --git a/handler.go b/handler.go
--- a/handler.go
+++ b/handler.go
@@ -1,2 +1,3 @@
 func process() {
-	fmt.Println("ok")
+	resp := srv.GetConfig().Logger().Name()
 }`,
			want: 0, // No guard was removed, so this isn't guard-removal —
		}, // it's an unchained new line, not a guard-removal pattern.
		{
			name: "js_guard_removed_deref",
			diff: `diff --git a/app.js b/app.js
--- a/app.js
+++ b/app.js
@@ -1,5 +1,6 @@
 function handle(err) {
-	if (err !== null) {
-		console.log(err.message);
-	}
+	if (err !== null) {
+		console.log(err.message);
+	}
+	console.log(err.stack);
 }`,
			want: 1,
		},
		{
			name: "go_guard_present_no_flag",
			diff: `diff --git a/handler.go b/handler.go
--- a/handler.go
+++ b/handler.go
@@ -1,4 +1,5 @@
 func process(req *Request) error {
+	if req == nil {
+		return fmt.Errorf("nil request")
+	}
	result := req.Body.Data
	return result
 }`,
			want: 0,
		},
		{
			name: "go_inline_nil_check_not_flagged",
			diff: `diff --git a/handler.go b/handler.go
--- a/handler.go
+++ b/handler.go
@@ -1,2 +1,3 @@
 func process() {
-	fmt.Println("ok")
+	if req != nil { fmt.Println(req.Name) }
 }`,
			want: 0,
		},
		{
			name: "python_guard_removed_deref",
			diff: `diff --git a/handler.py b/handler.py
--- a/handler.py
+++ b/handler.py
@@ -1,5 +1,6 @@
 def process(req):
-	if req is not None:
-		print(req.name)
-	else:
-		raise ValueError("no req")
+	if req is not None:
+		print(req.name)
+	print(req.age)
 `,
			want: 1,
		},
		{
			name: "test_file_not_flagged",
			diff: `diff --git a/handler_test.go b/handler_test.go
--- a/handler_test.go
+++ b/handler_test.go
@@ -1,3 +1,5 @@
 func TestProcess(t *testing.T) {
-	req := NewRequest()
-	process(req)
+	req := NewRequest()
+	process(req)
+	t.Log(req.Body.Data)
 }`,
			want: 0,
		},
		{
			name: "comment_line_not_flagged",
			diff: `diff --git a/handler.go b/handler.go
--- a/handler.go
+++ b/handler.go
@@ -1,2 +1,3 @@
 func process() {
-	fmt.Println("ok")
+	// req.Body.Data would panic here
 }`,
			want: 0,
		},
		{
			name: "closing_brace_not_flagged",
			diff: `diff --git a/handler.go b/handler.go
--- a/handler.go
+++ b/handler.go
@@ -1,4 +1,6 @@
 func process() {
-	if x != nil {
+	if x != nil {
+		x.Do()
	}
 }`,
			want: 0,
		},
		{
			name: "java_guard_removed_deref",
			diff: `diff --git a/Handler.java b/Handler.java
--- a/Handler.java
+++ b/Handler.java
@@ -1,5 +1,6 @@
 void process(Request req) {
-	if (req != null) {
-		System.out.println(req.name);
-	}
+	if (req != null) {
+		System.out.println(req.name);
+	}
+	System.out.println(req.id);
 }`,
			want: 1,
		},
		{
			name: "no_false_positive_on_new_code_without_guard",
			diff: `diff --git a/handler.go b/handler.go
--- a/handler.go
+++ b/handler.go
@@ -1,2 +1,3 @@
 func process() {
-	fmt.Println("ok")
+	fmt.Println(srv.Config().Logger().Name())
 }`,
			want: 0,
		},
		{
			name: "go_null_equality_check_reversed",
			diff: `diff --git a/handler.go b/handler.go
--- a/handler.go
+++ b/handler.go
@@ -1,6 +1,7 @@
 func process(req *Request) error {
-	if nil != req {
-		return fmt.Errorf("bad")
-	}
+	if nil != req {
+		return nil
+	}
+	_ = req.Body
 }`,
			want: 1,
		},
		{
			name: "optional_chaining_not_flagged",
			diff: `diff --git a/app.ts b/app.ts
--- a/app.ts
+++ b/app.ts
@@ -1,2 +1,3 @@
 function handle() {
-	console.log("start");
+	const name = req?.body?.name;
 }`,
			want: 0,
		},
		{
			name: "python_if_guard_on_same_line_not_flagged",
			diff: `diff --git a/handler.py b/handler.py
--- a/handler.py
+++ b/handler.py
@@ -1,2 +1,3 @@
 def process(req):
-	pass
+	if req is not None: print(req.name)
 `,
			want: 0,
		},
		{
			name: "rust_if_let_some_not_flagged",
			diff: `diff --git a/handler.rs b/handler.rs
--- a/handler.rs
+++ b/handler.rs
@@ -1,2 +1,3 @@
 fn process(opt: Option<String>) {
-	println!("start");
+	if let Some(val) = opt { println!("{}", val); }
 }`,
			want: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			d := parseDiff(t, tt.diff)
			r := SLP202{}
			got := r.Check(d)
			if len(got) != tt.want {
				t.Fatalf("got %d findings, want %d; findings: %v", len(got), tt.want, got)
			}
		})
	}
}

func TestSLP202_IDAndDescription(t *testing.T) {
	var r SLP202
	if r.ID() != "SLP202" {
		t.Errorf("ID() = %q, want SLP202", r.ID())
	}
	if r.DefaultSeverity() != SeverityBlock {
		t.Errorf("DefaultSeverity() = %v, want block", r.DefaultSeverity())
	}
	if !strings.Contains(r.Description(), "nil") && !strings.Contains(r.Description(), "dereference") {
		t.Errorf("Description() should mention nil/dereference: %q", r.Description())
	}
}
