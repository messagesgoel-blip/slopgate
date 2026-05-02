package rules

import (
	"strings"
	"testing"
)

// --- Go tests ---

func TestSLP003_GoEmptyBlock(t *testing.T) {
	// if err != nil { } — empty block, should fire.
	d := parseDiff(t, `diff --git a/a.go b/a.go
--- a/a.go
+++ b/a.go
@@ -1,1 +1,4 @@
 package a
+func Foo() error {
+	if err != nil {
+	}
+	return nil
+}
`)
	got := SLP003{}.Check(d)
	if len(got) != 1 {
		t.Fatalf("expected 1 finding, got %d: %+v", len(got), got)
	}
	if got[0].File != "a.go" {
		t.Errorf("file = %q, want a.go", got[0].File)
	}
	if !strings.Contains(got[0].Message, "empty") {
		t.Errorf("message should mention empty: %q", got[0].Message)
	}
}

func TestSLP003_GoReturnNil(t *testing.T) {
	// if err != nil { return nil } — swallows the error.
	d := parseDiff(t, `diff --git a/a.go b/a.go
--- a/a.go
+++ b/a.go
@@ -1,1 +1,4 @@
 package a
+func Foo() error {
+	if err != nil {
+		return nil
+	}
+	return nil
+}
`)
	got := SLP003{}.Check(d)
	if len(got) != 1 {
		t.Fatalf("expected 1 finding, got %d: %+v", len(got), got)
	}
	if !strings.Contains(got[0].Message, "swallow") {
		t.Errorf("message should mention swallow: %q", got[0].Message)
	}
}

func TestSLP003_GoLogPrintln_NoFinding(t *testing.T) {
	// if err != nil { log.Println(err); return nil } — logged, NOT a finding.
	d := parseDiff(t, `diff --git a/a.go b/a.go
--- a/a.go
+++ b/a.go
@@ -1,1 +1,4 @@
 package a
+func Foo() error {
+	if err != nil {
+		log.Println(err)
+		return nil
+	}
+	return nil
+}
`)
	got := SLP003{}.Check(d)
	if len(got) != 0 {
		t.Fatalf("expected 0 findings (logged), got %d: %+v", len(got), got)
	}
}

func TestSLP003_GoErrorWrap_NoFinding(t *testing.T) {
	// if err != nil { return fmt.Errorf("bad: %w", err) } — wrapped, NOT a finding.
	d := parseDiff(t, `diff --git a/a.go b/a.go
--- a/a.go
+++ b/a.go
@@ -1,1 +1,4 @@
 package a
+func Foo() error {
+	if err != nil {
+		return fmt.Errorf("bad: %w", err)
+	}
+	return nil
+}
`)
	got := SLP003{}.Check(d)
	if len(got) != 0 {
		t.Fatalf("expected 0 findings (error wrapped), got %d: %+v", len(got), got)
	}
}

func TestSLP003_GoSlog_NoFinding(t *testing.T) {
	// slog.Error counts as logging.
	d := parseDiff(t, `diff --git a/a.go b/a.go
--- a/a.go
+++ b/a.go
@@ -1,1 +1,4 @@
 package a
+func Foo() error {
+	if err != nil {
+		slog.Error("failed", "err", err)
+		return nil
+	}
+	return nil
+}
`)
	got := SLP003{}.Check(d)
	if len(got) != 0 {
		t.Fatalf("expected 0 findings (slog), got %d: %+v", len(got), got)
	}
}

func TestSLP003_GoPanic_NoFinding(t *testing.T) {
	// re-panic counts as handling.
	d := parseDiff(t, `diff --git a/a.go b/a.go
--- a/a.go
+++ b/a.go
@@ -1,1 +1,4 @@
 package a
+func Foo() error {
+	if err != nil {
+		panic(err)
+	}
+	return nil
+}
`)
	got := SLP003{}.Check(d)
	if len(got) != 0 {
		t.Fatalf("expected 0 findings (panic), got %d: %+v", len(got), got)
	}
}

func TestSLP003_GoMixedBodyNotAllAdded_NoFinding(t *testing.T) {
	// If the block body is not entirely added lines, we can't reason about it.
	d := parseDiff(t, `diff --git a/a.go b/a.go
--- a/a.go
+++ b/a.go
@@ -1,3 +1,5 @@
 package a
 func Foo() error {
 	if err != nil {
-		return err
+		return nil
 	}
`)
	got := SLP003{}.Check(d)
	if len(got) != 0 {
		t.Fatalf("expected 0 findings (mixed body), got %d: %+v", len(got), got)
	}
}

