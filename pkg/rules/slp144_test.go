package rules

import "testing"

func TestSLP144_FiresOnMixedErrorPatterns(t *testing.T) {
	d := parseDiff(t, `diff --git a/routes/user.js b/routes/user.js
--- a/routes/user.js
+++ b/routes/user.js
@@ -10,3 +10,8 @@
 router.get('/profile', (req, res) => {
+	if (!req.user) return res.fail(401);
 	res.json(req.user);
 });
+router.post('/users', (req, res) => {
+	const user = await User.create(req.body);
+	res.json(user);
+});
 `)
	got := SLP144{}.Check(d)
	if len(got) != 1 {
		t.Fatalf("expected 1 finding (mixed patterns), got %d: %+v", len(got), got)
	}
}

func TestSLP144_AllowsConsistentPatterns(t *testing.T) {
	d := parseDiff(t, `diff --git a/routes/auth.js b/routes/auth.js
--- a/routes/auth.js
+++ b/routes/auth.js
@@ -5,3 +5,7 @@
 router.post('/login', (req, res) => {
+	try {
+		const token = await authenticate(req.body);
+		res.json({token});
+	} catch (err) {
+		next(err);
+	}
 });
 `)
	got := SLP144{}.Check(d)
	if len(got) != 0 {
		t.Fatalf("expected 0 findings (consistent), got %d", len(got))
	}
}
