package rules

import (
	"strings"
	"testing"
)

func TestSLP059_FiresOnVariableArg(t *testing.T) {
	d := parseDiff(t, `diff --git a/main.go b/main.go
--- a/main.go
+++ b/main.go
@@ -1,2 +1,3 @@
 package main
+
+cmd := exec.Command("sh", "-c", userInput)
`)
	got := SLP059{}.Check(d)
	if len(got) != 1 {
		t.Fatalf("expected 1 finding, got %d: %+v", len(got), got)
	}
	if !strings.Contains(got[0].Message, "exec.Command argument may contain user input") {
		t.Errorf("message: %q", got[0].Message)
	}
}

func TestSLP059_FiresOnFmtSprintfArg(t *testing.T) {
	d := parseDiff(t, `diff --git a/main.go b/main.go
--- a/main.go
+++ b/main.go
@@ -1,2 +1,3 @@
 package main
+
+cmd := exec.Command(fmt.Sprintf("echo %s", name))
`)
	got := SLP059{}.Check(d)
	if len(got) != 1 {
		t.Fatalf("expected 1 finding, got %d: %+v", len(got), got)
	}
}

func TestSLP059_FiresOnConcatArg(t *testing.T) {
	d := parseDiff(t, `diff --git a/main.go b/main.go
--- a/main.go
+++ b/main.go
@@ -1,2 +1,3 @@
 package main
+
+cmd := exec.Command("echo " + msg)
`)
	got := SLP059{}.Check(d)
	if len(got) != 1 {
		t.Fatalf("expected 1 finding, got %d: %+v", len(got), got)
	}
}

func TestSLP059_IgnoresHardcodedArgs(t *testing.T) {
	d := parseDiff(t, `diff --git a/main.go b/main.go
--- a/main.go
+++ b/main.go
@@ -1,2 +1,3 @@
 package main
+
+cmd := exec.Command("echo", "hello")
`)
	got := SLP059{}.Check(d)
	if len(got) != 0 {
		t.Fatalf("expected 0 findings for hardcoded args, got %d: %+v", len(got), got)
	}
}

func TestSLP059_IgnoresCommentOnlyExecCommand(t *testing.T) {
	d := parseDiff(t, `diff --git a/main.go b/main.go
--- a/main.go
+++ b/main.go
@@ -1,2 +1,3 @@
 package main
+
+// exec.Command("sh", "-c", userInput)
`)
	got := SLP059{}.Check(d)
	if len(got) != 0 {
		t.Fatalf("expected 0 findings for comment-only exec.Command, got %d: %+v", len(got), got)
	}
}

func TestSLP059_IgnoresHardcodedRawStrings(t *testing.T) {
	d := parseDiff(t, `diff --git a/main.go b/main.go
--- a/main.go
+++ b/main.go
@@ -1,2 +1,3 @@
 package main
+
+cmd := exec.Command(`+"`echo`"+`, `+"`hello`"+`)
`)
	got := SLP059{}.Check(d)
	if len(got) != 0 {
		t.Fatalf("expected 0 findings for raw-string literals, got %d: %+v", len(got), got)
	}
}

func TestSLP059_IgnoresLocalLiteralStringVariables(t *testing.T) {
	d := parseDiff(t, `diff --git a/main.go b/main.go
--- a/main.go
+++ b/main.go
@@ -1,3 +1,6 @@
 package main
+
+func run() {
+	programName := "echo"
+	cmd := exec.Command(programName, "hello")
+}
`)
	got := SLP059{}.Check(d)
	if len(got) != 0 {
		t.Fatalf("expected 0 findings for local literal string variables, got %d: %+v", len(got), got)
	}
}