func TestSLP003_GoSingleLineReturnNil(t *testing.T) {
	// if err != nil { return nil } — single-line swallow.
	d := parseDiff(t, `diff --git a/a.go b/a.go
--- a/a.go
+++ b/a.go
@@ -1,1 +1,3 @@
 package a
+if err != nil { return nil }
`)
	got := SLP003{}.Check(d)
	if len(got) != 1 {
		t.Fatalf("expected 1 finding for single-line return nil, got %d: %+v", len(got), got)
	}
	if !strings.Contains(got[0].Message, "swallow") {
		t.Errorf("message should mention swallow: %q", got[0].Message)
	}
}

func TestSLP003_GoSingleLineEmptyBlock(t *testing.T) {
	// if err != nil { } — single-line empty block.
	d := parseDiff(t, `diff --git a/a.go b/a.go
--- a/a.go
+++ b/a.go
@@ -1,1 +1,3 @@
 package a
+if err != nil { }
`)
	got := SLP003{}.Check(d)
	if len(got) != 1 {
		t.Fatalf("expected 1 finding for single-line empty block, got %d: %+v", len(got), got)
	}
	if !strings.Contains(got[0].Message, "empty") {
		t.Errorf("message should mention empty: %q", got[0].Message)
	}
}

func TestSLP003_GoSingleLineWithLog_NoFinding(t *testing.T) {
	// if err != nil { log.Println(err); return nil } — logged, NOT a finding.
	d := parseDiff(t, `diff --git a/a.go b/a.go
--- a/a.go
+++ b/a.go
@@ -1,1 +1,3 @@
 package a
+if err != nil { log.Println(err); return nil }
`)
	got := SLP003{}.Check(d)
	if len(got) != 0 {
		t.Fatalf("expected 0 findings (logged), got %d: %+v", len(got), got)
	}
}

func TestSLP003_GoSingleLineReturnErr_NoFinding(t *testing.T) {
	// if err != nil { return err } — propagates the error, NOT a finding.
	d := parseDiff(t, `diff --git a/a.go b/a.go
--- a/a.go
+++ b/a.go
@@ -1,1 +1,3 @@
 package a
+if err != nil { return err }
`)
	got := SLP003{}.Check(d)
	if len(got) != 0 {
		t.Fatalf("expected 0 findings (return err), got %d: %+v", len(got), got)
	}
}

func TestSLP003_GoSingleLineReturnFalseNil(t *testing.T) {
	// if err != nil { return false, nil } — silent multi-return.
	d := parseDiff(t, `diff --git a/a.go b/a.go
--- a/a.go
+++ b/a.go
@@ -1,1 +1,3 @@
 package a
+if err != nil { return false, nil }
`)
	got := SLP003{}.Check(d)
	if len(got) != 1 {
		t.Fatalf("expected 1 finding for single-line return false, nil, got %d: %+v", len(got), got)
	}
}

func TestSLP003_GoIgnoreComment_NoFinding(t *testing.T) {
	// if err != nil { /* ignore */ } — intentional, NOT a finding.
	d := parseDiff(t, `diff --git a/a.go b/a.go
--- a/a.go
+++ b/a.go
@@ -1,1 +1,4 @@
 package a
+func Foo() error {
+	if err != nil {
+		// ignore
+	}
+	return nil
+}
`)
	got := SLP003{}.Check(d)
	if len(got) != 0 {
		t.Fatalf("expected 0 findings (ignored), got %d: %+v", len(got), got)
	}
}

// --- JS/TS tests ---

func TestSLP003_JSEmptyCatch(t *testing.T) {
	// catch (e) {} — empty block, should fire.
	d := parseDiff(t, `diff --git a/a.js b/a.js
--- a/a.js
+++ b/a.js
@@ -1,1 +1,4 @@
 // a
+function foo() {
+  try { bar(); } catch (e) {}
+}
`)
	got := SLP003{}.Check(d)
	if len(got) != 1 {
		t.Fatalf("expected 1 finding, got %d: %+v", len(got), got)
	}
	if got[0].File != "a.js" {
		t.Errorf("file = %q, want a.js", got[0].File)
	}
}

func TestSLP003_JSCatchReturn_NoFinding(t *testing.T) {
	// catch with logger.error — NOT a finding.
	d := parseDiff(t, `diff --git a/a.js b/a.js
--- a/a.js
+++ b/a.js
@@ -1,1 +1,4 @@
 // a
+function foo() {
+  try { bar(); } catch (e) { logger.error(e); }
+}
`)
	got := SLP003{}.Check(d)
	if len(got) != 0 {
		t.Fatalf("expected 0 findings (logger.error), got %d: %+v", len(got), got)
	}
}

