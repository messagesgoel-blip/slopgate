package rules

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestSLP007_GoSingleImportUnused(t *testing.T) {
	// import "fmt" added with no fmt. usage => finding
	d := parseDiff(t, `diff --git a/main.go b/main.go
--- a/main.go
+++ b/main.go
@@ -1,3 +1,6 @@
 package main

+import "fmt"
+
 func main() {}
`)
	got := SLP007{}.Check(d)
	if len(got) != 1 {
		t.Fatalf("expected 1 finding, got %d: %+v", len(got), got)
	}
	if got[0].File != "main.go" {
		t.Errorf("file: %q, want main.go", got[0].File)
	}
	if !strings.Contains(got[0].Message, "fmt") {
		t.Errorf("message should mention fmt: %q", got[0].Message)
	}
}

func TestSLP007_GoSingleImportUsed(t *testing.T) {
	// import "fmt" added AND fmt.Println(...) in same diff => NO finding
	d := parseDiff(t, `diff --git a/main.go b/main.go
--- a/main.go
+++ b/main.go
@@ -1,3 +1,6 @@
 package main

+import "fmt"
+func main() { fmt.Println("hi") }
`)
	got := SLP007{}.Check(d)
	if len(got) != 0 {
		t.Fatalf("expected 0 findings, got %d: %+v", len(got), got)
	}
}

func TestSLP007_GoGroupedImports(t *testing.T) {
	// import ( "fmt" "os" ) where only fmt is used => 1 finding for os
	d := parseDiff(t, `diff --git a/main.go b/main.go
--- a/main.go
+++ b/main.go
@@ -1,3 +1,9 @@
 package main

+import (
+	"fmt"
+	"os"
+)
+func main() { fmt.Println("hi") }
`)
	got := SLP007{}.Check(d)
	if len(got) != 1 {
		t.Fatalf("expected 1 finding (os), got %d: %+v", len(got), got)
	}
	if !strings.Contains(got[0].Message, "os") {
		t.Errorf("message should mention os: %q", got[0].Message)
	}
}

func TestSLP007_GoBlankImportNotFlagged(t *testing.T) {
	// import _ "image/png" => NO finding (side-effect import)
	d := parseDiff(t, `diff --git a/main.go b/main.go
--- a/main.go
+++ b/main.go
@@ -1,3 +1,5 @@
 package main

+import _ "image/png"
+func main() {}
`)
	got := SLP007{}.Check(d)
	if len(got) != 0 {
		t.Fatalf("expected 0 findings for blank import, got %d: %+v", len(got), got)
	}
}

func TestSLP007_GoDotImportNotFlagged(t *testing.T) {
	// import . "math" => NO finding (too ambiguous)
	d := parseDiff(t, `diff --git a/main.go b/main.go
--- a/main.go
+++ b/main.go
@@ -1,3 +1,5 @@
 package main

+import . "math"
+func main() {}
`)
	got := SLP007{}.Check(d)
	if len(got) != 0 {
		t.Fatalf("expected 0 findings for dot import, got %d: %+v", len(got), got)
	}
}

func TestSLP007_GoGroupedBlankAndDotNotFlagged(t *testing.T) {
	// Blank and dot imports inside grouped block are not flagged.
	d := parseDiff(t, `diff --git a/main.go b/main.go
--- a/main.go
+++ b/main.go
@@ -1,3 +1,8 @@
 package main

+import (
+	_ "image/png"
+	. "math"
+)
+func main() {}
`)
	got := SLP007{}.Check(d)
	if len(got) != 0 {
		t.Fatalf("expected 0 findings, got %d: %+v", len(got), got)
	}
}

func TestSLP007_GoAliasImport(t *testing.T) {
	// import f "fmt" => check for f. usage
	d := parseDiff(t, `diff --git a/main.go b/main.go
--- a/main.go
+++ b/main.go
@@ -1,3 +1,6 @@
 package main

+import f "fmt"
+func main() { f.Println("hi") }
`)
	got := SLP007{}.Check(d)
	if len(got) != 0 {
		t.Fatalf("expected 0 findings (alias used), got %d: %+v", len(got), got)
	}
}

func TestSLP007_GoAliasImportUnused(t *testing.T) {
	// import f "fmt" but no f. in diff => finding
	d := parseDiff(t, `diff --git a/main.go b/main.go
--- a/main.go
+++ b/main.go
@@ -1,3 +1,5 @@
 package main

+import f "fmt"
+func main() {}
`)
	got := SLP007{}.Check(d)
	if len(got) != 1 {
		t.Fatalf("expected 1 finding, got %d: %+v", len(got), got)
	}
}