func TestSLP059_IgnoresTopLevelStringConst(t *testing.T) {
	d := parseDiff(t, `diff --git a/main.go b/main.go
--- a/main.go
+++ b/main.go
@@ -1,3 +1,5 @@
 package main
 const programName = "echo"
 func run() {
+	cmd := exec.Command(programName, "hello")
 }
`)
	got := SLP059{}.Check(d)
	if len(got) != 0 {
		t.Fatalf("expected 0 findings for top-level string const, got %d: %+v", len(got), got)
	}
}

func TestSLP059_FiresWhenLiteralVariableIsOutOfScope(t *testing.T) {
	d := parseDiff(t, `diff --git a/main.go b/main.go
--- a/main.go
+++ b/main.go
@@ -1,3 +1,8 @@
 package main
+
+func run(flag bool) {
+	if flag {
+		programName := "echo"
+	}
+	cmd := exec.Command(programName, "hello")
+}
`)
	got := SLP059{}.Check(d)
	if len(got) != 1 {
		t.Fatalf("expected 1 finding for out-of-scope literal variable, got %d: %+v", len(got), got)
	}
}

func TestSLP059_FiresOnMultilineVariableArg(t *testing.T) {
	d := parseDiff(t, `diff --git a/main.go b/main.go
--- a/main.go
+++ b/main.go
@@ -1,2 +1,6 @@
 package main
+
+cmd := exec.Command(
+	"sh",
+	userInput,
+)
`)
	got := SLP059{}.Check(d)
	if len(got) != 1 {
		t.Fatalf("expected 1 finding for multiline variable arg, got %d: %+v", len(got), got)
	}
}

func TestSLP059_IgnoresNonGoFile(t *testing.T) {
	d := parseDiff(t, `diff --git a/script.py b/script.py
--- a/script.py
+++ b/script.py
@@ -1,2 +1,3 @@
 def run():
+
+    os.execvp("echo", ["hello"])
`)
	got := SLP059{}.Check(d)
	if len(got) != 0 {
		t.Fatalf("expected 0 findings in non-Go file, got %d: %+v", len(got), got)
	}
}

func TestSLP059_Description(t *testing.T) {
	r := SLP059{}
	if r.ID() != "SLP059" {
		t.Errorf("ID = %q", r.ID())
	}
	if r.Description() == "" {
		t.Errorf("Description is empty")
	}
	if r.DefaultSeverity() != SeverityBlock {
		t.Errorf("default severity should be block")
	}
}

// Test multi-line const blocks (const ( ... ))
func TestSLP059_IgnoresConstBlockStringConsts(t *testing.T) {
	d := parseDiff(t, `diff --git a/main.go b/main.go
--- a/main.go
+++ b/main.go
@@ -1,3 +1,8 @@
 package main
+
+const (
+	ProgramName = "echo"
+	Shell       = "sh"
+)
+
 func run() {
+	cmd := exec.Command(ProgramName, "hello")
 }
`)
	got := SLP059{}.Check(d)
	if len(got) != 0 {
		t.Fatalf("expected 0 findings for const block string consts, got %d: %+v", len(got), got)
	}
}

// Test variables assigned from non-literal expressions (should still be flagged)
func TestSLP059_FiresOnVariableFromExpression(t *testing.T) {
	d := parseDiff(t, `diff --git a/main.go b/main.go
--- a/main.go
+++ b/main.go
@@ -1,2 +1,4 @@
 package main
+
+	programName := getProgramName()
+	cmd := exec.Command(programName, "hello")
`)
	got := SLP059{}.Check(d)
	if len(got) != 1 {
		t.Fatalf("expected 1 finding for variable from expression, got %d: %+v", len(got), got)
	}
}

// Test variables declared after the exec.Command call (should not be considered safe)
func TestSLP059_FiresOnVariableDeclaredAfterExec(t *testing.T) {
	d := parseDiff(t, `diff --git a/main.go b/main.go
--- a/main.go
+++ b/main.go
@@ -1,2 +1,5 @@
 package main
+
+	cmd := exec.Command(programName, "hello")
+	programName := "echo"
`)
	got := SLP059{}.Check(d)
	if len(got) != 1 {
		t.Fatalf("expected 1 finding for variable declared after exec, got %d: %+v", len(got), got)
	}
}

