// Terminal output. Must stay byte-identical to the Python and Go versions (see SPEC.md).
import { runeLen } from "./stats.js";

const MONTHS = ["Jan", "Feb", "Mar", "Apr", "May", "Jun", "Jul", "Aug", "Sep", "Oct", "Nov", "Dec"];
const SHORT_WEEKDAYS = ["Mon", "Tue", "Wed", "Thu", "Fri", "Sat", "Sun"];
const SPARK = [..."▁▂▃▄▅▆▇█"];

export function fmt(n) {
  const s = String(Math.abs(n)).replace(/\B(?=(\d{3})+(?!\d))/g, ",");
  return n < 0 ? "-" + s : s;
}
export const plural = (n, word) => `${fmt(n)} ${word}${n === 1 ? "" : "s"}`;

export function hourLabel(h) {
  if (h === 0) return "12 AM";
  if (h < 12) return `${h} AM`;
  if (h === 12) return "12 PM";
  return `${h - 12} PM`;
}
const shortDate = (day) => `${MONTHS[Number(day.slice(5, 7)) - 1]} ${Number(day.slice(8, 10))}`;

function bar(n, mx, width) {
  let filled = mx === 0 ? 0 : Math.floor((n * width) / mx);
  if (n > 0 && filled === 0) filled = 1;
  return "█".repeat(filled) + "░".repeat(width - filled);
}
function spark(values) {
  const mx = Math.max(...values);
  return values.map((v) => (v === 0 ? " " : SPARK[Math.floor((v * 7) / mx)])).join("");
}
export function truncateStart(s, n) {
  const r = [...s];
  return r.length <= n ? s : "…" + r.slice(r.length - (n - 1)).join("");
}
export function truncateEnd(s, n) {
  const r = [...s];
  return r.length <= n ? s : r.slice(0, n - 1).join("") + "…";
}
const padEnd = (s, n) => s + " ".repeat(Math.max(0, n - runeLen(s)));
const padStart = (s, n) => " ".repeat(Math.max(0, n - runeLen(s))) + s;

function palette(enabled) {
  const wrap = (code) => (t) => (enabled ? `\x1b[${code}m${t}\x1b[0m` : t);
  return { bold: wrap("1"), dim: wrap("2"), magenta: wrap("35"), cyan: wrap("36"), yellow: wrap("33") };
}

export function renderText(s, color = false) {
  const p = palette(color);
  const out = [""];
  out.push("  " + p.bold(p.magenta(`★ GIT WRAPPED ${s.year} ★`)));
  const repos = s.repos;
  const where = repos.length > 1 ? plural(repos.length, "repo") : repos[0];
  out.push("  " + p.dim(`${s.author || "nobody"} · ${where}`));
  out.push("");

  if (s.commits === 0) {
    out.push("  No commits found for this year.");
    out.push("  Try --everyone, --author <name> or --year <year>.");
    out.push("");
    return out.join("\n") + "\n";
  }

  const row = (value, label) =>
    out.push("  " + p.bold(value) + " ".repeat(Math.max(1, 20 - runeLen(value))) + p.dim(label));

  const st = s.longest_streak;
  const bd = s.busiest_day;
  row(plural(s.commits, "commit"), "on " + plural(s.active_days, "active day"));
  row(`+${fmt(s.lines_added)} / -${fmt(s.lines_deleted)}`, "lines added / deleted");
  row(plural(st.days, "day"), `longest streak (${shortDate(st.start)} – ${shortDate(st.end)})`);
  row(s.busiest_weekday, "busiest weekday");
  row(hourLabel(s.busiest_hour), "busiest hour");
  row(shortDate(bd.date), `busiest day (${plural(bd.commits, "commit")})`);
  out.push("");

  out.push("  " + p.bold("When you code"));
  let mx = Math.max(...s.weekdays);
  s.weekdays.forEach((n, i) => out.push(`  ${SHORT_WEEKDAYS[i]} ` + p.cyan(bar(n, mx, 24)) + ` ${fmt(n)}`));
  out.push("");
  out.push("  Hours  " + p.cyan(spark(s.hours)));
  out.push("         " + p.dim("0     6     12    18"));
  out.push("  Months " + p.cyan(spark(s.months)));
  out.push("         " + p.dim("JFMAMJJASOND"));

  if (s.top_languages.length) {
    out.push("");
    out.push("  " + p.bold("Top languages"));
    mx = s.top_languages[0].lines;
    for (const lang of s.top_languages) {
      out.push("  " + padEnd(lang.name, 12) + " " + p.magenta(bar(lang.lines, mx, 20)) + " " + padStart(`${lang.percent}%`, 4));
    }
  }

  if (s.top_files.length) {
    out.push("");
    out.push("  " + p.bold("Most edited files"));
    s.top_files.forEach((f, i) =>
      out.push(`  ${i + 1}. ${truncateStart(f.path, 50)} ` + p.dim(`(${plural(f.commits, "commit")})`)),
    );
  }

  const lm = s.longest_message;
  out.push("");
  out.push("  " + p.bold("Fun facts"));
  out.push(`  · ${fmt(s.fix_commits)} of ${plural(s.commits, "commit message")} mention fix or bug`);
  out.push(`  · ${plural(s.files_touched, "file")} touched across ${plural(repos.length, "repo")}`);
  out.push(`  · Longest commit message: ${plural(lm.length, "character")}`);
  out.push("    " + p.dim('"' + truncateEnd(lm.subject, 60) + '"'));

  const pers = s.personality;
  out.push("");
  out.push("  You are " + p.bold(p.yellow(pers.title.toUpperCase())));
  out.push("  " + p.dim(pers.description));
  out.push("");
  out.push("  " + p.dim("Make your share card:  git-wrapped --svg wrapped.svg"));
  out.push("");
  return out.join("\n") + "\n";
}
