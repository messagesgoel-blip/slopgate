package rules

import (
	"strings"
	"testing"
)

func TestSLP014_FiresOnGoFmtPrintln(t *testing.T) {
	d := parseDiff(t, `diff --git a/svc/handler.go b/svc/handler.go
--- a/svc/handler.go
+++ b/svc/handler.go
@@ -10,3 +10,4 @@
 func Handle() {
   x := 1
+  fmt.Println("got here", x)
 }
`)
	got := SLP014{}.Check(d)
	if len(got) != 1 {
		t.Fatalf("expected 1 finding, got %d: %+v", len(got), got)
	}
	if got[0].Line != 12 {
		t.Errorf("line = %d, want 12", got[0].Line)
	}
}

func TestSLP014_FiresOnConsoleLog(t *testing.T) {
	d := parseDiff(t, `diff --git a/app/route.ts b/app/route.ts
--- a/app/route.ts
+++ b/app/route.ts
@@ -1,2 +1,3 @@
 export async function GET() {
+  console.log("debug: entering route");
   return new Response("ok")
`)
	got := SLP014{}.Check(d)
	if len(got) != 1 {
		t.Fatalf("expected 1 finding, got %d: %+v", len(got), got)
	}
}

func TestSLP014_FiresOnPythonPrint(t *testing.T) {
	d := parseDiff(t, `diff --git a/svc/app.py b/svc/app.py
--- a/svc/app.py
+++ b/svc/app.py
@@ -1,2 +1,3 @@
 def handle(req):
+    print("debug:", req)
     return {"ok": True}
`)
	got := SLP014{}.Check(d)
	if len(got) != 1 {
		t.Fatalf("expected 1 finding, got %d: %+v", len(got), got)
	}
}

func TestSLP014_IgnoresTestFiles(t *testing.T) {
	// Debug prints in _test.go are fine — tests often log for visibility.
	d := parseDiff(t, `diff --git a/svc/handler_test.go b/svc/handler_test.go
--- a/svc/handler_test.go
+++ b/svc/handler_test.go
@@ -1,2 +1,3 @@
 func TestFoo(t *testing.T) {
+	fmt.Println("debug output")
 }
`)
	got := SLP014{}.Check(d)
	if len(got) != 0 {
		t.Fatalf("expected 0 findings in _test.go, got %d", len(got))
	}
}

func TestSLP014_IgnoresMainPackageEntrypoints(t *testing.T) {
	// cmd/*/main.go is allowed to print — that is a CLI's job.
	d := parseDiff(t, `diff --git a/cmd/foo/main.go b/cmd/foo/main.go
--- a/cmd/foo/main.go
+++ b/cmd/foo/main.go
@@ -1,2 +1,3 @@
 func main() {
+	fmt.Println("starting foo")
 }
`)
	got := SLP014{}.Check(d)
	if len(got) != 0 {
		t.Fatalf("expected 0 findings in cmd/foo/main.go, got %d", len(got))
	}
}

func TestSLP014_IgnoresCLISubcommandFiles(t *testing.T) {
	// Real false positive: cmd/<tool>/cmd_branch.go is a CLI subcommand
	// whose entire job is to print to stdout. The whole cmd/** tree is
	// exempt from the debug-print rule.
	d := parseDiff(t, `diff --git a/cmd/foo/cmd_branch.go b/cmd/foo/cmd_branch.go
--- a/cmd/foo/cmd_branch.go
+++ b/cmd/foo/cmd_branch.go
@@ -1,2 +1,4 @@
 package main
+func showBranch(b *Branch) {
+	fmt.Printf("Branch: %s\n", b.Name)
+}
`)
	got := SLP014{}.Check(d)
	if len(got) != 0 {
		t.Fatalf("expected 0 findings in cmd/foo/cmd_branch.go, got %d", len(got))
	}
}

func TestSLP014_FiresOnPkgFileEvenIfNameLooksCLI(t *testing.T) {
	// A file named cmd_foo.go under pkg/ is still production code and
	// should be checked. The exemption is by directory, not by filename.
	d := parseDiff(t, `diff --git a/pkg/cli/cmd_foo.go b/pkg/cli/cmd_foo.go
--- a/pkg/cli/cmd_foo.go
+++ b/pkg/cli/cmd_foo.go
@@ -1,2 +1,4 @@
 package cli
+func Do() {
+	fmt.Println("debug: called Do")
+}
`)
	got := SLP014{}.Check(d)
	if len(got) != 1 {
		t.Fatalf("expected 1 finding in pkg/cli/cmd_foo.go, got %d: %+v", len(got), got)
	}
}

