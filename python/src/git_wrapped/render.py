"""Terminal output. Must stay byte-identical to the Go and JS versions (see SPEC.md)."""

MONTHS = ["Jan", "Feb", "Mar", "Apr", "May", "Jun", "Jul", "Aug", "Sep", "Oct", "Nov", "Dec"]
SHORT_WEEKDAYS = ["Mon", "Tue", "Wed", "Thu", "Fri", "Sat", "Sun"]
SPARK = "▁▂▃▄▅▆▇█"


def fmt(n):
    return f"{n:,}"


def plural(n, word):
    return f"{fmt(n)} {word}" + ("" if n == 1 else "s")


def hour_label(h):
    if h == 0:
        return "12 AM"
    if h < 12:
        return f"{h} AM"
    if h == 12:
        return "12 PM"
    return f"{h - 12} PM"


def short_date(day):
    return f"{MONTHS[int(day[5:7]) - 1]} {int(day[8:10])}"


def bar(n, mx, width):
    filled = 0 if mx == 0 else n * width // mx
    if n > 0 and filled == 0:
        filled = 1
    return "█" * filled + "░" * (width - filled)


def spark(values):
    mx = max(values)
    return "".join(" " if v == 0 else SPARK[v * 7 // mx] for v in values)


def truncate_start(s, n):
    return s if len(s) <= n else "…" + s[len(s) - (n - 1):]


def truncate_end(s, n):
    return s if len(s) <= n else s[: n - 1] + "…"


class Palette:
    def __init__(self, enabled):
        self.enabled = enabled

    def _wrap(self, code, text):
        return f"\x1b[{code}m{text}\x1b[0m" if self.enabled else text

    def bold(self, t): return self._wrap("1", t)
    def dim(self, t): return self._wrap("2", t)
    def magenta(self, t): return self._wrap("35", t)
    def cyan(self, t): return self._wrap("36", t)
    def yellow(self, t): return self._wrap("33", t)


def render_text(s, color=False):
    p = Palette(color)
    out = [""]
    out.append("  " + p.bold(p.magenta(f"★ GIT WRAPPED {s['year']} ★")))
    repos = s["repos"]
    where = plural(len(repos), "repo") if len(repos) > 1 else repos[0]
    out.append("  " + p.dim(f"{s['author'] or 'nobody'} · {where}"))
    out.append("")

    if s["commits"] == 0:
        out.append("  No commits found for this year.")
        out.append("  Try --everyone, --author <name> or --year <year>.")
        out.append("")
        return "\n".join(out) + "\n"

    def row(value, label):
        out.append("  " + p.bold(value) + " " * max(1, 20 - len(value)) + p.dim(label))

    st = s["longest_streak"]
    bd = s["busiest_day"]
    row(plural(s["commits"], "commit"), "on " + plural(s["active_days"], "active day"))
    row(f"+{fmt(s['lines_added'])} / -{fmt(s['lines_deleted'])}", "lines added / deleted")
    row(plural(st["days"], "day"), f"longest streak ({short_date(st['start'])} – {short_date(st['end'])})")
    row(s["busiest_weekday"], "busiest weekday")
    row(hour_label(s["busiest_hour"]), "busiest hour")
    row(short_date(bd["date"]), f"busiest day ({plural(bd['commits'], 'commit')})")
    out.append("")

    out.append("  " + p.bold("When you code"))
    mx = max(s["weekdays"])
    for i, n in enumerate(s["weekdays"]):
        out.append(f"  {SHORT_WEEKDAYS[i]} " + p.cyan(bar(n, mx, 24)) + f" {fmt(n)}")
    out.append("")
    out.append("  Hours  " + p.cyan(spark(s["hours"])))
    out.append("         " + p.dim("0     6     12    18"))
    out.append("  Months " + p.cyan(spark(s["months"])))
    out.append("         " + p.dim("JFMAMJJASOND"))

    if s["top_languages"]:
        out.append("")
        out.append("  " + p.bold("Top languages"))
        mx = s["top_languages"][0]["lines"]
        for lang in s["top_languages"]:
            out.append(
                "  " + lang["name"].ljust(12) + " " + p.magenta(bar(lang["lines"], mx, 20))
                + " " + f"{lang['percent']}%".rjust(4)
            )

    if s["top_files"]:
        out.append("")
        out.append("  " + p.bold("Most edited files"))
        for i, f in enumerate(s["top_files"]):
            out.append(f"  {i + 1}. {truncate_start(f['path'], 50)} " + p.dim(f"({plural(f['commits'], 'commit')})"))

    lm = s["longest_message"]
    out.append("")
    out.append("  " + p.bold("Fun facts"))
    out.append(f"  · {fmt(s['fix_commits'])} of {plural(s['commits'], 'commit message')} mention fix or bug")
    out.append(f"  · {plural(s['files_touched'], 'file')} touched across {plural(len(repos), 'repo')}")
    out.append(f"  · Longest commit message: {plural(lm['length'], 'character')}")
    out.append("    " + p.dim('"' + truncate_end(lm["subject"], 60) + '"'))

    pers = s["personality"]
    out.append("")
    out.append("  You are " + p.bold(p.yellow(pers["title"].upper())))
    out.append("  " + p.dim(pers["description"]))
    out.append("")
    out.append("  " + p.dim("Make your share card:  git-wrapped --svg wrapped.svg"))
    out.append("")
    return "\n".join(out) + "\n"
