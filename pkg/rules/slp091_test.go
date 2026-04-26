package rules

import (
	"testing"
)

func TestSLP091_FiresOnHardcodedJSDateInTest(t *testing.T) {
	d := parseDiff(t, `diff --git a/app.test.ts b/app.test.ts
--- a/app.test.ts
+++ b/app.test.ts
@@ -1,1 +1,3 @@
 describe("app", () => {
+  const fixture = { expires_at: new Date("2026-06-01").toISOString() };
 })
`)
	got := SLP091{}.Check(d)
	if len(got) != 1 {
		t.Fatalf("expected 1 finding, got %d: %+v", len(got), got)
	}
	if got[0].File != "app.test.ts" {
		t.Errorf("file = %q", got[0].File)
	}
}

func TestSLP091_FiresOnSQLExpiryInFixture(t *testing.T) {
	d := parseDiff(t, `diff --git a/data_test.go b/data_test.go
--- a/data_test.go
+++ b/data_test.go
@@ -1,1 +1,3 @@
 package data_test
+
+  expires_at: 2026-12-31T23:59:59Z,
`)
	got := SLP091{}.Check(d)
	if len(got) == 0 {
		t.Fatal("expected findings for hardcoded expiry")
	}
}

func TestSLP091_IgnoresNonTestFiles(t *testing.T) {
	d := parseDiff(t, `diff --git a/config.go b/config.go
--- a/config.go
+++ b/config.go
@@ -1,1 +1,3 @@
 package config
+
+const DefaultExpiry = "2026-12-31"
`)
	got := SLP091{}.Check(d)
	if len(got) != 0 {
		t.Fatalf("expected 0 findings for non-test file, got %d", len(got))
	}
}

func TestSLP091_IgnoresDateInComment(t *testing.T) {
	d := parseDiff(t, `diff --git a/app.test.ts b/app.test.ts
--- a/app.test.ts
+++ b/app.test.ts
@@ -1,1 +1,3 @@
 describe("app", () => {
+  // This fixture was created on 2026-04-26
 })
`)
	got := SLP091{}.Check(d)
	if len(got) != 0 {
		t.Fatalf("expected 0 findings for comment date, got %d", len(got))
	}
}

func TestSLP091_IgnoresOldHistoricalDates(t *testing.T) {
	d := parseDiff(t, `diff --git a/app.test.ts b/app.test.ts
--- a/app.test.ts
+++ b/app.test.ts
@@ -1,1 +1,3 @@
 describe("app", () => {
+  const createdAt = "2019-03-15T00:00:00Z";
 })
`)
	got := SLP091{}.Check(d)
	if len(got) != 0 {
		t.Fatalf("expected 0 findings for historical dates, got %d", len(got))
	}
}

func TestSLP091_IgnoresHistoricalDateWhenLineAlsoMentionsFutureYear(t *testing.T) {
	d := parseDiff(t, `diff --git a/app.test.ts b/app.test.ts
--- a/app.test.ts
+++ b/app.test.ts
@@ -1,1 +1,3 @@
 describe("app", () => {
+  const createdAt = "2019-03-15T00:00:00Z"; const note = "ticket-2026";
 })
`)
	got := SLP091{}.Check(d)
	if len(got) != 0 {
		t.Fatalf("expected 0 findings for historical date with unrelated 2026 token, got %d", len(got))
	}
}

func TestSLP091_IsTestFileDetectsCaseInsensitiveTestDirs(t *testing.T) {
	if !isTestFile("pkg/TestData/fixture.json") {
		t.Fatal("expected TestData directory to be treated as test content")
	}
}

func TestSLP091_FiresOnHardcodedTimestampInTest(t *testing.T) {
	d := parseDiff(t, `diff --git a/auth_test.go b/auth_test.go
--- a/auth_test.go
+++ b/auth_test.go
@@ -1,1 +1,3 @@
 package auth_test
+
+  "expires_at": 1735689600,
`)
	got := SLP091{}.Check(d)
	if len(got) == 0 {
		t.Fatal("expected findings for hardcoded timestamp")
	}
}

func TestSLP091_Description(t *testing.T) {
	r := SLP091{}
	if r.ID() != "SLP091" {
		t.Errorf("ID = %q", r.ID())
	}
	if r.Description() == "" {
		t.Errorf("Description is empty")
	}
	if r.DefaultSeverity() != SeverityBlock {
		t.Errorf("default severity should be block")
	}
}
