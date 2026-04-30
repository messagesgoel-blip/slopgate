package rules

import "testing"

func TestSLP128_FiresOnPositiveBotQueuePriority(t *testing.T) {
	d := parseDiff(t, `diff --git a/api/src/routes/slackBot.js b/api/src/routes/slackBot.js
--- a/api/src/routes/slackBot.js
+++ b/api/src/routes/slackBot.js
@@ -1,3 +1,6 @@
 async function enqueue(queue) {
+  await queue.add('bot.search', buildJobEnvelope(payload), {
+    priority: 1,
+  })
 }
`)
	assertFindings(t, SLP128{}.Check(d), 1, "SLP128", SeverityWarn)
}

func TestSLP128_IgnoresNegativePriority(t *testing.T) {
	d := parseDiff(t, `diff --git a/api/src/routes/slackBot.js b/api/src/routes/slackBot.js
--- a/api/src/routes/slackBot.js
+++ b/api/src/routes/slackBot.js
@@ -1,3 +1,6 @@
 async function enqueue(queue) {
+  await queue.add('bot.search', buildJobEnvelope(payload), {
+    priority: -1,
+  })
 }
`)
	assertFindings(t, SLP128{}.Check(d), 0, "SLP128", SeverityWarn)
}

func TestSLP129_FiresOnTrackedLiveEnv(t *testing.T) {
	d := parseDiff(t, `diff --git a/.env b/.env
--- a/.env
+++ b/.env
@@ -1,1 +1,3 @@
+VITE_SUPABASE_URL=https://project.supabase.co
+VITE_SUPABASE_ANON_KEY=prodAnonKeyValue123456789
`)
	got := SLP129{}.Check(d)
	assertFindings(t, got, 2, "SLP129", SeverityBlock)
}

func TestSLP129_IgnoresEnvExamplePlaceholders(t *testing.T) {
	d := parseDiff(t, `diff --git a/.env.example b/.env.example
--- a/.env.example
+++ b/.env.example
@@ -1,1 +1,3 @@
+VITE_SUPABASE_URL=https://example.supabase.co
+VITE_SUPABASE_ANON_KEY=your_key_here
`)
	assertFindings(t, SLP129{}.Check(d), 0, "SLP129", SeverityBlock)
}

func TestSLP130_FiresOnHardcodedProductionRedirect(t *testing.T) {
	d := parseDiff(t, `diff --git a/src/Root.jsx b/src/Root.jsx
--- a/src/Root.jsx
+++ b/src/Root.jsx
@@ -1,3 +1,5 @@
 function goHome() {
+  window.location.assign("https://numeracode.com/numera/")
 }
`)
	assertFindings(t, SLP130{}.Check(d), 1, "SLP130", SeverityWarn)
}

func TestSLP130_IgnoresLocalhostRedirect(t *testing.T) {
	d := parseDiff(t, `diff --git a/src/Root.jsx b/src/Root.jsx
--- a/src/Root.jsx
+++ b/src/Root.jsx
@@ -1,3 +1,5 @@
 function goHome() {
+  window.location.assign("http://localhost:5173/numera/")
 }
`)
	assertFindings(t, SLP130{}.Check(d), 0, "SLP130", SeverityWarn)
}

func TestSLP131_FiresOnNestedReactLink(t *testing.T) {
	d := parseDiff(t, `diff --git a/src/blog/components/BlogCard.tsx b/src/blog/components/BlogCard.tsx
--- a/src/blog/components/BlogCard.tsx
+++ b/src/blog/components/BlogCard.tsx
@@ -1,3 +1,8 @@
 export function BlogCard({ post }) {
+  return <Link to={"/post/" + post.slug}>
+    <article>
+      <Link to="/category/news">{post.category}</Link>
+    </article>
+  </Link>
 }
`)
	assertFindings(t, SLP131{}.Check(d), 1, "SLP131", SeverityWarn)
}

func TestSLP131_FiresOnSameLineNestedAnchorAndLink(t *testing.T) {
	cases := []struct {
		name string
		line string
	}{
		{name: "link wrapping anchor", line: `+  return <Link to="/post"><a href="/category">News</a></Link>`},
		{name: "anchor wrapping link", line: `+  return <a href="/post"><Link to="/category">News</Link></a>`},
		{name: "link wrapping link", line: `+  return <Link to="/post"><Link to="/category">News</Link></Link>`},
		{name: "anchor wrapping anchor", line: `+  return <a href="/post"><a href="/category">News</a></a>`},
		{name: "link with self-closing child wrapping anchor", line: `+  return <Link to="/post"><Icon /><a href="/category">News</a></Link>`},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			d := parseDiff(t, `diff --git a/src/blog/components/BlogCard.tsx b/src/blog/components/BlogCard.tsx
--- a/src/blog/components/BlogCard.tsx
+++ b/src/blog/components/BlogCard.tsx
@@ -1,3 +1,5 @@
 export function BlogCard() {
`+tc.line+`
 }
`)
			assertFindings(t, SLP131{}.Check(d), 1, "SLP131", SeverityWarn)
		})
	}
}