func TestSLP007_GoLastPathSegment(t *testing.T) {
	// import "encoding/json" => check for json. usage
	d := parseDiff(t, `diff --git a/main.go b/main.go
--- a/main.go
+++ b/main.go
@@ -1,3 +1,6 @@
 package main

+import "encoding/json"
+func main() { json.Marshal(map[string]string{}) }
`)
	got := SLP007{}.Check(d)
	if len(got) != 0 {
		t.Fatalf("expected 0 findings (json used), got %d: %+v", len(got), got)
	}
}

func TestSLP007_GoLastPathSegmentUnused(t *testing.T) {
	// import "encoding/json" added but no json. usage => finding
	d := parseDiff(t, `diff --git a/main.go b/main.go
--- a/main.go
+++ b/main.go
@@ -1,3 +1,5 @@
 package main

+import "encoding/json"
+func main() {}
`)
	got := SLP007{}.Check(d)
	if len(got) != 1 {
		t.Fatalf("expected 1 finding, got %d: %+v", len(got), got)
	}
	if !strings.Contains(got[0].Message, "json") {
		t.Errorf("message should mention json: %q", got[0].Message)
	}
}

func TestSLP007_JSNamedImportTypeModifierIgnored(t *testing.T) {
	d := parseDiff(t, `diff --git a/app.tsx b/app.tsx
--- a/app.tsx
+++ b/app.tsx
@@ -1,1 +1,4 @@
 // app
+import { type Page, render } from 'pkg';
+export function App(page: Page) { return render(page.title); }
`)
	got := SLP007{}.Check(d)
	if len(got) != 0 {
		t.Fatalf("expected 0 findings for TS type modifier, got %d: %+v", len(got), got)
	}
}

func TestSLP007_UsesCurrentFileWhenRepoRootAvailable(t *testing.T) {
	root := t.TempDir()
	target := filepath.Join(root, "src", "page.tsx")
	if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	content := `import { User } from "lucide-react";

export function SettingsPage() {
  return <User className="icon" />;
}
`
	if err := os.WriteFile(target, []byte(content), 0o644); err != nil {
		t.Fatalf("write file: %v", err)
	}

	d := parseDiffWithRoot(t, root, `diff --git a/src/page.tsx b/src/page.tsx
--- a/src/page.tsx
+++ b/src/page.tsx
@@ -1,2 +1,3 @@
+import { User } from "lucide-react";
 export function SettingsPage() {
   return <User className="icon" />;
`)
	got := SLP007{}.Check(d)
	if len(got) != 0 {
		t.Fatalf("expected 0 findings when import is used in current file, got %d: %+v", len(got), got)
	}
}

func TestSLP007_JSNamedImportUnused(t *testing.T) {
	// import { useState } from 'react' with no useState usage => finding
	d := parseDiff(t, `diff --git a/app.tsx b/app.tsx
--- a/app.tsx
+++ b/app.tsx
@@ -1,1 +1,4 @@
 // app
+import { useState } from 'react';
+function App() { return null; }
+export default App;
`)
	got := SLP007{}.Check(d)
	if len(got) != 1 {
		t.Fatalf("expected 1 finding, got %d: %+v", len(got), got)
	}
	if !strings.Contains(got[0].Message, "useState") {
		t.Errorf("message should mention useState: %q", got[0].Message)
	}
}

func TestSLP007_JSNamedImportUsed(t *testing.T) {
	// import { useState } from 'react' AND useState(...) in same diff => NO finding
	d := parseDiff(t, `diff --git a/app.tsx b/app.tsx
--- a/app.tsx
+++ b/app.tsx
@@ -1,1 +1,4 @@
 // app
+import { useState } from 'react';
+function App() { const [x, setX] = useState(0); return x; }
+export default App;
`)
	got := SLP007{}.Check(d)
	if len(got) != 0 {
		t.Fatalf("expected 0 findings, got %d: %+v", len(got), got)
	}
}

func TestSLP007_JSDefaultImportUnused(t *testing.T) {
	// import lodash from 'lodash' with no lodash usage => finding
	d := parseDiff(t, `diff --git a/util.js b/util.js
--- a/util.js
+++ b/util.js
@@ -1,1 +1,4 @@
 // util
+import lodash from 'lodash';
+export function hello() { return 42; }
`)
	got := SLP007{}.Check(d)
	if len(got) != 1 {
		t.Fatalf("expected 1 finding, got %d: %+v", len(got), got)
	}
}

func TestSLP007_JSDefaultImportUsed(t *testing.T) {
	// import _ from 'lodash' where the identifier _ appears — tricky but
	// we match the default import ident. If "lodash" appears as word, no finding.
	d := parseDiff(t, `diff --git a/util.js b/util.js
--- a/util.js
+++ b/util.js
@@ -1,1 +1,4 @@
 // util
+import lodash from 'lodash';
+export function hello() { return lodash.map([]); }
`)
	got := SLP007{}.Check(d)
	if len(got) != 0 {
		t.Fatalf("expected 0 findings, got %d: %+v", len(got), got)
	}
}

