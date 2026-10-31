"""Parse `git log` output and compute the yearly stats. See SPEC.md."""

EXTENSIONS = {
    "py": "Python", "go": "Go", "js": "JavaScript", "mjs": "JavaScript", "cjs": "JavaScript",
    "jsx": "JavaScript", "ts": "TypeScript", "tsx": "TypeScript", "rs": "Rust", "java": "Java",
    "kt": "Kotlin", "kts": "Kotlin", "swift": "Swift", "c": "C", "h": "C", "cc": "C++",
    "cpp": "C++", "cxx": "C++", "hpp": "C++", "cs": "C#", "rb": "Ruby", "php": "PHP",
    "scala": "Scala", "dart": "Dart", "lua": "Lua", "r": "R", "jl": "Julia", "ex": "Elixir",
    "exs": "Elixir", "erl": "Erlang", "hs": "Haskell", "ml": "OCaml", "clj": "Clojure",
    "zig": "Zig", "nim": "Nim", "sh": "Shell", "bash": "Shell", "zsh": "Shell",
    "ps1": "PowerShell", "sql": "SQL", "html": "HTML", "htm": "HTML", "css": "CSS",
    "scss": "CSS", "sass": "CSS", "less": "CSS", "vue": "Vue", "svelte": "Svelte",
    "md": "Markdown", "mdx": "Markdown", "json": "JSON", "yaml": "YAML", "yml": "YAML",
    "toml": "TOML", "xml": "XML", "ipynb": "Jupyter", "tf": "Terraform", "sol": "Solidity",
    "m": "Objective-C", "mm": "Objective-C", "pl": "Perl", "proto": "Protobuf",
    "graphql": "GraphQL", "gql": "GraphQL",
}
FILENAMES = {"dockerfile": "Docker", "makefile": "Makefile"}
LOCKFILES = {
    "package-lock.json", "yarn.lock", "pnpm-lock.yaml", "go.sum", "cargo.lock",
    "poetry.lock", "composer.lock", "gemfile.lock", "bun.lockb",
}
IGNORED_DIRS = {"node_modules", "vendor"}

PERSONALITIES = {
    "ghost": ("The Ghost", "A mysterious and powerful silence."),
    "night_owl": ("The Night Owl", "Your best code happens after dark."),
    "early_bird": ("The Early Bird", "Shipping before the coffee kicks in."),
    "weekend_warrior": ("The Weekend Warrior", "Saturdays are for side projects."),
    "code_janitor": ("The Code Janitor", "Deleted more than you wrote. Legend."),
    "commit_machine": ("The Commit Machine", "Small commits, all day long."),
    "bug_hunter": ("The Bug Hunter", "Nothing escapes you. Especially bugs."),
    "steady_shipper": ("The Steady Shipper", "Consistent, reliable, always shipping."),
}


def is_ignored(path):
    parts = path.split("/")
    if parts[-1].lower() in LOCKFILES:
        return True
    return any(p in IGNORED_DIRS for p in parts[:-1])


def language_of(path):
    base = path.rsplit("/", 1)[-1].lower()
    if base in FILENAMES:
        return FILENAMES[base]
    i = base.rfind(".")
    if i <= 0:
        return None
    return EXTENSIONS.get(base[i + 1:])


def days_from_civil(y, m, d):
    """Days since 1970-01-01 (proleptic Gregorian)."""
    y -= 1 if m <= 2 else 0
    era = y // 400
    yoe = y - era * 400
    doy = (153 * (m + (-3 if m > 2 else 9)) + 2) // 5 + d - 1
    doe = yoe * 365 + yoe // 4 - yoe // 100 + doy
    return era * 146097 + doe - 719468


def parse_log(output, repo):
    """Turn `git log --pretty=format:%x1e%H%x1f%an%x1f%ae%x1f%aI%x1f%s --numstat` into dicts."""
    commits = []
    for record in output.split("\x1e")[1:]:
        lines = record.split("\n")
        fields = lines[0].split("\x1f", 4)
        if len(fields) < 5:
            continue
        files = []
        for line in lines[1:]:
            if not line:
                continue
            parts = line.split("\t", 2)
            if len(parts) < 3:
                continue
            added = int(parts[0]) if parts[0].isdigit() else 0
            deleted = int(parts[1]) if parts[1].isdigit() else 0
            files.append((added, deleted, parts[2]))
        commits.append({
            "hash": fields[0], "name": fields[1], "email": fields[2],
            "date": fields[3], "subject": fields[4], "files": files, "repo": repo,
        })
    return commits


