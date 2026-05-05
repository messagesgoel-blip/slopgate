package rules

import "testing"

func TestSLP143_FiresOnUncheckedEnvVar(t *testing.T) {
	d := parseDiff(t, `diff --git a/server.js b/server.js
--- a/server.js
+++ b/server.js
@@ -1,2 +1,4 @@
 import express from 'express';
+const apiKey = process.env.STRIPE_SECRET_KEY;
 const app = express();
 `)
	got := SLP143{}.Check(d)
	if len(got) != 1 {
		t.Fatalf("expected 1 finding, got %d", len(got))
	}
	if got[0].RuleID != "SLP143" {
		t.Errorf("rule ID = %q", got[0].RuleID)
	}
}

func TestSLP143_AllowsValidatedEnvVar(t *testing.T) {
	d := parseDiff(t, `diff --git a/config.js b/config.js
--- a/config.js
+++ b/config.js
@@ -1,2 +1,4 @@
 import express from 'express';
+const apiKey = process.env.STRIPE_SECRET_KEY || '';
 const app = express();
 `)
	got := SLP143{}.Check(d)
	if len(got) != 0 {
		t.Fatalf("expected 0 findings (has || default), got %d: %+v", len(got), got)
	}
}

func TestSLP143_IgnoresTestFiles(t *testing.T) {
	d := parseDiff(t, `diff --git a/api.test.js b/api.test.js
--- a/api.test.js
+++ b/api.test.js
@@ -1,2 +1,4 @@
 import { config } from './config';
+const key = process.env.API_KEY;
 `)
	got := SLP143{}.Check(d)
	if len(got) != 0 {
		t.Fatalf("expected 0 findings (test file), got %d", len(got))
	}
}