func TestSLP014_IgnoresCommentsAndStrings(t *testing.T) {
	// Mentioning fmt.Println inside a comment or a doc string is not a real
	// debug call.
	d := parseDiff(t, `diff --git a/pkg/x/x.go b/pkg/x/x.go
--- a/pkg/x/x.go
+++ b/pkg/x/x.go
@@ -1,1 +1,3 @@
 package x
+// This helper replaces fmt.Println calls in the migration path.
+var help = "use fmt.Println for debugging"
`)
	got := SLP014{}.Check(d)
	if len(got) != 0 {
		t.Fatalf("expected 0 findings for comments/strings, got %d: %+v", len(got), got)
	}
}

func TestSLP014_FiresOnFmtPrintf(t *testing.T) {
	// fmt.Printf with a format string is also a debug print.
	d := parseDiff(t, `diff --git a/svc/handler.go b/svc/handler.go
--- a/svc/handler.go
+++ b/svc/handler.go
@@ -1,2 +1,3 @@
 func Do(x int) {
+	fmt.Printf("x=%d\n", x)
 }
`)
	got := SLP014{}.Check(d)
	if len(got) != 1 {
		t.Fatalf("expected 1 finding, got %d", len(got))
	}
}

func TestSLP014_IgnoresPrintInsideGoRawString(t *testing.T) {
	// Real false positive: a Go raw string literal containing a
	// Python `print(...)` call was flagged. Strings of any quote kind
	// should be stripped before pattern matching.
	d := parseDiff(t, `diff --git a/pkg/x/run.go b/pkg/x/run.go
--- a/pkg/x/run.go
+++ b/pkg/x/run.go
@@ -1,2 +1,4 @@
 package x
+func Check() {
+	cmd := exec.Command("python3", "-c", `+"`"+`import sys; print(sys.version_info[:]); sys.exit(0)`+"`"+`)
+}
`)
	got := SLP014{}.Check(d)
	if len(got) != 0 {
		t.Fatalf("expected 0 findings for print() inside raw string, got %d: %+v", len(got), got)
	}
}

func TestSLP014_IgnoresBlockComments(t *testing.T) {
	// C-style block comments: /* fmt.Println("x") */ should be stripped.
	d := parseDiff(t, `diff --git a/pkg/x/x.go b/pkg/x/x.go
--- a/pkg/x/x.go
+++ b/pkg/x/x.go
@@ -1,1 +1,4 @@
 package x
+/* fmt.Println("debug") */
+var y = 1 /* fmt.Printf("inline block comment") */ + 2
+/* console.log("also a block comment") */
`)
	got := SLP014{}.Check(d)
	if len(got) != 0 {
		t.Fatalf("expected 0 findings on block comments, got %d: %+v", len(got), got)
	}
}

func TestSLP014_IgnoresFullLineBlockComment(t *testing.T) {
	// A full line starting with /* should be treated like a comment.
	d := parseDiff(t, `diff --git a/pkg/x/x.go b/pkg/x/x.go
--- a/pkg/x/x.go
+++ b/pkg/x/x.go
@@ -1,1 +1,3 @@
 package x
+  /* this line has fmt.Println("debug") inside */
+  /* and print("also blocked") */
`)
	got := SLP014{}.Check(d)
	if len(got) != 0 {
		t.Fatalf("expected 0 findings on full-line block comments, got %d: %+v", len(got), got)
	}
}

func TestSLP014_IgnoresConsoleWarnAndError(t *testing.T) {
	// Real false positive: console.warn / console.error in catch blocks
	// are legitimate logging, not debug slop.
	d := parseDiff(t, `diff --git a/x.js b/x.js
--- a/x.js
+++ b/x.js
@@ -1,2 +1,5 @@
 async function load() {
+  try { await fetch(); }
+  catch (err) { console.warn('load failed:', err); }
+  catch (err) { console.error('load failed:', err); }
 }
`)
	got := SLP014{}.Check(d)
	if len(got) != 0 {
		t.Fatalf("expected 0 findings on console.warn/error, got %d: %+v", len(got), got)
	}
}

func TestSLP014_Description(t *testing.T) {
	r := SLP014{}
	if r.ID() != "SLP014" {
		t.Errorf("ID = %q", r.ID())
	}
	if !strings.Contains(strings.ToLower(r.Description()), "debug") {
		t.Errorf("description should mention debug: %q", r.Description())
	}
}