func TestSLP131_IgnoresSiblingReactLinks(t *testing.T) {
	d := parseDiff(t, `diff --git a/src/Nav.tsx b/src/Nav.tsx
--- a/src/Nav.tsx
+++ b/src/Nav.tsx
@@ -1,3 +1,7 @@
 export function Nav() {
+  return <nav>
+    <Link to="/a">A</Link>
+    <Link to="/b">B</Link>
+    <Link to="/c">C</Link><Link to="/d">D</Link>
+  </nav>
 }
`)
	assertFindings(t, SLP131{}.Check(d), 0, "SLP131", SeverityWarn)
}

func TestSLP132_FiresOnShortcutWithoutEditableGuard(t *testing.T) {
	d := parseDiff(t, `diff --git a/src/blog/components/SearchBar.tsx b/src/blog/components/SearchBar.tsx
--- a/src/blog/components/SearchBar.tsx
+++ b/src/blog/components/SearchBar.tsx
@@ -1,3 +1,12 @@
 export function SearchBar() {
+  React.useEffect(() => {
+    const handler = (event) => {
+      if ((event.metaKey || event.ctrlKey) && event.key === "k") {
+        setOpen(true)
+      }
+    }
+    window.addEventListener("keydown", handler)
+  }, [])
 }
`)
	assertFindings(t, SLP132{}.Check(d), 1, "SLP132", SeverityWarn)
}

func TestSLP132_IgnoresShortcutWithEditableGuard(t *testing.T) {
	d := parseDiff(t, `diff --git a/src/blog/components/SearchBar.tsx b/src/blog/components/SearchBar.tsx
--- a/src/blog/components/SearchBar.tsx
+++ b/src/blog/components/SearchBar.tsx
@@ -1,3 +1,15 @@
 export function SearchBar() {
+  React.useEffect(() => {
+    const handler = (event) => {
+      const target = event.target
+      if (target instanceof HTMLInputElement || target.isContentEditable) return
+      if ((event.metaKey || event.ctrlKey) && event.key === "k") {
+        setOpen(true)
+      }
+    }
+    window.addEventListener("keydown", handler)
+  }, [])
 }
`)
	assertFindings(t, SLP132{}.Check(d), 0, "SLP132", SeverityWarn)
}

func TestSLP132_FiresWhenInputIsMentionedWithoutTargetGuard(t *testing.T) {
	d := parseDiff(t, `diff --git a/src/blog/components/SearchBar.tsx b/src/blog/components/SearchBar.tsx
--- a/src/blog/components/SearchBar.tsx
+++ b/src/blog/components/SearchBar.tsx
@@ -1,3 +1,13 @@
 export function SearchBar() {
+  React.useEffect(() => {
+    const handler = (event) => {
+      const inputHint = "input shortcut"
+      if ((event.metaKey || event.ctrlKey) && event.key === "k") {
+        setOpen(inputHint)
+      }
+    }
+    window.addEventListener("keydown", handler)
+  }, [])
 }
`)
	assertFindings(t, SLP132{}.Check(d), 1, "SLP132", SeverityWarn)
}

func TestSLP132_IgnoresLocalElementKeyDown(t *testing.T) {
	d := parseDiff(t, `diff --git a/src/blog/components/SearchInput.tsx b/src/blog/components/SearchInput.tsx
--- a/src/blog/components/SearchInput.tsx
+++ b/src/blog/components/SearchInput.tsx
@@ -1,3 +1,10 @@
 export function SearchInput() {
+  const handleKeyDown = (event) => {
+    if ((event.metaKey || event.ctrlKey) && event.key === "k") {
+      setOpen(true)
+    }
+  }
+  return <input onKeyDown={handleKeyDown} />
 }
`)
	assertFindings(t, SLP132{}.Check(d), 0, "SLP132", SeverityWarn)
}

