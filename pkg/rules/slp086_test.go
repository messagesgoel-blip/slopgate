package rules

import (
	"strings"
	"testing"
)

func TestSLP086(t *testing.T) {
	tests := []struct {
		name     string
		diff     string
		expected int
	}{
		{
			name: "sensitive route without auth",
			diff: strings.TrimSpace(`diff --git a/src/routes/admin.js b/src/routes/admin.js
index 123..456 100644
--- a/src/routes/admin.js
+++ b/src/routes/admin.js
@@ -1,5 +1,8 @@
 const router = require('express').Router()

-router.post('/admin/delete-user', async (req, res) => {
-  const { userId } = req.body
-  await User.destroy({ where: { id: userId } })
+router.post('/admin/delete-user', async (req, res) => {
+  const { userId } = req.body
+  await User.destroy({ where: { id: userId } })
   res.json({ success: true })
 })
`),
			expected: 1,
		},
		{
			name: "sensitive route with auth check",
			diff: strings.TrimSpace(`diff --git a/src/routes/admin.js b/src/routes/admin.js
index 123..456 100644
--- a/src/routes/admin.js
+++ b/src/routes/admin.js
@@ -1,7 +1,10 @@
 const router = require('express').Router()

-router.post('/admin/delete-user', async (req, res) => {
-  if (!req.user.isAdmin) return res.status(403).json({ error: 'Forbidden' })
-  const { userId } = req.body
-  await User.destroy({ where: { id: userId } })
+router.post('/admin/delete-user', async (req, res) => {
+  if (!req.user.isAdmin) return res.status(403).json({ error: 'Forbidden' })
+  const { userId } = req.body
+  await User.destroy({ where: { id: userId } })
   res.json({ success: true })
 })
`),
			expected: 0,
		},
		{
			name: "normal GET route without auth is ok",
			diff: strings.TrimSpace(`diff --git a/src/routes/products.js b/src/routes/products.js
index 123..456 100644
--- a/src/routes/products.js
+++ b/src/routes/products.js
@@ -1,5 +1,8 @@
 const router = require('express').Router()

-router.get('/products', async (req, res) => {
-  const products = await Product.findAll()
+router.get('/products', async (req, res) => {
+  const products = await Product.findAll()
   res.json(products)
 })
`),
			expected: 0,
		},
		{
			name: "sensitive password route without auth",
			diff: strings.TrimSpace(`diff --git a/src/routes/users.js b/src/routes/users.js
index 123..456 100644
--- a/src/routes/users.js
+++ b/src/routes/users.js
@@ -1,5 +1,8 @@
 const router = require('express').Router()

-router.put('/users/password', async (req, res) => {
-  const { newPassword } = req.body
-  req.user.password = hash(newPassword)
+router.put('/users/password', async (req, res) => {
+  const { newPassword } = req.body
+  req.user.password = hash(newPassword)
   await req.user.save()
   res.json({ success: true })
 })
`),
			expected: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			d := parseDiff(t, tt.diff)
			r := SLP086{}
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