func TestSLP007_JSStarImportUnused(t *testing.T) {
	// import * as React from 'react' with no React usage => finding
	d := parseDiff(t, `diff --git a/app.tsx b/app.tsx
--- a/app.tsx
+++ b/app.tsx
@@ -1,1 +1,4 @@
 // app
+import * as React from 'react';
+function App() { return null; }
+export default App;
`)
	got := SLP007{}.Check(d)
	if len(got) != 1 {
		t.Fatalf("expected 1 finding, got %d: %+v", len(got), got)
	}
}

func TestSLP007_JSStarImportUsed(t *testing.T) {
	// import * as React from 'react' AND React.createElement in diff => NO finding
	d := parseDiff(t, `diff --git a/app.tsx b/app.tsx
--- a/app.tsx
+++ b/app.tsx
@@ -1,1 +1,4 @@
 // app
+import * as React from 'react';
+function App() { return React.createElement('div'); }
+export default App;
`)
	got := SLP007{}.Check(d)
	if len(got) != 0 {
		t.Fatalf("expected 0 findings, got %d: %+v", len(got), got)
	}
}

func TestSLP007_JSNamedImportWithAlias(t *testing.T) {
	// import { useState as useSt } from 'react' — ident is useSt
	d := parseDiff(t, `diff --git a/app.tsx b/app.tsx
--- a/app.tsx
+++ b/app.tsx
@@ -1,1 +1,4 @@
 // app
+import { useState as useSt } from 'react';
+function App() { const [x] = useSt(0); return x; }
+export default App;
`)
	got := SLP007{}.Check(d)
	if len(got) != 0 {
		t.Fatalf("expected 0 findings (alias used), got %d: %+v", len(got), got)
	}
}

func TestSLP007_JSNamedImportWithAliasUnused(t *testing.T) {
	// import { useState as useSt } from 'react' — useSt not used => finding
	d := parseDiff(t, `diff --git a/app.tsx b/app.tsx
--- a/app.tsx
+++ b/app.tsx
@@ -1,1 +1,3 @@
 // app
+import { useState as useSt } from 'react';
+function App() { return null; }
`)
	got := SLP007{}.Check(d)
	if len(got) != 1 {
		t.Fatalf("expected 1 finding, got %d: %+v", len(got), got)
	}
}

func TestSLP007_PythonUnusedImport(t *testing.T) {
	// Python file — import os is unused => finding.
	d := parseDiff(t, `diff --git a/app.py b/app.py
--- a/app.py
+++ b/app.py
@@ -1,1 +1,3 @@
 # app
+import os
+def main(): pass
`)
	got := SLP007{}.Check(d)
	if len(got) != 1 {
		t.Fatalf("expected 1 finding for Python, got %d: %+v", len(got), got)
	}
	if !strings.Contains(got[0].Message, "os") {
		t.Errorf("message should mention os: %q", got[0].Message)
	}
}

func TestSLP007_PythonUnusedFromImport(t *testing.T) {
	// Python: from datetime import datetime — unused => finding.
	d := parseDiff(t, `diff --git a/app.py b/app.py
--- a/app.py
+++ b/app.py
@@ -1,1 +1,3 @@
 # app
+from datetime import datetime
+def main(): pass
`)
	got := SLP007{}.Check(d)
	if len(got) != 1 {
		t.Fatalf("expected 1 finding for Python from-import, got %d: %+v", len(got), got)
	}
}

func TestSLP007_PythonUsedImport_NoFinding(t *testing.T) {
	// Python: import os used as os.getenv => no finding.
	d := parseDiff(t, `diff --git a/app.py b/app.py
--- a/app.py
+++ b/app.py
@@ -1,1 +1,3 @@
 # app
+import os
+x = os.getenv("HOME")
`)
	got := SLP007{}.Check(d)
	if len(got) != 0 {
		t.Fatalf("expected 0 findings (os used), got %d: %+v", len(got), got)
	}
}

func TestSLP007_JavaUnusedImport(t *testing.T) {
	// Java: import java.util.ArrayList — unused => finding.
	d := parseDiff(t, `diff --git a/Foo.java b/Foo.java
--- a/Foo.java
+++ b/Foo.java
@@ -1,1 +1,3 @@
 // foo
+import java.util.ArrayList;
+public class Foo {}
`)
	got := SLP007{}.Check(d)
	if len(got) != 1 {
		t.Fatalf("expected 1 finding for Java, got %d: %+v", len(got), got)
	}
	if !strings.Contains(got[0].Message, "ArrayList") {
		t.Errorf("message should mention ArrayList: %q", got[0].Message)
	}
}

