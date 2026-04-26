package rules

import "testing"

func TestSLP098_FiresOnRouteWithoutTest(t *testing.T) {
	d := parseDiff(t, `diff --git a/routes.go b/routes.go
--- a/routes.go
+++ b/routes.go
@@ -1,1 +1,3 @@
+  mux.HandleFunc("/api/items", itemsHandler)
`)
	got := SLP098{}.Check(d)
	if len(got) != 1 {
		t.Fatalf("expected 1 finding, got %d", len(got))
	}
}

func TestSLP098_FiresOnExpressRouteWithoutTest(t *testing.T) {
	d := parseDiff(t, `diff --git a/router.ts b/router.ts
--- a/router.ts
+++ b/router.ts
@@ -1,1 +1,3 @@
+  app.post("/api/users", usersHandler);
`)
	got := SLP098{}.Check(d)
	if len(got) != 1 {
		t.Fatalf("expected 1 finding, got %d", len(got))
	}
}

func TestSLP098_NoFireWhenTestAlsoModified(t *testing.T) {
	d := parseDiff(t, `diff --git a/routes.go b/routes.go
--- a/routes.go
+++ b/routes.go
@@ -1,1 +1,3 @@
+  mux.HandleFunc("/api/items", itemsHandler)
diff --git a/routes_test.go b/routes_test.go
--- a/routes_test.go
+++ b/routes_test.go
@@ -1,1 +1,3 @@
+  func TestItemsHandler(t *testing.T) {
`)
	got := SLP098{}.Check(d)
	if len(got) != 0 {
		t.Fatalf("expected 0 findings when test also modified, got %d", len(got))
	}
}

func TestSLP098_FiresWhenRelatedTestHasNoAddedLines(t *testing.T) {
	d := parseDiff(t, `diff --git a/routes.go b/routes.go
--- a/routes.go
+++ b/routes.go
@@ -1,1 +1,3 @@
+  mux.HandleFunc("/api/items", itemsHandler)
diff --git a/routes_test.go b/routes_test.go
--- a/routes_test.go
+++ b/routes_test.go
@@ -1,1 +1,0 @@
-  func TestOldItemsHandler(t *testing.T) {}
`)
	got := SLP098{}.Check(d)
	if len(got) != 1 {
		t.Fatalf("expected 1 finding when related test has no added lines, got %d", len(got))
	}
}

func TestSLP098_DoesNotTreatPrefixMatchAsRelatedTest(t *testing.T) {
	d := parseDiff(t, `diff --git a/user.go b/user.go
--- a/user.go
+++ b/user.go
@@ -1,1 +1,3 @@
+  router.Handle("/users", usersHandler)
diff --git a/user_profile_test.go b/user_profile_test.go
--- a/user_profile_test.go
+++ b/user_profile_test.go
@@ -1,1 +1,3 @@
+  func TestUserProfile(t *testing.T) {}
`)
	got := SLP098{}.Check(d)
	if len(got) != 1 {
		t.Fatalf("expected 1 finding when only prefixed test file changed, got %d", len(got))
	}
}

func TestSLP098_Description(t *testing.T) {
	r := SLP098{}
	if r.ID() != "SLP098" {
		t.Errorf("ID = %q", r.ID())
	}
	if r.Description() == "" {
		t.Errorf("Description is empty")
	}
	if r.DefaultSeverity() != SeverityWarn {
		t.Errorf("default severity should be warn")
	}
}
