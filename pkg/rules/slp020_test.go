package rules

import (
	"strings"
	"testing"
)

func TestSLP020_InsecureRandomCrypto(t *testing.T) {
	tests := []struct {
		name string
		diff string
		want int
	}{
		{
			name: "go math/rand flagged",
			diff: `diff --git a/main.go b/main.go
--- a/main.go
+++ b/main.go
@@ -1,3 +1,4 @@
 import "math/rand"
+func gen() int { return rand.Intn(10) }
 }`,
			want: 1,
		},
		{
			name: "python random flagged",
			diff: `diff --git a/main.py b/main.py
--- a/main.py
+++ b/main.py
@@ -1,3 +1,4 @@
 import random
+def gen(): return random.randint(1, 10)
 }`,
			want: 1,
		},
		{
			name: "python secrets not flagged",
			diff: `diff --git a/main.py b/main.py
--- a/main.py
+++ b/main.py
@@ -1,3 +1,4 @@
 import secrets
+def gen(): return secrets.token_hex(16)
 }`,
			want: 0,
		},
		{
			name: "js Math.random flagged",
			diff: `diff --git a/main.js b/main.js
--- a/main.js
+++ b/main.js
@@ -1,3 +1,4 @@
 function gen() {
-    return 0;
+    return Math.random();
 }
 }`,
			want: 1,
		},
		{
			name: "java util Random flagged",
			diff: `diff --git a/Main.java b/Main.java
--- a/Main.java
+++ b/Main.java
@@ -1,3 +1,4 @@
 public class Main {
-    public int gen() { return 0; }
+    public int gen() { return new java.util.Random().nextInt(); }
 }
 }`,
			want: 1,
		},
		{
			name: "go crypto/md5 flagged",
			diff: `diff --git a/main.go b/main.go
--- a/main.go
+++ b/main.go
@@ -1,3 +1,4 @@
 import "crypto/md5"
+func hash(d []byte) [16]byte { return md5.Sum(d) }
 }`,
			want: 1,
		},
		{
			name: "python hashlib sha1 flagged",
			diff: `diff --git a/main.py b/main.py
--- a/main.py
+++ b/main.py
@@ -1,3 +1,4 @@
 import hashlib
+def hash(d): return hashlib.sha1(d).hexdigest()
 }`,
			want: 1,
		},
		{
			name: "security context upgrades to warn",
			diff: `diff --git a/main.go b/main.go
--- a/main.go
+++ b/main.go
@@ -1,3 +1,4 @@
 import "crypto/md5"
+func hashToken(token string) [16]byte { return md5.Sum([]byte(token)) }
 }`,
			want: 1,
		},
		{
			name: "test file not flagged",
			diff: `diff --git a/main_test.go b/main_test.go
--- a/main_test.go
+++ b/main_test.go
@@ -1,3 +1,4 @@
 import "math/rand"
+func TestGen(t *testing.T) { rand.Intn(10) }
 }`,
			want: 0,
		},
		{
			name: "non-source file not flagged",
			diff: `diff --git a/README.md b/README.md
--- a/README.md
+++ b/README.md
@@ -1,3 +1,4 @@
 # Project
-Use crypto
+Use crypto/md5 for hashing
 }`,
			want: 0,
		},
		{
			name: "go crypto/rand not flagged",
			diff: `diff --git a/main.go b/main.go
--- a/main.go
+++ b/main.go
@@ -1,3 +1,4 @@
 import "crypto/rand"
+func gen(n int) int { b := make([]byte, n); rand.Read(b); return int(b[0]) }
 }`,
			want: 0,
		},
		{
			name: "go crypto/rand + md5 flagged",
			diff: `diff --git a/main.go b/main.go
--- a/main.go
+++ b/main.go
@@ -1,3 +1,5 @@
 import "crypto/rand"
+import "crypto/md5"
+func hash(d []byte) [16]byte { return md5.Sum(d) }
 }`,
			want: 1,
		},
		{
			name: "go rand call without import not flagged",
			diff: `diff --git a/main.go b/main.go
--- a/main.go
+++ b/main.go
@@ -1,3 +1,4 @@
 func foo() {
-	return 0
+	return rand.Intn(10)
 }`,
			want: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			d := parseDiff(t, tt.diff)
			r := SLP020{}
			got := r.Check(d)
			if len(got) != tt.want {
				t.Fatalf("got %d findings, want %d; findings: %v", len(got), tt.want, got)
			}
			// Check security context severity upgrade.
			if tt.name == "security context upgrades to warn" && len(got) == 1 {
				if got[0].Severity != SeverityWarn {
					t.Errorf("expected warn severity with security context, got %v", got[0].Severity)
				}
			}
			if tt.name == "go math/rand flagged" && len(got) == 1 {
				if got[0].Severity != SeverityInfo {
					t.Errorf("expected info severity without security context, got %v", got[0].Severity)
				}
			}
		})
	}
}

func TestSLP020_IDAndDescription(t *testing.T) {
	var r SLP020
	if r.ID() != "SLP020" {
		t.Errorf("ID() = %q, want SLP020", r.ID())
	}
	if !strings.Contains(r.Description(), "insecure") {
		t.Errorf("Description() should mention insecure: %q", r.Description())
	}
}
