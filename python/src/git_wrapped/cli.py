import datetime
import json
import os
import subprocess
import sys

from . import __version__
from .card import render_svg
from .render import render_text
from .stats import compute, parse_log

HELP = """git-wrapped - your year in code, Spotify Wrapped style

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
"""

VALUE_FLAGS = {"--year", "--author", "--svg"}
BOOL_FLAGS = {"--everyone", "--json", "--no-color"}
SKIP_DIRS = {"node_modules", "vendor"}


def fail(msg, code=1):
    sys.stderr.write(f"git-wrapped: {msg}\n")
    sys.exit(code)


def parse_args(argv):
    opts = {"paths": [], "--year": None, "--author": None, "--svg": None,
            "--everyone": False, "--json": False, "--no-color": False}
    i = 0
    while i < len(argv):
        arg = argv[i]
        if arg in ("-h", "--help"):
            sys.stdout.write(HELP)
            sys.exit(0)
        if arg in ("-v", "--version"):
            sys.stdout.write(f"git-wrapped {__version__}\n")
            sys.exit(0)
        if arg.startswith("-"):
            name, eq, value = arg.partition("=")
            if name in VALUE_FLAGS:
                if not eq:
                    i += 1
                    if i >= len(argv):
                        fail(f"{name} needs a value", 2)
                    value = argv[i]
                opts[name] = value
            elif name in BOOL_FLAGS and not eq:
                opts[name] = True
            else:
                fail(f"unknown option {arg} (see --help)", 2)
        else:
            opts["paths"].append(arg)
        i += 1
    if opts["--year"] is not None:
        y = opts["--year"]
        if not y.isdigit() or not 1970 <= int(y) <= 9999:
            fail(f"invalid year {y}", 2)
        opts["--year"] = int(y)
    return opts


def find_repos(paths):
    found = set()

    def walk(d, depth):
        if os.path.exists(os.path.join(d, ".git")):
            found.add(d)
            return
        if depth == 0:
            return
        try:
            names = sorted(os.listdir(d))
        except OSError:
            return
        for name in names:
            full = os.path.join(d, name)
            if name.startswith(".") or name in SKIP_DIRS:
                continue
            if os.path.isdir(full) and not os.path.islink(full):
                walk(full, depth - 1)

    for p in paths:
        if not os.path.isdir(p):
            fail(f"not a directory: {p}")
        walk(os.path.abspath(p), 3)
    return sorted(found)


def git(repo, *args):
    try:
        res = subprocess.run(["git", "-C", repo, *args], capture_output=True)
    except FileNotFoundError:
        fail("git is not installed or not on PATH")
    return res.returncode, res.stdout.decode("utf-8", errors="replace")


def main(argv=None):
    if hasattr(sys.stdout, "reconfigure"):
        sys.stdout.reconfigure(encoding="utf-8")
    opts = parse_args(sys.argv[1:] if argv is None else argv)
    repos = find_repos(opts["paths"] or ["."])
    if not repos:
        fail("no git repositories found")
    year = opts["--year"] or datetime.date.today().year

    everyone = opts["--everyone"]
    query = (opts["--author"] or "").lower()
    email = ""
    if not everyone and not query:
        _, email = git(repos[0], "config", "user.email")
        email = email.strip().lower()
        if not email:
            everyone = True

    names = [os.path.basename(r) for r in repos]
    commits, seen = [], set()
    for repo, name in zip(repos, names):
        code, out = git(
            repo, "-c", "core.quotepath=off", "log", "--branches", "--remotes", "--tags",
            "--no-merges", "--no-renames", f"--since={year - 1}-12-25", f"--until={year + 1}-01-31",
            "--pretty=format:%x1e%H%x1f%an%x1f%ae%x1f%aI%x1f%s", "--numstat",
        )
        if code != 0:
            sys.stderr.write(f"git-wrapped: skipping {repo} (git log failed)\n")
            continue
        for c in parse_log(out, name):
            if c["hash"] in seen:
                continue
            seen.add(c["hash"])
            if c["date"][:4] != str(year):
                continue
            if not everyone:
                if query and query not in c["name"].lower() and query not in c["email"].lower():
                    continue
                if not query and c["email"].lower() != email:
                    continue
            commits.append(c)

    stats = compute(commits, year, names, everyone)
    if opts["--json"]:
        sys.stdout.write(json.dumps(stats, indent=2, ensure_ascii=False) + "\n")
    else:
        color = (not opts["--no-color"] and "NO_COLOR" not in os.environ
                 and sys.stdout.isatty())
        sys.stdout.write(render_text(stats, color))

    if opts["--svg"]:
        with open(opts["--svg"], "w", encoding="utf-8", newline="\n") as f:
            f.write(render_svg(stats))
        sys.stderr.write(f"Saved share card to {opts['--svg']}\n")
