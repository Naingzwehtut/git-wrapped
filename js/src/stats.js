// Parse `git log` output and compute the yearly stats. See SPEC.md.

export const EXTENSIONS = {
  py: "Python", go: "Go", js: "JavaScript", mjs: "JavaScript", cjs: "JavaScript",
  jsx: "JavaScript", ts: "TypeScript", tsx: "TypeScript", rs: "Rust", java: "Java",
  kt: "Kotlin", kts: "Kotlin", swift: "Swift", c: "C", h: "C", cc: "C++",
  cpp: "C++", cxx: "C++", hpp: "C++", cs: "C#", rb: "Ruby", php: "PHP",
  scala: "Scala", dart: "Dart", lua: "Lua", r: "R", jl: "Julia", ex: "Elixir",
  exs: "Elixir", erl: "Erlang", hs: "Haskell", ml: "OCaml", clj: "Clojure",
  zig: "Zig", nim: "Nim", sh: "Shell", bash: "Shell", zsh: "Shell",
  ps1: "PowerShell", sql: "SQL", html: "HTML", htm: "HTML", css: "CSS",
  scss: "CSS", sass: "CSS", less: "CSS", vue: "Vue", svelte: "Svelte",
  md: "Markdown", mdx: "Markdown", json: "JSON", yaml: "YAML", yml: "YAML",
  toml: "TOML", xml: "XML", ipynb: "Jupyter", tf: "Terraform", sol: "Solidity",
  m: "Objective-C", mm: "Objective-C", pl: "Perl", proto: "Protobuf",
  graphql: "GraphQL", gql: "GraphQL",
};
const FILENAMES = { dockerfile: "Docker", makefile: "Makefile" };
const LOCKFILES = new Set([
  "package-lock.json", "yarn.lock", "pnpm-lock.yaml", "go.sum", "cargo.lock",
  "poetry.lock", "composer.lock", "gemfile.lock", "bun.lockb",
]);
const IGNORED_DIRS = new Set(["node_modules", "vendor"]);

const PERSONALITIES = {
  ghost: ["The Ghost", "A mysterious and powerful silence."],
  night_owl: ["The Night Owl", "Your best code happens after dark."],
  early_bird: ["The Early Bird", "Shipping before the coffee kicks in."],
  weekend_warrior: ["The Weekend Warrior", "Saturdays are for side projects."],
  code_janitor: ["The Code Janitor", "Deleted more than you wrote. Legend."],
  commit_machine: ["The Commit Machine", "Small commits, all day long."],
  bug_hunter: ["The Bug Hunter", "Nothing escapes you. Especially bugs."],
  steady_shipper: ["The Steady Shipper", "Consistent, reliable, always shipping."],
};

const WEEKDAY_NAMES = ["Monday", "Tuesday", "Wednesday", "Thursday", "Friday", "Saturday", "Sunday"];

/** Length in Unicode code points (matches Python len() and Go RuneCount). */
export const runeLen = (s) => [...s].length;

export function isIgnored(path) {
  const parts = path.split("/");
  if (LOCKFILES.has(parts[parts.length - 1].toLowerCase())) return true;
  return parts.slice(0, -1).some((p) => IGNORED_DIRS.has(p));
}

export function languageOf(path) {
  const base = path.slice(path.lastIndexOf("/") + 1).toLowerCase();
  if (Object.hasOwn(FILENAMES, base)) return FILENAMES[base];
  const i = base.lastIndexOf(".");
  if (i <= 0) return null;
  const ext = base.slice(i + 1);
  return Object.hasOwn(EXTENSIONS, ext) ? EXTENSIONS[ext] : null;
}

/** Days since 1970-01-01 (proleptic Gregorian). */
export function daysFromCivil(y, m, d) {
  y -= m <= 2 ? 1 : 0;
  const era = Math.floor(y / 400);
  const yoe = y - era * 400;
  const doy = Math.floor((153 * (m + (m > 2 ? -3 : 9)) + 2) / 5) + d - 1;
  const doe = yoe * 365 + Math.floor(yoe / 4) - Math.floor(yoe / 100) + doy;
  return era * 146097 + doe - 719468;
}

const isDigits = (s) => /^[0-9]+$/.test(s);

export function parseLog(output, repo) {
  const commits = [];
  for (const record of output.split("\x1e").slice(1)) {
    const lines = record.split("\n");
    const raw = lines[0].split("\x1f");
    if (raw.length < 5) continue;
    const fields = [...raw.slice(0, 4), raw.slice(4).join("\x1f")];
    const files = [];
    for (const line of lines.slice(1)) {
      if (!line) continue;
      const t1 = line.indexOf("\t");
      const t2 = t1 < 0 ? -1 : line.indexOf("\t", t1 + 1);
      if (t2 < 0) continue;
      const a = line.slice(0, t1);
      const d = line.slice(t1 + 1, t2);
      files.push([isDigits(a) ? Number(a) : 0, isDigits(d) ? Number(d) : 0, line.slice(t2 + 1)]);
    }
    commits.push({
      hash: fields[0], name: fields[1], email: fields[2],
      date: fields[3], subject: fields[4], files, repo,
    });
  }
  return commits;
}