func TestSLP003_JSCatchReturnSemicolon(t *testing.T) {
	// catch (e) { return; } — bail-only, should fire.
	d := parseDiff(t, `diff --git a/a.ts b/a.ts
--- a/a.ts
+++ b/a.ts
@@ -1,1 +1,4 @@
 // a
+function foo() {
+  try { bar(); } catch (e) { return; }
+}
`)
	got := SLP003{}.Check(d)
	if len(got) != 1 {
		t.Fatalf("expected 1 finding (return; bail), got %d: %+v", len(got), got)
	}
}

func TestSLP003_JSIgnoreComment_NoFinding(t *testing.T) {
	// catch (e) { /* ignore */ } — intentional, NOT a finding.
	d := parseDiff(t, `diff --git a/a.js b/a.js
--- a/a.js
+++ b/a.js
@@ -1,1 +1,4 @@
+try {
+  foo();
+} catch (e) {
+  // intentional
+}
`)
	got := SLP003{}.Check(d)
	if len(got) != 0 {
		t.Fatalf("expected 0 findings (ignored), got %d: %+v", len(got), got)
	}
}

// --- Python tests ---

func TestSLP003_PythonExceptPass(t *testing.T) {
	// except: pass — should fire.
	d := parseDiff(t, `diff --git a/a.py b/a.py
--- a/a.py
+++ b/a.py
@@ -1,1 +1,5 @@
 # a
+def foo():
+    try:
+        bar()
+    except:
+        pass
`)
	got := SLP003{}.Check(d)
	if len(got) != 1 {
		t.Fatalf("expected 1 finding, got %d: %+v", len(got), got)
	}
	if !strings.Contains(got[0].Message, "bare except") {
		t.Errorf("message should mention bare except: %q", got[0].Message)
	}
}

func TestSLP003_PythonExceptExceptionPass(t *testing.T) {
	// except Exception: pass — should fire.
	d := parseDiff(t, `diff --git a/a.py b/a.py
--- a/a.py
+++ b/a.py
@@ -1,1 +1,5 @@
 # a
+def foo():
+    try:
+        bar()
+    except Exception:
+        pass
`)
	got := SLP003{}.Check(d)
	if len(got) != 1 {
		t.Fatalf("expected 1 finding, got %d: %+v", len(got), got)
	}
}

func TestSLP003_PythonExceptReturnNone(t *testing.T) {
	// except: return None — should fire.
	d := parseDiff(t, `diff --git a/a.py b/a.py
--- a/a.py
+++ b/a.py
@@ -1,1 +1,5 @@
 # a
+def foo():
+    try:
+        bar()
+    except:
+        return None
`)
	got := SLP003{}.Check(d)
	if len(got) != 1 {
		t.Fatalf("expected 1 finding, got %d: %+v", len(got), got)
	}
}

func TestSLP003_PythonLogger_NoFinding(t *testing.T) {
	// except Exception as e: logger.error(e) — NOT a finding.
	d := parseDiff(t, `diff --git a/a.py b/a.py
--- a/a.py
+++ b/a.py
@@ -1,1 +1,5 @@
 # a
+def foo():
+    try:
+        bar()
+    except Exception as e:
+        logger.error(e)
`)
	got := SLP003{}.Check(d)
	if len(got) != 0 {
		t.Fatalf("expected 0 findings (logger.error), got %d: %+v", len(got), got)
	}
}

func TestSLP003_PythonRaise_NoFinding(t *testing.T) {
	// except Exception as e: raise — NOT a finding.
	d := parseDiff(t, `diff --git a/a.py b/a.py
--- a/a.py
+++ b/a.py
@@ -1,1 +1,5 @@
 # a
+def foo():
+    try:
+        bar()
+    except Exception as e:
+        raise
`)
	got := SLP003{}.Check(d)
	if len(got) != 0 {
		t.Fatalf("expected 0 findings (raise), got %d: %+v", len(got), got)
	}
}

// --- Interface conformance ---

func TestSLP003_Description(t *testing.T) {
	r := SLP003{}
	if r.ID() != "SLP003" {
		t.Errorf("ID = %q, want SLP003", r.ID())
	}
	if r.Description() == "" {
		t.Errorf("Description is empty")
	}
	if r.DefaultSeverity() != SeverityWarn {
		t.Errorf("default severity = %v, want warn", r.DefaultSeverity())
	}
}

// --- Java tests ---

func TestSLP003_JavaEmptyCatch(t *testing.T) {
	// } catch (Exception e) { } — empty block, should fire.
	d := parseDiff(t, `diff --git a/a.java b/a.java
--- a/a.java
+++ b/a.java
@@ -1,1 +1,4 @@
 // a
+public void foo() {
+    try { bar(); } catch (Exception e) {}
+}
`)
	got := SLP003{}.Check(d)
	if len(got) != 1 {
		t.Fatalf("expected 1 finding, got %d: %+v", len(got), got)
	}
	if got[0].File != "a.java" {
		t.Errorf("file = %q, want a.java", got[0].File)
	}
}