func TestSLP133_FiresOnInlineExpressRawParser(t *testing.T) {
	d := parseDiff(t, `diff --git a/api/src/routes/discordBot.js b/api/src/routes/discordBot.js
--- a/api/src/routes/discordBot.js
+++ b/api/src/routes/discordBot.js
@@ -1,3 +1,5 @@
+router.post('/', express.raw({ type: 'application/json' }), async (req, res) => {
+  res.sendStatus(204)
+})
`)
	assertFindings(t, SLP133{}.Check(d), 1, "SLP133", SeverityWarn)
}

func TestSLP133_IgnoresParserNotPassedToRouter(t *testing.T) {
	d := parseDiff(t, `diff --git a/api/src/routes/discordBot.js b/api/src/routes/discordBot.js
--- a/api/src/routes/discordBot.js
+++ b/api/src/routes/discordBot.js
@@ -1,3 +1,7 @@
+const parser = express.json()
+router.get('/health', health)
+app.use(parser)
`)
	assertFindings(t, SLP133{}.Check(d), 0, "SLP133", SeverityWarn)
}

func TestSLP134_FiresOnPersistedTransferArrays(t *testing.T) {
	d := parseDiff(t, `diff --git a/api/src/services/structuralAutomationRunService.js b/api/src/services/structuralAutomationRunService.js
--- a/api/src/services/structuralAutomationRunService.js
+++ b/api/src/services/structuralAutomationRunService.js
@@ -1,3 +1,8 @@
 const completedSummary = {
+  status: 'completed',
+  transferIds: summary.transferIds,
+  skippedTransfers: summary.skippedTransfers,
+}
+metadata.structuralRuntime.lastRun = completedSummary
`)
	got := SLP134{}.Check(d)
	assertFindings(t, got, 2, "SLP134", SeverityWarn)
}

func TestSLP134_FiresOnStringifiedSummaryWithTransferArrays(t *testing.T) {
	d := parseDiff(t, `diff --git a/api/src/services/structuralAutomationRunService.js b/api/src/services/structuralAutomationRunService.js
--- a/api/src/services/structuralAutomationRunService.js
+++ b/api/src/services/structuralAutomationRunService.js
@@ -1,3 +1,7 @@
 function record(ids) {
+  const summary = { transferIds: ids, status: 'completed' }
+  audit_logs.insert({ metadata: JSON.stringify(summary) })
 }
`)
	assertFindings(t, SLP134{}.Check(d), 1, "SLP134", SeverityWarn)
}

func TestSLP134_IgnoresStringifiedBoundedSummary(t *testing.T) {
	d := parseDiff(t, `diff --git a/api/src/services/structuralAutomationRunService.js b/api/src/services/structuralAutomationRunService.js
--- a/api/src/services/structuralAutomationRunService.js
+++ b/api/src/services/structuralAutomationRunService.js
@@ -1,3 +1,7 @@
 function record(count) {
+  const summary = { transferCount: count, status: 'completed' }
+  metadata.lastRun = JSON.stringify(summary)
 }
`)
	assertFindings(t, SLP134{}.Check(d), 0, "SLP134", SeverityWarn)
}

func TestSLP134_IgnoresIntermediateTransferArrays(t *testing.T) {
	d := parseDiff(t, `diff --git a/api/src/services/run.js b/api/src/services/run.js
--- a/api/src/services/run.js
+++ b/api/src/services/run.js
@@ -1,3 +1,7 @@
 function build(summary) {
+  const transient = {
+    transferIds: summary.transferIds,
+    skippedTransfers: summary.skippedTransfers,
+  }
 }
`)
	assertFindings(t, SLP134{}.Check(d), 0, "SLP134", SeverityWarn)
}

func TestSLP135_FiresOnRawErrMessageInSummary(t *testing.T) {
	d := parseDiff(t, `diff --git a/api/src/services/structuralAutomationHandlers.js b/api/src/services/structuralAutomationHandlers.js
--- a/api/src/services/structuralAutomationHandlers.js
+++ b/api/src/services/structuralAutomationHandlers.js
@@ -1,3 +1,8 @@
 function record(summary, err) {
+  summary.deleteFailures.push({
+    path: ref.path,
+    error: err.message,
+  })
 }
`)
	assertFindings(t, SLP135{}.Check(d), 1, "SLP135", SeverityWarn)
}

func TestSLP135_IgnoresThrownErrMessage(t *testing.T) {
	d := parseDiff(t, `diff --git a/api/src/services/run.js b/api/src/services/run.js
--- a/api/src/services/run.js
+++ b/api/src/services/run.js
@@ -1,3 +1,5 @@
 function handle(err) {
+  throw new Error(err.message)
 }
`)
	assertFindings(t, SLP135{}.Check(d), 0, "SLP135", SeverityWarn)
}