def _top(counter, limit):
    return sorted(counter.items(), key=lambda kv: (-kv[1], kv[0]))[:limit]


def compute(commits, year, repo_names, everyone):
    total = len(commits)
    hours, weekdays, months = [0] * 24, [0] * 7, [0] * 12
    per_day, names, langs, files = {}, {}, {}, {}
    added = deleted = fix = 0
    longest = ""
    longest_len = -1
    multi = len(repo_names) > 1

    for c in commits:
        date = c["date"]
        day = date[:10]
        y, m, d = int(day[:4]), int(day[5:7]), int(day[8:10])
        hours[int(date[11:13])] += 1
        weekdays[(days_from_civil(y, m, d) + 3) % 7] += 1
        months[m - 1] += 1
        per_day[day] = per_day.get(day, 0) + 1
        names[c["name"]] = names.get(c["name"], 0) + 1
        subject = c["subject"]
        low = subject.lower()
        if "fix" in low or "bug" in low:
            fix += 1
        if len(subject) > longest_len:
            longest, longest_len = subject, len(subject)
        seen = set()
        for a, dl, path in c["files"]:
            if is_ignored(path):
                continue
            added += a
            deleted += dl
            lang = language_of(path)
            if lang:
                langs[lang] = langs.get(lang, 0) + a + dl
            key = c["repo"] + "/" + path if multi else path
            if key not in seen:
                seen.add(key)
                files[key] = files.get(key, 0) + 1

    busiest_day = {"date": "", "commits": 0}
    if per_day:
        date, n = _top(per_day, 1)[0]
        busiest_day = {"date": date, "commits": n}

    streak = {"days": 0, "start": "", "end": ""}
    run, start, prev = 0, "", None
    for day in sorted(per_day):
        n = days_from_civil(int(day[:4]), int(day[5:7]), int(day[8:10]))
        if prev is not None and n == prev + 1:
            run += 1
        else:
            run, start = 1, day
        if run > streak["days"]:
            streak = {"days": run, "start": start, "end": day}
        prev = n

    lang_total = sum(langs.values())
    top_languages = [
        {"name": k, "lines": v, "percent": v * 100 // lang_total}
        for k, v in _top(langs, 5) if v > 0
    ]
    top_files = [{"path": k, "commits": v} for k, v in _top(files, 5)]

    weekday_names = ["Monday", "Tuesday", "Wednesday", "Thursday", "Friday", "Saturday", "Sunday"]
    if everyone:
        author = "All contributors"
    elif names:
        author = _top(names, 1)[0][0]
    else:
        author = ""

    return {
        "year": year,
        "author": author,
        "everyone": everyone,
        "repos": list(repo_names),
        "commits": total,
        "active_days": len(per_day),
        "lines_added": added,
        "lines_deleted": deleted,
        "files_touched": len(files),
        "busiest_day": busiest_day,
        "busiest_weekday": weekday_names[weekdays.index(max(weekdays))] if total else "",
        "busiest_hour": hours.index(max(hours)) if total else -1,
        "longest_streak": streak,
        "weekdays": weekdays,
        "hours": hours,
        "months": months,
        "top_languages": top_languages,
        "top_files": top_files,
        "fix_commits": fix,
        "longest_message": {"length": max(longest_len, 0), "subject": longest},
        "personality": personality(total, len(per_day), hours, weekdays, added, deleted, fix),
    }


def personality(total, active_days, hours, weekdays, added, deleted, fix):
    night = sum(hours[22:]) + sum(hours[:4])
    early = sum(hours[5:9])
    weekend = weekdays[5] + weekdays[6]
    if total == 0:
        pid = "ghost"
    elif night * 100 >= total * 30:
        pid = "night_owl"
    elif early * 100 >= total * 25:
        pid = "early_bird"
    elif weekend * 100 >= total * 30:
        pid = "weekend_warrior"
    elif deleted > added:
        pid = "code_janitor"
    elif total >= active_days * 8:
        pid = "commit_machine"
    elif fix * 100 >= total * 30:
        pid = "bug_hunter"
    else:
        pid = "steady_shipper"
    title, description = PERSONALITIES[pid]
    return {"id": pid, "title": title, "description": description}
