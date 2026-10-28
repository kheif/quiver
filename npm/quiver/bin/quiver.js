#!/usr/bin/env node
// Runs the quiver binary of this platform. npm installs only the platform
// package that matches (see optionalDependencies in package.json).
"use strict";

const { spawnSync } = require("node:child_process");

const packages = {
  "darwin arm64": "@kheif/quiver-darwin-arm64",
  "darwin x64": "@kheif/quiver-darwin-x64",
  "linux arm64": "@kheif/quiver-linux-arm64",
  "linux x64": "@kheif/quiver-linux-x64",
  "win32 arm64": "@kheif/quiver-win32-arm64",
  "win32 x64": "@kheif/quiver-win32-x64",
};

const pkg = packages[`${process.platform} ${process.arch}`];
if (!pkg) {
  console.error(`quiver: there is no build for ${process.platform} ${process.arch}`);
  process.exit(1);
}
const file = process.platform === "win32" ? "quiver.exe" : "quiver";
let bin;
try {
  bin = require.resolve(`${pkg}/bin/${file}`);
} catch {
  console.error(`quiver: ${pkg} is missing. Reinstall without --omit=optional: npm install -g @kheif/quiver`);
  process.exit(1);
}

// QUIVER_NPM tells quiver mcp install that it runs from an npm install.
const r = spawnSync(bin, process.argv.slice(2), {
  stdio: "inherit",
  env: { ...process.env, QUIVER_NPM: "1" },
});
if (r.error) {
  console.error(`quiver: ${r.error.message}`);
  process.exit(1);
}
if (r.signal) {
  process.kill(process.pid, r.signal);
}
process.exit(r.status ?? 1);