func TestSLP007_JavaUsedImport_NoFinding(t *testing.T) {
	// Java: import java.util.ArrayList used as new ArrayList<>() => no finding.
	d := parseDiff(t, `diff --git a/Foo.java b/Foo.java
--- a/Foo.java
+++ b/Foo.java
@@ -1,1 +1,3 @@
 // foo
+import java.util.ArrayList;
+public class Foo { ArrayList<String> list = new ArrayList<>(); }
`)
	got := SLP007{}.Check(d)
	if len(got) != 0 {
		t.Fatalf("expected 0 findings (ArrayList used), got %d: %+v", len(got), got)
	}
}

func TestSLP007_RustUnusedUse(t *testing.T) {
	// Rust: use std::collections::HashMap — unused => finding.
	d := parseDiff(t, `diff --git a/src/main.rs b/src/main.rs
--- a/src/main.rs
+++ b/src/main.rs
@@ -1,1 +1,3 @@
 // main
+use std::collections::HashMap;
+fn main() {}
`)
	got := SLP007{}.Check(d)
	if len(got) != 1 {
		t.Fatalf("expected 1 finding for Rust, got %d: %+v", len(got), got)
	}
	if !strings.Contains(got[0].Message, "HashMap") {
		t.Errorf("message should mention HashMap: %q", got[0].Message)
	}
}

func TestSLP007_RustUsedUse_NoFinding(t *testing.T) {
	// Rust: use std::collections::HashMap used in code => no finding.
	d := parseDiff(t, `diff --git a/src/main.rs b/src/main.rs
--- a/src/main.rs
+++ b/src/main.rs
@@ -1,1 +1,3 @@
 // main
+use std::collections::HashMap;
+fn main() { let m: HashMap<String, i32> = HashMap::new(); }
`)
	got := SLP007{}.Check(d)
	if len(got) != 0 {
		t.Fatalf("expected 0 findings (HashMap used), got %d: %+v", len(got), got)
	}
}

func TestSLP007_IgnoresPreExistingImports(t *testing.T) {
	// Pre-existing import "fmt" is context (not added), so no finding.
	d := parseDiff(t, `diff --git a/main.go b/main.go
--- a/main.go
+++ b/main.go
@@ -1,4 +1,5 @@
 package main
 import "fmt"
-// old
+// new
 func main() { fmt.Println("hi") }
`)
	got := SLP007{}.Check(d)
	if len(got) != 0 {
		t.Fatalf("expected 0 findings (import not added), got %d: %+v", len(got), got)
	}
}

func TestSLP007_JSOnlyFlagsUnusedIdentifiers(t *testing.T) {
	// import { useState, useEffect } from 'react'; only useState is used
	// => 1 finding for useEffect
	d := parseDiff(t, `diff --git a/app.tsx b/app.tsx
--- a/app.tsx
+++ b/app.tsx
@@ -1,1 +1,4 @@
 // app
+import { useState, useEffect } from 'react';
+function App() { const [x] = useState(0); return x; }
+export default App;
`)
	got := SLP007{}.Check(d)
	if len(got) != 1 {
		t.Fatalf("expected 1 finding (useEffect unused), got %d: %+v", len(got), got)
	}
	if !strings.Contains(got[0].Message, "useEffect") {
		t.Errorf("message should mention useEffect: %q", got[0].Message)
	}
}

func TestSLP007_Description(t *testing.T) {
	r := SLP007{}
	if r.ID() != "SLP007" {
		t.Errorf("ID = %q", r.ID())
	}
	if r.Description() == "" {
		t.Errorf("Description is empty")
	}
	if r.DefaultSeverity() != SeverityWarn {
		t.Errorf("default severity should be warn, got %v", r.DefaultSeverity())
	}
}

func TestSLP007_DeletedFileIgnored(t *testing.T) {
	// A deleted file should not produce findings.
	d := parseDiff(t, `diff --git a/main.go b/main.go
--- a/main.go
+++ /dev/null
@@ -1,4 +0,0 @@
-package main
-import "fmt"
-func main() {}
`)
	got := SLP007{}.Check(d)
	if len(got) != 0 {
		t.Fatalf("expected 0 findings for deleted file, got %d", len(got))
	}
}

func TestSLP007_JSMultipleNamedImportsAllUnused(t *testing.T) {
	// import { A, B, C } from 'mod'; none are used => 3 findings
	d := parseDiff(t, `diff --git a/app.js b/app.js
--- a/app.js
+++ b/app.js
@@ -1,1 +1,3 @@
 // app
+import { A, B, C } from 'mod';
+export function run() { return 1; }
`)
	got := SLP007{}.Check(d)
	if len(got) != 3 {
		t.Fatalf("expected 3 findings, got %d: %+v", len(got), got)
	}
}