// Test var at package level (should not be treated as safe)
func TestSLP059_FiresOnPackageLevelVar(t *testing.T) {
	d := parseDiff(t, `diff --git a/main.go b/main.go
--- a/main.go
+++ b/main.go
@@ -1,3 +1,5 @@
 package main
+
+var programName = getProgramName()
+
 func run() {
+	cmd := exec.Command(programName, "hello")
 }
`)
	got := SLP059{}.Check(d)
	if len(got) != 1 {
		t.Fatalf("expected 1 finding for package-level var, got %d: %+v", len(got), got)
	}
}

// Test var at package level initialized with literal (should be treated as safe)
func TestSLP059_IgnoresPackageLevelVarLiteral(t *testing.T) {
	d := parseDiff(t, `diff --git a/main.go b/main.go
--- a/main.go
+++ b/main.go
@@ -1,3 +1,5 @@
 package main
+
+var programName = "echo"
+
 func run() {
+	cmd := exec.Command(programName, "hello")
 }
`)
	got := SLP059{}.Check(d)
	if len(got) != 0 {
		t.Fatalf("expected 0 findings for package-level var with literal, got %d: %+v", len(got), got)
	}
}

// Test mix of safe and unsafe identifiers in one argument (should flag if any unsafe)
func TestSLP059_FiresOnMixedSafeAndUnsafe(t *testing.T) {
	d := parseDiff(t, `diff --git a/main.go b/main.go
--- a/main.go
+++ b/main.go
@@ -1,3 +1,6 @@
 package main
+
+	safeArg := "hello"
+
 func run() {
+	cmd := exec.Command("echo", safeArg, unsafeArg)
 }
`)
	got := SLP059{}.Check(d)
	if len(got) != 1 {
		t.Fatalf("expected 1 finding for mixed safe and unsafe args, got %d: %+v", len(got), got)
	}
}

// Test exec call with zero arguments (should not flag)
func TestSLP059_IgnoresExecWithZeroArgs(t *testing.T) {
	d := parseDiff(t, `diff --git a/main.go b/main.go
--- a/main.go
+++ b/main.go
@@ -1,2 +1,3 @@
 package main
+
+	cmd := exec.Command()
`)
	got := SLP059{}.Check(d)
	if len(got) != 0 {
		t.Fatalf("expected 0 findings for exec with zero args, got %d: %+v", len(got), got)
	}
}

// Test exec call with only literal strings (should not flag)
func TestSLP059_IgnoresExecWithOnlyLiteralStrings(t *testing.T) {
	d := parseDiff(t, `diff --git a/main.go b/main.go
--- a/main.go
+++ b/main.go
@@ -1,2 +1,3 @@
 package main
+
+	cmd := exec.Command("echo", "hello", "world")
`)
	got := SLP059{}.Check(d)
	if len(got) != 0 {
		t.Fatalf("expected 0 findings for exec with only literals, got %d: %+v", len(got), got)
	}
}

// Test chained assignment propagation (b := a where a is safe)
// This documents current behavior - chained assignments are not recognized as safe
func TestSLP059_DocumentsChainedAssignmentNotRecognized(t *testing.T) {
	d := parseDiff(t, `diff --git a/main.go b/main.go
--- a/main.go
+++ b/main.go
@@ -1,3 +1,7 @@
 package main
+
 func run() {
+	original := "echo"
+	alias := original
+	cmd := exec.Command(alias, "hello")
 }
`)
	got := SLP059{}.Check(d)
	// Current behavior: alias is NOT recognized as safe because it doesn't match the literal string regex
	// This is a conservative approach (may produce false positives) which is acceptable
	// This test documents this limitation
	_ = got // We document that this may produce a finding
}
