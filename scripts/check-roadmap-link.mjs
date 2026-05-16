#!/usr/bin/env node

import fs from "node:fs";

const eventPath = process.env.GITHUB_EVENT_PATH;
if (!eventPath) {
  console.error("GITHUB_EVENT_PATH is required");
  process.exit(2);
}

if (!fs.existsSync(eventPath)) {
  console.error(`Event payload does not exist at ${eventPath}`);
  process.exit(2);
}

let event;
try {
  event = JSON.parse(fs.readFileSync(eventPath, "utf8"));
} catch (error) {
  console.error(`Unable to read or parse event payload at ${eventPath}: ${error.message}`);
  process.exit(2);
}

const pr = event.pull_request;
if (!pr) {
  process.exit(0);
}

const labels = (pr.labels || []).map((label) => label.name);
if (labels.includes("no-roadmap-task")) {
  process.exit(0);
}

const body = pr.body || "";
const closingKeyword =
  /\b(close[sd]?|fix(e[sd])?|resolve[sd]?)\s*:?\s+((messagesgoel-blip\/slopgate)?#\d+)/gi;
const matches = [...body.matchAll(closingKeyword)];

if (matches.length !== 1) {
  console.error(
    "PR body must include exactly one closing keyword such as `Closes #123`, or the PR must be labeled `no-roadmap-task`.",
  );
  process.exit(1);
}
