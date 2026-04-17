package rules

import (
	"strings"
	"testing"
)

func TestSLP018_OverlyBroadCatch(t *testing.T) {
	tests := []struct {
		name string
		diff string
		want int
	}{
		{
			name: "java catch Exception flagged",
			diff: `diff --git a/Main.java b/Main.java
--- a/Main.java
+++ b/Main.java
@@ -1,3 +1,4 @@
 public class Main {
-	public void run() {}
+	public void run() { try { doThing(); } catch (Exception e) { log(e); } }
 }`,
			want: 1,
		},
		{
			name: "java catch specific not flagged",
			diff: `diff --git a/Main.java b/Main.java
--- a/Main.java
+++ b/Main.java
@@ -1,3 +1,4 @@
 public class Main {
-	public void run() {}
+	public void run() { try { doThing(); } catch (IOException e) { log(e); } }
 }`,
			want: 0,
		},
		{
			name: "java catch Throwable flagged",
			diff: `diff --git a/Main.java b/Main.java
--- a/Main.java
+++ b/Main.java
@@ -1,3 +1,4 @@
 public class Main {
-	public void run() {}
+	public void run() { try { doThing(); } catch (Throwable t) { log(t); } }
 }`,
			want: 1,
		},
		{
			name: "python except bare flagged",
			diff: `diff --git a/main.py b/main.py
--- a/main.py
+++ b/main.py
@@ -1,3 +1,4 @@
 def foo():
-    pass
+    try: do_thing()
+    except: pass
 }`,
			want: 1,
		},
		{
			name: "python except Exception flagged",
			diff: `diff --git a/main.py b/main.py
--- a/main.py
+++ b/main.py
@@ -1,3 +1,4 @@
 def foo():
-    pass
+    try: do_thing()
+    except Exception: pass
 }`,
			want: 1,
		},
		{
			name: "python specific except not flagged",
			diff: `diff --git a/main.py b/main.py
--- a/main.py
+++ b/main.py
@@ -1,3 +1,4 @@
 def foo():
-    pass
+    try: do_thing()
+    except ValueError: pass
 }`,
			want: 0,
		},
		{
			name: "test file not flagged",
			diff: `diff --git a/MainTest.java b/MainTest.java
--- a/MainTest.java
+++ b/MainTest.java
@@ -1,3 +1,4 @@
 public class MainTest {
-	public void testFoo() {}
+	public void testFoo() { try { doThing(); } catch (Exception e) {} }
 }`,
			want: 0,
		},
		{
			name: "java final Exception flagged",
			diff: `diff --git a/Main.java b/Main.java
--- a/Main.java
+++ b/Main.java
@@ -1,3 +1,4 @@
 public class Main {
-	public void run() {}
+	public void run() { try { doThing(); } catch (final Exception e) {} }
 }`,
			want: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			d := parseDiff(t, tt.diff)
			r := SLP018{}
			got := r.Check(d)
			if len(got) != tt.want {
				t.Fatalf("got %d findings, want %d; findings: %v", len(got), tt.want, got)
			}
		})
	}
}

func TestSLP018_IDAndDescription(t *testing.T) {
	var r SLP018
	if r.ID() != "SLP018" {
		t.Errorf("ID() = %q, want SLP018", r.ID())
	}
	if r.DefaultSeverity() != SeverityWarn {
		t.Errorf("DefaultSeverity() = %v, want warn", r.DefaultSeverity())
	}
	if !strings.Contains(r.Description(), "broad") {
		t.Errorf("Description() should mention broad: %q", r.Description())
	}
}
