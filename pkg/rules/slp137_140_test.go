package rules

import (
	"os"
	"path/filepath"
	"testing"
)

func TestSLP137_FiresOnMixedBotPriorityAcrossRepo(t *testing.T) {
	root := t.TempDir()
	oldFile := filepath.Join(root, "api/src/workers/botFetch.js")
	if err := os.MkdirAll(filepath.Dir(oldFile), 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(oldFile, []byte(`async function run(queue, payload) {
  await queue.add('bot.fetch', buildJobEnvelope('bot.fetch', payload));
}
`), 0o644); err != nil {
		t.Fatalf("write old file: %v", err)
	}

	d := parseDiffWithRoot(t, root, `diff --git a/api/src/routes/slackBot.js b/api/src/routes/slackBot.js
--- a/api/src/routes/slackBot.js
+++ b/api/src/routes/slackBot.js
@@ -1,3 +1,6 @@
 async function enqueue(queue, payload) {
+  await queue.add('bot.search', buildJobEnvelope('bot.search', payload), {
+    priority: 1,
+  });
 }
`)
	assertFindings(t, SLP137{}.Check(d), 1, "SLP137", SeverityWarn)
}

func TestSLP137_IgnoresWhenSiblingCallsAlsoUsePriority(t *testing.T) {
	root := t.TempDir()
	oldFile := filepath.Join(root, "api/src/workers/botFetch.js")
	if err := os.MkdirAll(filepath.Dir(oldFile), 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(oldFile, []byte(`async function run(queue, payload) {
  await queue.add('bot.fetch', buildJobEnvelope('bot.fetch', payload), { priority: 1 });
}
`), 0o644); err != nil {
		t.Fatalf("write old file: %v", err)
	}

	d := parseDiffWithRoot(t, root, `diff --git a/api/src/routes/slackBot.js b/api/src/routes/slackBot.js
--- a/api/src/routes/slackBot.js
+++ b/api/src/routes/slackBot.js
@@ -1,3 +1,6 @@
 async function enqueue(queue, payload) {
+  await queue.add('bot.search', buildJobEnvelope('bot.search', payload), {
+    priority: 1,
+  });
 }
`)
	assertFindings(t, SLP137{}.Check(d), 0, "SLP137", SeverityWarn)
}

func TestSLP138_FiresWhenCreateFolderDropsCreds(t *testing.T) {
	d := parseDiff(t, `diff --git a/api/src/services/provider.js b/api/src/services/provider.js
--- a/api/src/services/provider.js
+++ b/api/src/services/provider.js
@@ -1,3 +1,9 @@
 async function create(remote, fileProvider) {
+  if (remote.creds) logger.info("creds present");
+  return fileProvider.createFolder({
+    token: remote.accessToken,
+    remoteId: remote.remoteId,
+  });
 }
`)
	assertFindings(t, SLP138{}.Check(d), 1, "SLP138", SeverityWarn)
}

func TestSLP138_IgnoresWhenCredsAreForwarded(t *testing.T) {
	d := parseDiff(t, `diff --git a/api/src/services/provider.js b/api/src/services/provider.js
--- a/api/src/services/provider.js
+++ b/api/src/services/provider.js
@@ -1,3 +1,10 @@
 async function create(remote, fileProvider) {
+  if (remote.creds) logger.info("creds present");
+  return fileProvider.createFolder({
+    token: remote.accessToken,
+    creds: remote.creds,
+    remoteId: remote.remoteId,
+  });
 }
`)
	assertFindings(t, SLP138{}.Check(d), 0, "SLP138", SeverityWarn)
}

func TestSLP139_FiresWhenRawS3SiblingRemains(t *testing.T) {
	root := t.TempDir()
	sibling := filepath.Join(root, "api/src/services/driveListService.js")
	if err := os.MkdirAll(filepath.Dir(sibling), 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(sibling, []byte(`function open(accessToken) {
  const creds = JSON.parse(accessToken);
  return new S3Client({ credentials: creds });
}
`), 0o644); err != nil {
		t.Fatalf("write sibling: %v", err)
	}

	d := parseDiffWithRoot(t, root, `diff --git a/api/src/routes/userRemotes.js b/api/src/routes/userRemotes.js
--- a/api/src/routes/userRemotes.js
+++ b/api/src/routes/userRemotes.js
@@ -1,3 +1,5 @@
 async function load(accessToken) {
+  const creds = parseAndNormalizeStoredS3Creds(accessToken, provider);
 }
`)
	assertFindings(t, SLP139{}.Check(d), 1, "SLP139", SeverityWarn)
}

func TestSLP139_IgnoresWhenSiblingAlreadyUsesHardening(t *testing.T) {
	root := t.TempDir()
	sibling := filepath.Join(root, "api/src/services/driveListService.js")
	if err := os.MkdirAll(filepath.Dir(sibling), 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(sibling, []byte(`function open(accessToken) {
  const creds = parseAndNormalizeStoredS3Creds(accessToken, provider);
  return new S3Client(s3ClientConfigSafe(creds));
}
`), 0o644); err != nil {
		t.Fatalf("write sibling: %v", err)
	}

	d := parseDiffWithRoot(t, root, `diff --git a/api/src/routes/userRemotes.js b/api/src/routes/userRemotes.js
--- a/api/src/routes/userRemotes.js
+++ b/api/src/routes/userRemotes.js
@@ -1,3 +1,5 @@
 async function load(accessToken) {
+  const creds = parseAndNormalizeStoredS3Creds(accessToken, provider);
 }
`)
	assertFindings(t, SLP139{}.Check(d), 0, "SLP139", SeverityWarn)
}

func TestSLP140_FiresOnUnguardedHardenerCall(t *testing.T) {
	d := parseDiff(t, `diff --git a/api/src/routes/userRemotes.js b/api/src/routes/userRemotes.js
--- a/api/src/routes/userRemotes.js
+++ b/api/src/routes/userRemotes.js
@@ -1,3 +1,6 @@
 async function save(provider, accessToken) {
+  const normalizedAccessToken = await hardenServerCredentials(provider, accessToken);
+  return normalizedAccessToken;
 }
`)
	assertFindings(t, SLP140{}.Check(d), 1, "SLP140", SeverityWarn)
}

func TestSLP140_IgnoresProviderGuardedHardenerCall(t *testing.T) {
	d := parseDiff(t, `diff --git a/api/src/routes/userRemotes.js b/api/src/routes/userRemotes.js
--- a/api/src/routes/userRemotes.js
+++ b/api/src/routes/userRemotes.js
@@ -1,3 +1,8 @@
 async function save(provider, accessToken) {
+  if (SERVER_PROVIDERS.has(provider)) {
+    const normalizedAccessToken = await hardenServerCredentials(provider, accessToken);
+    return normalizedAccessToken;
+  }
 }
`)
	assertFindings(t, SLP140{}.Check(d), 0, "SLP140", SeverityWarn)
}

func TestSLP140_IgnoresJSONGuardedHardenerCall(t *testing.T) {
	d := parseDiff(t, `diff --git a/api/src/routes/userRemotes.js b/api/src/routes/userRemotes.js
--- a/api/src/routes/userRemotes.js
+++ b/api/src/routes/userRemotes.js
@@ -1,3 +1,8 @@
 async function save(provider, accessToken) {
+  if (accessToken.trim().startsWith("{")) {
+    const normalizedAccessToken = await hardenServerCredentials(provider, accessToken);
+    return normalizedAccessToken;
+  }
 }
`)
	assertFindings(t, SLP140{}.Check(d), 0, "SLP140", SeverityWarn)
}