const cmp = (a, b) => (a < b ? -1 : a > b ? 1 : 0);
function top(counter, limit) {
  return [...counter.entries()].sort((x, y) => y[1] - x[1] || cmp(x[0], y[0])).slice(0, limit);
}
const bump = (map, key, n = 1) => map.set(key, (map.get(key) || 0) + n);
const sum = (arr) => arr.reduce((a, b) => a + b, 0);

export function compute(commits, year, repoNames, everyone) {
  const total = commits.length;
  const hours = Array(24).fill(0), weekdays = Array(7).fill(0), months = Array(12).fill(0);
  const perDay = new Map(), names = new Map(), langs = new Map(), files = new Map();
  let added = 0, deleted = 0, fix = 0, longest = "", longestLen = -1;
  const multi = repoNames.length > 1;

  for (const c of commits) {
    const day = c.date.slice(0, 10);
    const y = Number(day.slice(0, 4)), m = Number(day.slice(5, 7)), d = Number(day.slice(8, 10));
    hours[Number(c.date.slice(11, 13))] += 1;
    weekdays[(daysFromCivil(y, m, d) + 3) % 7] += 1;
    months[m - 1] += 1;
    bump(perDay, day);
    bump(names, c.name);
    const low = c.subject.toLowerCase();
    if (low.includes("fix") || low.includes("bug")) fix += 1;
    const len = runeLen(c.subject);
    if (len > longestLen) { longest = c.subject; longestLen = len; }
    const seen = new Set();
    for (const [a, dl, path] of c.files) {
      if (isIgnored(path)) continue;
      added += a;
      deleted += dl;
      const lang = languageOf(path);
      if (lang) bump(langs, lang, a + dl);
      const key = multi ? `${c.repo}/${path}` : path;
      if (!seen.has(key)) { seen.add(key); bump(files, key); }
    }
  }

  let busiestDay = { date: "", commits: 0 };
  if (perDay.size) {
    const [date, n] = top(perDay, 1)[0];
    busiestDay = { date, commits: n };
  }

  let streak = { days: 0, start: "", end: "" };
  let run = 0, start = "", prev = null;
  for (const day of [...perDay.keys()].sort()) {
    const n = daysFromCivil(Number(day.slice(0, 4)), Number(day.slice(5, 7)), Number(day.slice(8, 10)));
    if (prev !== null && n === prev + 1) run += 1;
    else { run = 1; start = day; }
    if (run > streak.days) streak = { days: run, start, end: day };
    prev = n;
  }

  const langTotal = sum([...langs.values()]);
  const topLanguages = top(langs, 5)
    .filter(([, v]) => v > 0)
    .map(([name, lines]) => ({ name, lines, percent: Math.floor((lines * 100) / langTotal) }));
  const topFiles = top(files, 5).map(([path, n]) => ({ path, commits: n }));

  let author = "";
  if (everyone) author = "All contributors";
  else if (names.size) author = top(names, 1)[0][0];

  return {
    year,
    author,
    everyone,
    repos: [...repoNames],
    commits: total,
    active_days: perDay.size,
    lines_added: added,
    lines_deleted: deleted,
    files_touched: files.size,
    busiest_day: busiestDay,
    busiest_weekday: total ? WEEKDAY_NAMES[weekdays.indexOf(Math.max(...weekdays))] : "",
    busiest_hour: total ? hours.indexOf(Math.max(...hours)) : -1,
    longest_streak: streak,
    weekdays,
    hours,
    months,
    top_languages: topLanguages,
    top_files: topFiles,
    fix_commits: fix,
    longest_message: { length: Math.max(longestLen, 0), subject: longest },
    personality: personality(total, perDay.size, hours, weekdays, added, deleted, fix),
  };
}

export function personality(total, activeDays, hours, weekdays, added, deleted, fix) {
  const night = sum(hours.slice(22)) + sum(hours.slice(0, 4));
  const early = sum(hours.slice(5, 9));
  const weekend = weekdays[5] + weekdays[6];
  let id;
  if (total === 0) id = "ghost";
  else if (night * 100 >= total * 30) id = "night_owl";
  else if (early * 100 >= total * 25) id = "early_bird";
  else if (weekend * 100 >= total * 30) id = "weekend_warrior";
  else if (deleted > added) id = "code_janitor";
  else if (total >= activeDays * 8) id = "commit_machine";
  else if (fix * 100 >= total * 30) id = "bug_hunter";
  else id = "steady_shipper";
  const [title, description] = PERSONALITIES[id];
  return { id, title, description };
}
