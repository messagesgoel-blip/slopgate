package rules

import (
	"strings"
	"testing"
)

func TestSLP087(t *testing.T) {
	tests := []struct {
		name     string
		diff     string
		expected int
	}{
		{
			name: "webhook handler without timeout",
			diff: strings.TrimSpace(`diff --git a/src/webhooks/stripe.js b/src/webhooks/stripe.js
index 123..456 100644
--- a/src/webhooks/stripe.js
+++ b/src/webhooks/stripe.js
@@ -1,5 +1,8 @@
-import { webhookHandler } from '../utils'
-webhookHandler.post('/stripe/webhook', (req, res) => {
-  const event = req.body
+import { webhookHandler } from '../utils'
+webhookHandler.post('/stripe/webhook', (req, res) => {
+  const event = req.body
   // Process Stripe webhook
   res.status(200).end()
 })
`),
			expected: 1,
		},
		{
			name: "webhook handler with timeout is ok",
			diff: strings.TrimSpace(`diff --git a/src/webhooks/stripe.js b/src/webhooks/stripe.js
index 123..456 100644
--- a/src/webhooks/stripe.js
+++ b/src/webhooks/stripe.js
@@ -1,5 +1,8 @@
-import { webhookHandler } from '../utils'
-webhookHandler.post('/stripe/webhook', (req, res) => {
-  const controller = new AbortController()
-  setTimeout(() => controller.abort(), 5000)
+import { webhookHandler } from '../utils'
+webhookHandler.post('/stripe/webhook', (req, res) => {
+  const controller = new AbortController()
+  setTimeout(() => controller.abort(), 5000)
   const event = req.body
   res.status(200).end()
 })
`),
			expected: 0,
		},
		{
			name: "GitHub webhook without timeout",
			diff: strings.TrimSpace(`diff --git a/src/webhooks/github.js b/src/webhooks/github.js
index 123..456 100644
--- a/src/webhooks/github.js
+++ b/src/webhooks/github.js
@@ -1,5 +1,8 @@
-const handleGitHubWebhook = (req, res) => {
-  const { action, repository } = req.body
+const handleGitHubWebhook = (req, res) => {
+  const { action, repository } = req.body
   // Process GitHub event
   res.status(200).json({ received: true })
 }
`),
			expected: 1,
		},
		{
			name: "non-webhook file should be skipped",
			diff: strings.TrimSpace(`diff --git a/src/utils.js b/src/utils.js
index 123..456 100644
--- a/src/utils.js
+++ b/src/utils.js
@@ -1,3 +1,5 @@
-const process = () => { return true }
+const process = () => { return true }
 export { process }
`),
			expected: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			d := parseDiff(t, tt.diff)
			r := SLP087{}
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
