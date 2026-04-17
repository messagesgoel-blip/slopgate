package rules

import (
	"strings"
	"testing"
)

func TestSLP016_ShadowVariable(t *testing.T) {
	tests := []struct {
		name string
		diff string
		want int
	}{
		{
			name: "inner scope shadow flagged",
			diff: `diff --git a/main.go b/main.go
--- a/main.go
+++ b/main.go
@@ -1,4 +1,7 @@
 func foo() {
-	x := 1
+	x := 1
+	if true {
+		x := 2
+	}
 }`,
			want: 1,
		},
		{
			name: "same scope reassign not flagged",
			diff: `diff --git a/main.go b/main.go
--- a/main.go
+++ b/main.go
@@ -1,4 +1,5 @@
 func foo() {
-	x := 1
+	x := 1
+	x := 2
 }`,
			want: 0,
		},
		{
			name: "loop variable i not flagged",
			diff: `diff --git a/main.go b/main.go
--- a/main.go
+++ b/main.go
@@ -1,4 +1,7 @@
 func foo() {
-	for i := 0; i < 10; i++ {}
+	for i := 0; i < 10; i++ {
+		for j := range items {
+			i := j
+		}
+	}
 }`,
			want: 0,
		},
		{
			name: "err shadow at info level",
			diff: `diff --git a/main.go b/main.go
--- a/main.go
+++ b/main.go
@@ -1,4 +1,7 @@
 func foo() error {
-	err := doThing()
+	err := doThing()
+	if err != nil {
+		err := doOther()
+	}
 }`,
			want: 1,
		},
		{
			name: "test file not flagged",
			diff: `diff --git a/main_test.go b/main_test.go
--- a/main_test.go
+++ b/main_test.go
@@ -1,4 +1,7 @@
 func TestFoo(t *testing.T) {
-	x := 1
+	x := 1
+	if true {
+		x := 2
+	}
 }`,
			want: 0,
		},
		{
			name: "java shadow flagged",
			diff: `diff --git a/Main.java b/Main.java
--- a/Main.java
+++ b/Main.java
@@ -1,4 +1,7 @@
 public void foo() {
-    int x = 1;
+    int x = 1;
+    if (true) {
+        int x = 2;
+    }
 }`,
			want: 1,
		},
		{
			name: "different name not flagged",
			diff: `diff --git a/main.go b/main.go
--- a/main.go
+++ b/main.go
@@ -1,4 +1,7 @@
 func foo() {
-	x := 1
+	x := 1
+	if true {
+		y := 2
+	}
 }`,
			want: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			d := parseDiff(t, tt.diff)
			r := SLP016{}
			got := r.Check(d)
			if len(got) != tt.want {
				t.Fatalf("got %d findings, want %d; findings: %v", len(got), tt.want, got)
			}
			// Check err shadow severity.
			if tt.name == "err shadow at info level" && len(got) == 1 {
				if got[0].Severity != SeverityInfo {
					t.Errorf("expected info severity for err shadow, got %v", got[0].Severity)
				}
			}
		})
	}
}

func TestSLP016_IDAndDescription(t *testing.T) {
	var r SLP016
	if r.ID() != "SLP016" {
		t.Errorf("ID() = %q, want SLP016", r.ID())
	}
	if r.DefaultSeverity() != SeverityWarn {
		t.Errorf("DefaultSeverity() = %v, want warn", r.DefaultSeverity())
	}
	if !strings.Contains(r.Description(), "shadow") {
		t.Errorf("Description() should mention shadow: %q", r.Description())
	}
}
