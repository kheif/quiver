#!/usr/bin/env node
// Builds the npm packages from GoReleaser's dist/ and publishes them:
// one package per platform with its binary, then @kheif/quiver, which
// depends on all of them as optionalDependencies. npm installs only the
// one whose os and cpu match.
//
//   node scripts/npm-publish.mjs <version> [--dry-run] [--pack]
//
// --dry-run runs npm publish --dry-run; --pack only writes the packages to
// dist/npm. Versions already on the registry are skipped, so a failed run
// can be repeated.
import { execFileSync } from "node:child_process";
import { chmodSync, copyFileSync, cpSync, mkdirSync, readFileSync, rmSync, writeFileSync } from "node:fs";
import { join, resolve } from "node:path";

const scope = "@kheif";
// Node's platform and arch names, mapped to GoReleaser's.
const targets = [
  { os: "darwin", cpu: "arm64", goos: "darwin", goarch: "arm64" },
  { os: "darwin", cpu: "x64", goos: "darwin", goarch: "amd64" },
  { os: "linux", cpu: "arm64", goos: "linux", goarch: "arm64" },
  { os: "linux", cpu: "x64", goos: "linux", goarch: "amd64" },
  { os: "win32", cpu: "arm64", goos: "windows", goarch: "arm64" },
  { os: "win32", cpu: "x64", goos: "windows", goarch: "amd64" },
];

const args = process.argv.slice(2);
const version = args.find((a) => !a.startsWith("--"))?.replace(/^v/, "");
const dryRun = args.includes("--dry-run");
const packOnly = args.includes("--pack");
if (!version) {
  console.error("usage: node scripts/npm-publish.mjs <version> [--dry-run] [--pack]");
  process.exit(2);
}

const main = JSON.parse(readFileSync("npm/quiver/package.json", "utf8"));
const artifacts = JSON.parse(readFileSync("dist/artifacts.json", "utf8"));
const out = join("dist", "npm");
rmSync(out, { recursive: true, force: true });

function binaryOf(t) {
  const a = artifacts.find((x) => x.type === "Binary" && x.goos === t.goos && x.goarch === t.goarch);
  if (!a) throw new Error(`no ${t.goos}/${t.goarch} binary in dist/artifacts.json`);
  return a.path;
}

const common = {
  version,
  description: main.description,
  homepage: main.homepage,
  bugs: main.bugs,
  repository: main.repository,
};
if (main.license) common.license = main.license;

const dirs = [];
for (const t of targets) {
  const name = `${scope}/quiver-${t.os}-${t.cpu}`;
  const dir = join(out, `quiver-${t.os}-${t.cpu}`);
  const file = t.os === "win32" ? "quiver.exe" : "quiver";
  mkdirSync(join(dir, "bin"), { recursive: true });
  copyFileSync(binaryOf(t), join(dir, "bin", file));
  chmodSync(join(dir, "bin", file), 0o755);
  const pkg = { name, ...common, os: [t.os], cpu: [t.cpu], files: ["bin"] };
  pkg.description = `The ${t.os} ${t.cpu} binary of ${scope}/quiver. Install ${scope}/quiver instead.`;
  writeFileSync(join(dir, "package.json"), JSON.stringify(pkg, null, 2) + "\n");
  copyFileSync("LICENSE", join(dir, "LICENSE"));
  dirs.push({ name, dir });
}

const mainDir = join(out, "quiver");
cpSync("npm/quiver", mainDir, { recursive: true });
copyFileSync("README.md", join(mainDir, "README.md"));
copyFileSync("LICENSE", join(mainDir, "LICENSE"));
main.version = version;
main.optionalDependencies = Object.fromEntries(dirs.map((d) => [d.name, version]));
writeFileSync(join(mainDir, "package.json"), JSON.stringify(main, null, 2) + "\n");
// The main package goes last, so its dependencies exist when it appears.
dirs.push({ name: main.name, dir: mainDir });

if (packOnly) {
  for (const d of dirs) execFileSync("npm", ["pack", "--pack-destination", resolve(out)], { cwd: d.dir, stdio: "inherit" });
  process.exit(0);
}

function published(name) {
  try {
    execFileSync("npm", ["view", `${name}@${version}`, "version"], { stdio: ["ignore", "pipe", "ignore"] });
    return true;
  } catch {
    return false;
  }
}

for (const d of dirs) {
  if (!dryRun && published(d.name)) {
    console.log(`${d.name}@${version} is already published; skipped`);
    continue;
  }
  const cmd = ["publish", "--access", "public"];
  if (dryRun) cmd.push("--dry-run");
  // A prerelease such as 1.1.0-rc.1 must not become the default install.
  if (version.includes("-")) cmd.push("--tag", "next");
  if (process.env.GITHUB_ACTIONS) cmd.push("--provenance");
  console.log(`npm ${cmd.join(" ")}  (${d.name}@${version})`);
  execFileSync("npm", cmd, { cwd: d.dir, stdio: "inherit" });
}
