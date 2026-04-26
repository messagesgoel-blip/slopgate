package rules

import "testing"

func TestSLP107_FiresOnCleanupOnlyInErrorPath(t *testing.T) {
	d := parseDiff(t, `diff --git a/handler.go b/handler.go
--- a/handler.go
+++ b/handler.go
@@ -1,1 +1,5 @@
+  conn, err := net.Dial("tcp", addr)
+  if err != nil {
+      conn.Close()
+      return err
+  }
`)
	got := SLP107{}.Check(d)
	if len(got) == 0 {
		t.Fatal("expected findings for cleanup only in error path")
	}
}

func TestSLP107_FiresOnSingleLineErrorBlock(t *testing.T) {
	d := parseDiff(t, `diff --git a/handler.go b/handler.go
--- a/handler.go
+++ b/handler.go
@@ -1,1 +1,3 @@
+  if err != nil { conn.Close(); return err }
`)
	got := SLP107{}.Check(d)
	if len(got) == 0 {
		t.Fatal("expected findings for single-line error block cleanup")
	}
}

func TestSLP107_NoFireOnCleanupInNormalPath(t *testing.T) {
	d := parseDiff(t, `diff --git a/handler.go b/handler.go
--- a/handler.go
+++ b/handler.go
@@ -1,7 +1,11 @@
+  if err != nil {
+      conn.Close()
+      return err
+  }
+  
+  defer conn.Close()
+`)
	got := SLP107{}.Check(d)
	if len(got) != 0 {
		t.Fatalf("expected 0 findings for normal cleanup in mixed path, got %d", len(got))
	}
}

func TestSLP107_NoFireWhenEarlierDeferMatchesSameResource(t *testing.T) {
	d := parseDiff(t, `diff --git a/handler.go b/handler.go
--- a/handler.go
+++ b/handler.go
@@ -1,7 +1,11 @@
+  defer conn.Close()
+  if err != nil {
+      conn.Close()
+      return err
+  }
`)
	got := SLP107{}.Check(d)
	if len(got) != 0 {
		t.Fatalf("expected 0 findings when earlier defer matches same resource, got %d", len(got))
	}
}

func TestSLP107_PythonExceptBlock(t *testing.T) {
	d := parseDiff(t, `diff --git a/handler.py b/handler.py
--- a/handler.py
+++ b/handler.py
@@ -1,5 +1,9 @@
+def handle():
+    try:
+        do_something()
+    except Exception:
+        conn.close()
+    
+    print("Success")
+`)
	got := SLP107{}.Check(d)
	if len(got) == 0 {
		t.Fatal("expected findings for Python except block cleanup")
	}
}

func TestSLP107_PythonNoFire(t *testing.T) {
	d := parseDiff(t, `diff --git a/handler.py b/handler.py
--- a/handler.py
+++ b/handler.py
@@ -1,5 +1,9 @@
+def handle():
+    try:
+        do_something()
+    except Exception:
+        conn.close()
+    
+    conn.close()
+`)
	got := SLP107{}.Check(d)
	if len(got) != 0 {
		t.Fatalf("expected 0 findings for Python with success path cleanup, got %d", len(got))
	}
}

func TestSLP107_FiresWhenDeletedSuccessCleanupIsReplacedForDifferentResource(t *testing.T) {
	d := parseDiff(t, `diff --git a/handler.go b/handler.go
--- a/handler.go
+++ b/handler.go
@@ -1,7 +1,11 @@
   if err != nil {
+      conn.Close()
       return err
   }
-  defer conn.Close()
+  defer other.Close()
`)
	got := SLP107{}.Check(d)
	if len(got) == 0 {
		t.Fatal("expected findings when conn cleanup only remains in error path")
	}
}

func TestSLP107_FiresOnLowercaseCleanupForDifferentResource(t *testing.T) {
	d := parseDiff(t, `diff --git a/handler.py b/handler.py
--- a/handler.py
+++ b/handler.py
@@ -1,1 +1,8 @@
+  try:
+      do_thing()
+  except Exception:
+      conn.close()
+  other.close()
`)
	got := SLP107{}.Check(d)
	if len(got) == 0 {
		t.Fatal("expected findings when lowercase cleanup targets a different resource")
	}
}

func TestSLP107_IgnoresDeletedErrorBlockMarkers(t *testing.T) {
	d := parseDiff(t, `diff --git a/handler.go b/handler.go
--- a/handler.go
+++ b/handler.go
@@ -1,4 +1,4 @@
-  if err != nil {
+  if failure != nil {
+      conn.Close()
+      return failure
+  }
`)
	got := SLP107{}.Check(d)
	if len(got) != 0 {
		t.Fatalf("expected 0 findings when only deleted line contains error marker, got %d", len(got))
	}
}

func TestSLP107_Description(t *testing.T) {
	r := SLP107{}
	if r.ID() != "SLP107" {
		t.Errorf("ID = %q", r.ID())
	}
	if r.Description() == "" {
		t.Errorf("Description is empty")
	}
	if r.DefaultSeverity() != SeverityBlock {
		t.Errorf("default severity should be block")
	}
}
