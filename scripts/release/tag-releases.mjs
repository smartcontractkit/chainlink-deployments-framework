#!/usr/bin/env node

import { execFileSync } from "node:child_process";
import { readFileSync } from "node:fs";
import { dirname, resolve } from "node:path";
import { fileURLToPath } from "node:url";

const repoRoot = resolve(dirname(fileURLToPath(import.meta.url)), "../..");
const dryRun = process.argv.includes("--dry-run");
const rootPackageJSONPath = resolve(repoRoot, "package.json");
const operationsGenPackageJSONPath = resolve(repoRoot, "tools/operations-gen/package.json");
const rootPackageJSON = JSON.parse(readFileSync(rootPackageJSONPath, "utf8"));
const operationsGenPackageJSON = JSON.parse(readFileSync(operationsGenPackageJSONPath, "utf8"));
const rootVersion = rootPackageJSON.version;
const operationsGenVersion = operationsGenPackageJSON.version;

const SEMVER_RE = /^\d+\.\d+\.\d+(-[\w.]+)?(\+[\w.]+)?$/;

function validateSemver(version, source) {
  if (!version) {
    throw new Error(`Missing version in ${source}`);
  }
  if (!SEMVER_RE.test(version)) {
    throw new Error(`Invalid semver "${version}" in ${source}`);
  }
}

validateSemver(rootVersion, rootPackageJSONPath);
validateSemver(operationsGenVersion, operationsGenPackageJSONPath);

const tagsToCreate = [`v${rootVersion}`, `tools/operations-gen/v${operationsGenVersion}`];

function git(...args) {
  return execFileSync("git", args, {
    cwd: repoRoot,
    encoding: "utf8",
    stdio: ["ignore", "pipe", "pipe"],
  }).trim();
}

try {
  git("fetch", "--tags", "--quiet");
} catch (err) {
  console.warn(`Warning: unable to refresh tags: ${err.message}`);
}

for (const tag of tagsToCreate) {
  // Remote is authoritative: check origin first so a locally-created-but-unpushed
  // tag does not cause us to silently skip the push.
  let existsOnRemote = false;
  try {
    const remoteTag = git("ls-remote", "--tags", "origin", `refs/tags/${tag}`);
    existsOnRemote = !!remoteTag;
  } catch (err) {
    if (dryRun) {
      console.warn(`[dry-run] Unable to verify remote tag ${tag}: ${err.message}`);
      console.log(`[dry-run] Would create and push ${tag}`);
      continue;
    }
    throw new Error(`Failed checking remote tag ${tag}: ${err.message}`);
  }

  if (existsOnRemote) {
    console.log(`Tag ${tag} already exists on origin; skipping.`);
    continue;
  }

  if (dryRun) {
    console.log(`[dry-run] Would create and push ${tag}`);
    continue;
  }

  // Force-create the local tag at HEAD so a stale local tag with the same
  // name cannot accidentally be pushed to origin.
  git("tag", "-f", tag);
  git("push", "origin", `refs/tags/${tag}`);
  console.log(`Created and pushed ${tag}`);
}
