import { execFileSync } from "node:child_process";
import { existsSync, lstatSync, readdirSync, statSync, writeFileSync } from "node:fs";
import { basename, join, resolve } from "node:path";
import { renderSvg } from "./card.js";
import { renderText } from "./render.js";
import { compute, parseLog } from "./stats.js";

export const VERSION = "0.1.0";

const HELP = `git-wrapped - your year in code, Spotify Wrapped style

Usage:
  git-wrapped [paths...] [options]

  Paths can be repos or folders containing repos (searched 3 levels deep).
  Defaults to the current directory.

Options:
  --year <year>     Year to wrap (default: current year)
  --author <text>   Only commits whose author name or email contains <text>
  --everyone        Include every author (default: you, from git config user.email)
  --svg <file>      Also write a 1200x630 share card as SVG
  --json            Print raw stats as JSON
  --no-color        Disable colors (also respects NO_COLOR)
  -v, --version     Print version
  -h, --help        Show this help
`;

const VALUE_FLAGS = new Set(["--year", "--author", "--svg"]);
const BOOL_FLAGS = new Set(["--everyone", "--json", "--no-color"]);
const SKIP_DIRS = new Set(["node_modules", "vendor"]);

function fail(msg, code = 1) {
  process.stderr.write(`git-wrapped: ${msg}\n`);
  process.exit(code);
}

function parseArgs(argv) {
  const opts = { paths: [], "--year": null, "--author": null, "--svg": null, "--everyone": false, "--json": false, "--no-color": false };
  for (let i = 0; i < argv.length; i++) {
    const arg = argv[i];
    if (arg === "-h" || arg === "--help") { process.stdout.write(HELP); process.exit(0); }
    if (arg === "-v" || arg === "--version") { process.stdout.write(`git-wrapped ${VERSION}\n`); process.exit(0); }
    if (arg.startsWith("-")) {
      const eq = arg.indexOf("=");
      const name = eq < 0 ? arg : arg.slice(0, eq);
      if (VALUE_FLAGS.has(name)) {
        let value = eq < 0 ? undefined : arg.slice(eq + 1);
        if (value === undefined) {
          i++;
          if (i >= argv.length) fail(`${name} needs a value`, 2);
          value = argv[i];
        }
        opts[name] = value;
      } else if (BOOL_FLAGS.has(name) && eq < 0) {
        opts[name] = true;
      } else {
        fail(`unknown option ${arg} (see --help)`, 2);
      }
    } else {
      opts.paths.push(arg);
    }
  }
  if (opts["--year"] !== null) {
    const y = opts["--year"];
    if (!/^[0-9]+$/.test(y) || Number(y) < 1970 || Number(y) > 9999) fail(`invalid year ${y}`, 2);
    opts["--year"] = Number(y);
  }
  return opts;
}

function findRepos(paths) {
  const found = new Set();
  const walk = (d, depth) => {
    if (existsSync(join(d, ".git"))) { found.add(d); return; }
    if (depth === 0) return;
    let names;
    try { names = readdirSync(d).sort(); } catch { return; }
    for (const name of names) {
      if (name.startsWith(".") || SKIP_DIRS.has(name)) continue;
      const full = join(d, name);
      try {
        if (lstatSync(full).isDirectory()) walk(full, depth - 1);
      } catch { /* unreadable, skip */ }
    }
  };
  for (const p of paths) {
    let ok = false;
    try { ok = statSync(p).isDirectory(); } catch { /* missing */ }
    if (!ok) fail(`not a directory: ${p}`);
    walk(resolve(p), 3);
  }
  return [...found].sort();
}

function git(repo, ...args) {
  try {
    const out = execFileSync("git", ["-C", repo, ...args], {
      encoding: "utf8", stdio: ["ignore", "pipe", "pipe"], maxBuffer: 1 << 30,
    });
    return [0, out];
  } catch (err) {
    if (err.code === "ENOENT") fail("git is not installed or not on PATH");
    return [err.status ?? 1, err.stdout || ""];
  }
}

export function main(argv = process.argv.slice(2)) {
  const opts = parseArgs(argv);
  const repos = findRepos(opts.paths.length ? opts.paths : ["."]);
  if (!repos.length) fail("no git repositories found");
  const year = opts["--year"] ?? new Date().getFullYear();

  let everyone = opts["--everyone"];
  const query = (opts["--author"] || "").toLowerCase();
  let email = "";
  if (!everyone && !query) {
    email = git(repos[0], "config", "user.email")[1].trim().toLowerCase();
    if (!email) everyone = true;
  }

  const names = repos.map((r) => basename(r));
  const commits = [];
  const seen = new Set();
  repos.forEach((repo, idx) => {
    const [code, out] = git(
      repo, "-c", "core.quotepath=off", "log", "--branches", "--remotes", "--tags",
      "--no-merges", "--no-renames", `--since=${year - 1}-12-25`, `--until=${year + 1}-01-31`,
      "--pretty=format:%x1e%H%x1f%an%x1f%ae%x1f%aI%x1f%s", "--numstat",
    );
    if (code !== 0) { process.stderr.write(`git-wrapped: skipping ${repo} (git log failed)\n`); return; }
    for (const c of parseLog(out, names[idx])) {
      if (seen.has(c.hash)) continue;
      seen.add(c.hash);
      if (c.date.slice(0, 4) !== String(year)) continue;
      if (!everyone) {
        if (query && !c.name.toLowerCase().includes(query) && !c.email.toLowerCase().includes(query)) continue;
        if (!query && c.email.toLowerCase() !== email) continue;
      }
      commits.push(c);
    }
  });

  const stats = compute(commits, year, names, everyone);
  if (opts["--json"]) {
    process.stdout.write(JSON.stringify(stats, null, 2) + "\n");
  } else {
    const color = !opts["--no-color"] && !("NO_COLOR" in process.env) && process.stdout.isTTY === true;
    process.stdout.write(renderText(stats, color));
  }

  if (opts["--svg"]) {
    writeFileSync(opts["--svg"], renderSvg(stats), "utf8");
    process.stderr.write(`Saved share card to ${opts["--svg"]}\n`);
  }
}
