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
			name: "hardcoded API key in config",
			diff: strings.TrimSpace(`diff --git a/config.json b/config.json
index 123..456 100644
--- a/config.json
+++ b/config.json
@@ -1,5 +1,8 @@
 {
-  "apiKey": "sk-1234567890abcdef1234567890abcdef"
+  "apiKey": "sk-1234567890abcdef1234567890abcdef"
 }
`),
			expected: 1,
		},
		{
			name: "hardcoded password in YAML",
			diff: strings.TrimSpace(`diff --git a/config.yml b/config.yml
index 123..456 100644
--- a/config.yml
+++ b/config.yml
@@ -1,5 +1,8 @@
 database:
-  password: "supersecret123"
+  password: "supersecret123"
   host: localhost
`),
			expected: 1,
		},
		{
			name: "process.env is ok in config",
			diff: strings.TrimSpace(`diff --git a/config.yaml b/config.yaml
index 123..456 100644
--- a/config.yaml
+++ b/config.yaml
@@ -1,5 +1,8 @@
 database:
-  password: process.env.DB_PASSWORD
+  password: process.env.DB_PASSWORD
   host: localhost
`),
			expected: 0,
		},
		{
			name: "AWS credentials in TOML",
			diff: strings.TrimSpace(`diff --git a/settings.toml b/settings.toml
index 123..456 100644
--- a/settings.toml
+++ b/settings.toml
@@ -1,5 +1,8 @@
 [aws]
-  access_key = "AKIAIOSFODNN7EXAMPLE"
-  secret_key = "wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY"
+  access_key = "AKIAIOSFODNN7EXAMPLE"
+  secret_key = "wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY"
`),
			expected: 1,
		},
		{
			name: "private key in settings file",
			diff: strings.TrimSpace(`diff --git a/config.json b/config.json
index 123..456 100644
--- a/config.json
+++ b/config.json
@@ -1,5 +1,8 @@
 {
-  "privateKey": "-----BEGIN RSA PRIVATE KEY-----\\nMIIEpAIBAAKCAQEA2Z3qX2BTLS4e0ek346h\\n-----END RSA PRIVATE KEY-----"
+  "privateKey": "-----BEGIN RSA PRIVATE KEY-----\\nMIIEpAIBAAKCAQEA2Z3qX2BTLS4e0ek346h\\n-----END RSA PRIVATE KEY-----"
 }
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
		{
			name: "source .js file should be skipped",
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
