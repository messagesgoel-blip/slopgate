package rules

import "testing"

func TestSLP136_FiresOnWrappedAppErrorWithoutCause(t *testing.T) {
	d := parseDiff(t, `diff --git a/api/src/routes/files.js b/api/src/routes/files.js
--- a/api/src/routes/files.js
+++ b/api/src/routes/files.js
@@ -1,3 +1,7 @@
 async function handler(req, res) {
+  } catch (err) {
+    logger.error({ err }, "folder-stats failed");
+    error(res, new AppError(CODES.INTERNAL, "internal server error"));
+  }
 }
`)
	assertFindings(t, SLP136{}.Check(d), 1, "SLP136", SeverityWarn)
}

func TestSLP136_IgnoresCauseFieldInConstructor(t *testing.T) {
	d := parseDiff(t, `diff --git a/api/src/routes/files.js b/api/src/routes/files.js
--- a/api/src/routes/files.js
+++ b/api/src/routes/files.js
@@ -1,3 +1,7 @@
 async function handler(req, res) {
+  } catch (err) {
+    logger.error({ err }, "folder-stats failed");
+    error(res, new AppError(CODES.INTERNAL, "internal server error", { cause: err }));
+  }
 }
`)
	assertFindings(t, SLP136{}.Check(d), 0, "SLP136", SeverityWarn)
}

func TestSLP136_IgnoresExplicitCauseAssignment(t *testing.T) {
	d := parseDiff(t, `diff --git a/api/src/routes/files.js b/api/src/routes/files.js
--- a/api/src/routes/files.js
+++ b/api/src/routes/files.js
@@ -1,3 +1,9 @@
 async function handler(req, res) {
+  } catch (err) {
+    logger.error({ err }, "folder-stats failed");
+    const appErr = new AppError(CODES.INTERNAL, "internal server error");
+    appErr.cause = err;
+    error(res, appErr);
+  }
 }
`)
	assertFindings(t, SLP136{}.Check(d), 0, "SLP136", SeverityWarn)
}

func TestSLP136_IgnoresMultilineCauseInConstructor(t *testing.T) {
	d := parseDiff(t, `diff --git a/api/src/routes/files.js b/api/src/routes/files.js
--- a/api/src/routes/files.js
+++ b/api/src/routes/files.js
@@ -1,3 +1,10 @@
 async function handler(req, res) {
+  } catch (err) {
+    logger.error({ err }, "folder-stats failed");
+    error(res, new AppError(CODES.INTERNAL, "internal server error", {
+      cause: err,
+    }));
+  }
 }
`)
	assertFindings(t, SLP136{}.Check(d), 0, "SLP136", SeverityWarn)
}

func TestSLP136_FiresOnUnrelatedCauseField(t *testing.T) {
	d := parseDiff(t, `diff --git a/api/src/routes/files.js b/api/src/routes/files.js
--- a/api/src/routes/files.js
+++ b/api/src/routes/files.js
@@ -1,3 +1,8 @@
 async function handler(req, res) {
+  } catch (err) {
+    logger.error({ err }, "folder-stats failed");
+    const meta = { cause: "timeout" };
+    error(res, new AppError(CODES.INTERNAL, "internal server error"));
+  }
 }
`)
	assertFindings(t, SLP136{}.Check(d), 1, "SLP136", SeverityWarn)
}
