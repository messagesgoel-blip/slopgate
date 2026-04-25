package rules

import (
	"strings"
	"testing"
)

func TestSLP088(t *testing.T) {
	tests := []struct {
		name     string
		diff     string
		expected int
	}{
		{
			name: "hardcoded API key",
			diff: strings.TrimSpace(`diff --git a/src/config.js b/src/config.js
index 123..456 100644
--- a/src/config.js
+++ b/src/config.js
@@ -1,5 +1,8 @@
-const config = {
-  apiKey: "sk-1234567890abcdef1234567890abcdef"
+const config = {
+  apiKey: "sk-1234567890abcdef1234567890abcdef"
 }
`),
			expected: 1,
		},
		{
			name: "hardcoded password",
			diff: strings.TrimSpace(`diff --git a/src/db.js b/src/db.js
index 123..456 100644
--- a/src/db.js
+++ b/src/db.js
@@ -1,5 +1,8 @@
-const dbConfig = {
-  password: "supersecret123"
+const dbConfig = {
+  password: "supersecret123"
 }
`),
			expected: 1,
		},
		{
			name: "process.env is ok",
			diff: strings.TrimSpace(`diff --git a/src/config.js b/src/config.js
index 123..456 100644
--- a/src/config.js
+++ b/src/config.js
@@ -1,5 +1,8 @@
-const config = {
-  apiKey: process.env.API_KEY
+const config = {
+  apiKey: process.env.API_KEY
 }
`),
			expected: 0,
		},
		{
			name: "AWS credentials",
			diff: strings.TrimSpace(`diff --git a/src/aws.js b/src/aws.js
index 123..456 100644
--- a/src/aws.js
+++ b/src/aws.js
@@ -1,5 +1,8 @@
-const AWS = require('aws-sdk')
-AWS.config.update({
-  accessKeyId: "AKIAIOSFODNN7EXAMPLE",
-  secretAccessKey: "wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY"
+const AWS = require('aws-sdk')
+AWS.config.update({
+  accessKeyId: "AKIAIOSFODNN7EXAMPLE",
+  secretAccessKey: "wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY"
 })
`),
			expected: 1,
		},
		{
			name: "private key in code",
			diff: strings.TrimSpace(`diff --git a/src/jwt.js b/src/jwt.js
index 123..456 100644
--- a/src/jwt.js
+++ b/src/jwt.js
@@ -1,5 +1,8 @@
-const privateKey = '-----BEGIN RSA PRIVATE KEY-----\nMIIEpAIBAAKCAQEA2Z3qX2BTLS4e0ek346h\n-----END RSA PRIVATE KEY-----'
+const privateKey = '-----BEGIN RSA PRIVATE KEY-----\nMIIEpAIBAAKCAQEA2Z3qX2BTLS4e0ek346h\n-----END RSA PRIVATE KEY-----'
`),
			expected: 1,
		},
		{
			name: "env var in .env file should be skipped",
			diff: strings.TrimSpace(`diff --git a/.env b/.env
index 123..456 100644
--- a/.env
+++ b/.env
@@ -1,2 +1,3 @@
-SECRET_KEY=mysecretkey123
-API_KEY=sk-abcdef123456
+SECRET_KEY=mysecretkey123
+API_KEY=sk-abcdef123456
`),
			expected: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			d := parseDiff(t, tt.diff)
			r := SLP088{}
			findings := r.Check(d)

			if len(findings) != tt.expected {
				t.Errorf("expected %d findings, got %d", tt.expected, len(findings))
				for _, f := range findings {
					t.Logf("  - %s:%d: %s", f.File, f.Line, f.Message)
				}
			}
		})
	}
}