func TestSLP003_JavaCatchReturnNull(t *testing.T) {
	// } catch (Exception e) { return null; } — bail-only, should fire.
	d := parseDiff(t, `diff --git a/Foo.java b/Foo.java
--- a/Foo.java
+++ b/Foo.java
@@ -1,1 +1,5 @@
 // a
+public Object foo() {
+    try { bar(); }
+    catch (Exception e) { return null; }
+}
`)
	got := SLP003{}.Check(d)
	if len(got) != 1 {
		t.Fatalf("expected 1 finding, got %d: %+v", len(got), got)
	}
	if !strings.Contains(got[0].Message, "swallow") {
		t.Errorf("message should mention swallow: %q", got[0].Message)
	}
}

func TestSLP003_JavaCatchLogger_NoFinding(t *testing.T) {
	// } catch (Exception e) { logger.error(e.getMessage()); } — logged, NOT a finding.
	d := parseDiff(t, `diff --git a/Foo.java b/Foo.java
--- a/Foo.java
+++ b/Foo.java
@@ -1,1 +1,5 @@
 // a
+public void foo() {
+    try { bar(); }
+    catch (Exception e) { logger.error(e.getMessage()); }
+}
`)
	got := SLP003{}.Check(d)
	if len(got) != 0 {
		t.Fatalf("expected 0 findings (logger.error), got %d: %+v", len(got), got)
	}
}

func TestSLP003_JavaCatchThrow_NoFinding(t *testing.T) {
	// } catch (Exception e) { throw new RuntimeException(e); } — re-thrown, NOT a finding.
	d := parseDiff(t, `diff --git a/Foo.java b/Foo.java
--- a/Foo.java
+++ b/Foo.java
@@ -1,1 +1,5 @@
 // a
+public void foo() {
+    try { bar(); }
+    catch (Exception e) { throw new RuntimeException(e); }
+}
`)
	got := SLP003{}.Check(d)
	if len(got) != 0 {
		t.Fatalf("expected 0 findings (throw), got %d: %+v", len(got), got)
	}
}

// --- Rust tests ---

func TestSLP003_RustMatchErrEmpty(t *testing.T) {
	// Err(e) => {} — empty match arm, should fire.
	d := parseDiff(t, `diff --git a/a.rs b/a.rs
--- a/a.rs
+++ b/a.rs
@@ -1,1 +1,4 @@
 // a
+match result {
+    Err(e) => {}
+}
`)
	got := SLP003{}.Check(d)
	if len(got) != 1 {
		t.Fatalf("expected 1 finding, got %d: %+v", len(got), got)
	}
	if got[0].File != "a.rs" {
		t.Errorf("file = %q, want a.rs", got[0].File)
	}
}

func TestSLP003_RustIfLetErrReturnNone(t *testing.T) {
	// if let Err(e) = result { return None } — bail-only, should fire.
	d := parseDiff(t, `diff --git a/a.rs b/a.rs
--- a/a.rs
+++ b/a.rs
@@ -1,1 +1,4 @@
 // a
+if let Err(e) = result {
+    return None;
+}
`)
	got := SLP003{}.Check(d)
	if len(got) != 1 {
		t.Fatalf("expected 1 finding, got %d: %+v", len(got), got)
	}
	if !strings.Contains(got[0].Message, "swallow") {
		t.Errorf("message should mention swallow: %q", got[0].Message)
	}
}

func TestSLP003_RustMatchErrLog_NoFinding(t *testing.T) {
	// Err(e) => { error!("fail: {e}"); return None } — logged, NOT a finding.
	d := parseDiff(t, `diff --git a/a.rs b/a.rs
--- a/a.rs
+++ b/a.rs
@@ -1,1 +1,4 @@
 // a
+match result {
+    Err(e) => { error!("fail: {e}"); return None; }
+}
`)
	got := SLP003{}.Check(d)
	if len(got) != 0 {
		t.Fatalf("expected 0 findings (error!), got %d: %+v", len(got), got)
	}
}

func TestSLP003_RustMatchErrReturnErr_NoFinding(t *testing.T) {
	// Err(e) => Err(e) — propagated, NOT a finding.
	d := parseDiff(t, `diff --git a/a.rs b/a.rs
--- a/a.rs
+++ b/a.rs
@@ -1,1 +1,4 @@
 // a
+match result {
+    Err(e) => { return Err(e); }
+}
`)
	got := SLP003{}.Check(d)
	if len(got) != 0 {
		t.Fatalf("expected 0 findings (Err), got %d: %+v", len(got), got)
	}
}
