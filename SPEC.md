# git-wrapped spec

Every implementation (Go, Python, JavaScript) follows these rules exactly. Output must be byte-identical;
`tests/run.sh` enforces it. Use integer arithmetic everywhere (no floats), and measure string length in
Unicode code points.

## 1. Command line

```
git-wrapped [paths...] [--year N] [--author TEXT] [--everyone] [--svg FILE] [--json] [--no-color]
```

- Flags and paths may appear in any order. Value flags accept `--flag value` or `--flag=value`.
- Boolean flags given `=value` are errors.
- `-h/--help` and `-v/--version` print and exit 0.
- Unknown flag, missing value, or invalid year (not all digits, or outside 1970–9999): message on stderr, exit **2**.
- A path that is not a directory: exit **1**. No repos found: exit **1**.
- Default path is `.`. Default year is the current local year.

## 2. Finding repos

For each path (made absolute, symlinks not resolved), walk directories up to depth 3:

- A directory containing `.git` (file or folder) is a repo; do not descend into it.
- Skip entries starting with `.`, and directories named `node_modules` or `vendor`. Do not follow symlinks.

Repos are deduplicated and sorted by absolute path. A repo's display name is its basename.

## 3. Reading history

For each repo, run:

```
git -C <repo> -c core.quotepath=off log --branches --remotes --tags --no-merges --no-renames
    --since=<year-1>-12-25 --until=<year+1>-01-31
    --pretty=format:%x1e%H%x1f%an%x1f%ae%x1f%aI%x1f%s --numstat
```

If the command exits non-zero, print `git-wrapped: skipping <repo> (git log failed)` on stderr and continue.

Parsing: split stdout on `\x1e` and drop the first chunk. In each record, line 1 is split on `\x1f` into at most 5
fields (hash, name, email, ISO date, subject). Each later non-empty line is `added\tdeleted\tpath`, split into at most 3
parts; a count that is not all ASCII digits (binary files show `-`) is 0.

Keep a commit only if all of these hold, checked in order:

1. Its hash has not been seen yet (in any repo). Mark it seen.
2. The first 4 characters of its author date equal the year.
3. Author filter:
   - `--everyone`: keep all.
   - `--author TEXT`: lowercase TEXT is a substring of the lowercase name or lowercase email.
   - Neither: `git -C <first repo> config user.email`, trimmed and lowercased. If empty, behave as `--everyone`.
     Otherwise keep commits whose lowercase email equals it exactly.

All times come straight from the author date string (author's own timezone): local date = chars 0–9,
hour = chars 11–12. Weekday is `(days_from_civil(y, m, d) + 3) mod 7`, Monday = 0.

## 4. Files

A file is **ignored** (contributes nothing at all) when its lowercased basename is a lockfile
(`package-lock.json`, `yarn.lock`, `pnpm-lock.yaml`, `go.sum`, `cargo.lock`, `poetry.lock`, `composer.lock`,
`gemfile.lock`, `bun.lockb`) or any directory segment is exactly `node_modules` or `vendor`.

Language: lowercased basename `dockerfile` → Docker, `makefile` → Makefile. Otherwise take the text after the last `.`
of the lowercased basename, if that dot is not at position 0, and look it up in the extension table (identical in all
implementations). No match: no language.

File key: the path, prefixed with `<repo name>/` when more than one repo was found.

## 5. Stats (JSON field order)

| Field | Rule |
| --- | --- |
| `year` | the year |
| `author` | `"All contributors"` if everyone mode; else the most frequent author name (ties: smallest name); `""` if no commits |
| `everyone` | bool |
| `repos` | repo display names in order |
| `commits` | kept commits |
| `active_days` | distinct local dates |
| `lines_added`, `lines_deleted` | sums over non-ignored files |
| `files_touched` | distinct file keys |
| `busiest_day` | `{date, commits}`; most commits, ties earliest date; `{"", 0}` if none |
| `busiest_weekday` | full English name of the first max weekday; `""` if none |
| `busiest_hour` | first max hour 0–23; `-1` if none |
| `longest_streak` | `{days, start, end}`: longest run of consecutive dates, first one wins ties; `{0, "", ""}` if none |
| `weekdays`, `hours`, `months` | counts, 7 (Mon first) / 24 / 12 |
| `top_languages` | up to 5 `{name, lines, percent}`, lines = added + deleted; sort lines desc then name asc; `percent = lines * 100 // total lines over all languages` |
| `top_files` | up to 5 `{path, commits}`, commits counted once per commit; sort commits desc then path asc |
| `fix_commits` | lowercased subject contains `fix` or `bug` |
| `longest_message` | `{length, subject}`; longest subject in code points, first wins ties; `{0, ""}` if none |
| `personality` | `{id, title, description}`, see below |

`--json` prints this with 2-space indentation, non-ASCII unescaped, followed by a newline. Nothing else goes to stdout.

## 6. Personality

With `night` = commits in hours 22, 23, 0, 1, 2, 3; `early` = hours 5–8; `weekend` = Saturday + Sunday;
`total` = commits. The first matching rule wins:

| id | Rule | Title | Description |
| --- | --- | --- | --- |
| ghost | total = 0 | The Ghost | A mysterious and powerful silence. |
| night_owl | night·100 ≥ total·30 | The Night Owl | Your best code happens after dark. |
| early_bird | early·100 ≥ total·25 | The Early Bird | Shipping before the coffee kicks in. |
| weekend_warrior | weekend·100 ≥ total·30 | The Weekend Warrior | Saturdays are for side projects. |
| code_janitor | deleted > added | The Code Janitor | Deleted more than you wrote. Legend. |
| commit_machine | total ≥ active_days·8 | The Commit Machine | Small commits, all day long. |
| bug_hunter | fix·100 ≥ total·30 | The Bug Hunter | Nothing escapes you. Especially bugs. |
| steady_shipper | otherwise | The Steady Shipper | Consistent, reliable, always shipping. |

## 7. Terminal and SVG output

The reference is the Python code in `python/src/git_wrapped/render.py` and `card.py`; the Go and JS versions mirror it
line by line. Helpers shared by both:

- `fmt(n)`: thousands separated by `,`.
- `plural(n, word)`: `fmt(n) + " " + word`, plus `s` unless n = 1.
- `bar(n, max, width)`: `n*width // max` full blocks (at least 1 if n > 0), padded with `░`.
- `spark(values)`: space for 0, else `▁▂▃▄▅▆▇█`[v*7 // max].
- `truncate_end(s, n)` keeps the first n-1 code points + `…`; `truncate_start` keeps the last n-1.
- Colors only when stdout is a TTY, `NO_COLOR` is unset and `--no-color` is not given. Each colored span is
  `ESC[<code>m … ESC[0m`.

`--svg FILE` writes the card and prints `Saved share card to FILE` on stderr.
