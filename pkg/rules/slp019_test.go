package rules

import (
	"strings"
	"testing"
)

func TestSLP019_UnreachableCode(t *testing.T) {
	tests := []struct {
		name string
		diff string
		want int
	}{
		{
			name: "unreachable after return",
			diff: `diff --git a/main.go b/main.go
--- a/main.go
+++ b/main.go
@@ -1,4 +1,6 @@
 func foo() int {
-	return 0
+	return 1
+	x := 2
 }`,
			want: 1,
		},
		{
			name: "unreachable after panic",
			diff: `diff --git a/main.go b/main.go
--- a/main.go
+++ b/main.go
@@ -1,4 +1,6 @@
 func foo() {
-	return
+	panic("boom")
+	cleanup()
 }`,
			want: 1,
		},
		{
			name: "unreachable after throw java",
			diff: `diff --git a/Main.java b/Main.java
--- a/Main.java
+++ b/Main.java
@@ -1,4 +1,6 @@
 void foo() {
-    return;
+    throw new Exception("boom");
+    cleanup();
 }`,
			want: 1,
		},
		{
			name: "reachable at shallower indent",
			diff: `diff --git a/main.go b/main.go
--- a/main.go
+++ b/main.go
@@ -1,4 +1,6 @@
 func foo() int {
-	return 0
+	return 1
+var x = 2
 }`,
			want: 0,
		},
		{
			name: "defer after return is unreachable",
			diff: `diff --git a/main.go b/main.go
--- a/main.go
+++ b/main.go
@@ -1,4 +1,6 @@
 func foo() {
-	return
+	return
+	defer cleanup()
 }`,
			want: 1,
		},
		{
			name: "closing brace after return not flagged",
			diff: `diff --git a/main.go b/main.go
--- a/main.go
+++ b/main.go
@@ -1,4 +1,6 @@
 func foo() int {
-	return 0
+	return 1
+}
 }`,
			want: 0,
		},
		{
			name: "unreachable after break",
			diff: `diff --git a/main.go b/main.go
--- a/main.go
+++ b/main.go
@@ -1,4 +1,6 @@
 func foo() {
-		fmt.Println("ok")
+		break
+		doMore()
 }`,
			want: 1,
		},
		{
			name: "test file not flagged",
			diff: `diff --git a/main_test.go b/main_test.go
--- a/main_test.go
+++ b/main_test.go
@@ -1,4 +1,6 @@
 func TestFoo(t *testing.T) {
-	return
+	return
+	x := 2
 }`,
			want: 0,
		},
		{
			name: "python raise unreachable",
			diff: `diff --git a/main.py b/main.py
--- a/main.py
+++ b/main.py
@@ -1,4 +1,6 @@
 def foo():
-    return 0
+    raise ValueError("boom")
+    cleanup()
 }`,
			want: 1,
		},
		{
			name: "multiline return object not flagged",
			diff: `diff --git a/main.go b/main.go
--- a/main.go
+++ b/main.go
@@ -1,4 +1,8 @@
 func foo() any {
+	return {
+		ok: true,
+		value: 1,
+	}
 }`,
			want: 0,
		},
		{
			name: "return cleanup callback not flagged",
			diff: `diff --git a/app.tsx b/app.tsx
--- a/app.tsx
+++ b/app.tsx
@@ -1,4 +1,7 @@
 function useThing() {
+  return () => {
+    cleanup()
+  }
 }`,
			want: 0,
		},
		{
			name: "multiline throw constructor not flagged",
			diff: `diff --git a/main.js b/main.js
--- a/main.js
+++ b/main.js
@@ -1,4 +1,8 @@
 function fail() {
+  throw new AppError("bad", {
+    field: "name",
+  })
 }`,
			want: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			d := parseDiff(t, tt.diff)
			r := SLP019{}
			got := r.Check(d)
			if len(got) != tt.want {
				t.Fatalf("got %d findings, want %d; findings: %v", len(got), tt.want, got)
			}
		})
	}
}

func TestSLP019_IDAndDescription(t *testing.T) {
	var r SLP019
	if r.ID() != "SLP019" {
		t.Errorf("ID() = %q, want SLP019", r.ID())
	}
	if r.DefaultSeverity() != SeverityWarn {
		t.Errorf("DefaultSeverity() = %v, want warn", r.DefaultSeverity())
	}
	if !strings.Contains(r.Description(), "unreachable") {
		t.Errorf("Description() should mention unreachable: %q", r.Description())
	}
}
