// Command git-wrapped shows your year in code, Spotify Wrapped style.
package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"
)

const version = "0.1.0"

const help = `git-wrapped - your year in code, Spotify Wrapped style

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
`

var valueFlags = map[string]bool{"--year": true, "--author": true, "--svg": true}
var boolFlags = map[string]bool{"--everyone": true, "--json": true, "--no-color": true}
var skipDirs = map[string]bool{"node_modules": true, "vendor": true}

type options struct {
	paths                   []string
	year                    int
	author, svg             string
	everyone, json, noColor bool
}

func fail(msg string, code int) {
	fmt.Fprintf(os.Stderr, "git-wrapped: %s\n", msg)
	os.Exit(code)
}

func parseArgs(argv []string) options {
	opts := options{}
	values := map[string]string{}
	for i := 0; i < len(argv); i++ {
		arg := argv[i]
		switch arg {
		case "-h", "--help":
			fmt.Print(help)
			os.Exit(0)
		case "-v", "--version":
			fmt.Printf("git-wrapped %s\n", version)
			os.Exit(0)
		}
		if !strings.HasPrefix(arg, "-") {
			opts.paths = append(opts.paths, arg)
			continue
		}
		name, value, hasEq := strings.Cut(arg, "=")
		switch {
		case valueFlags[name]:
			if !hasEq {
				i++
				if i >= len(argv) {
					fail(name+" needs a value", 2)
				}
				value = argv[i]
			}
			values[name] = value
		case boolFlags[name] && !hasEq:
			switch name {
			case "--everyone":
				opts.everyone = true
			case "--json":
				opts.json = true
			case "--no-color":
				opts.noColor = true
			}
		default:
			fail("unknown option "+arg+" (see --help)", 2)
		}
	}
	if y, ok := values["--year"]; ok {
		n, err := strconv.Atoi(y)
		if !isDigits(y) || err != nil || n < 1970 || n > 9999 {
			fail("invalid year "+y, 2)
		}
		opts.year = n
	}
	opts.author = values["--author"]
	opts.svg = values["--svg"]
	return opts
}

func findRepos(paths []string) []string {
	found := map[string]bool{}
	var walk func(d string, depth int)
	walk = func(d string, depth int) {
		if _, err := os.Stat(filepath.Join(d, ".git")); err == nil {
			found[d] = true
			return
		}
		if depth == 0 {
			return
		}
		entries, err := os.ReadDir(d) // sorted by name
		if err != nil {
			return
		}
		for _, e := range entries {
			name := e.Name()
			if strings.HasPrefix(name, ".") || skipDirs[name] {
				continue
			}
			if e.IsDir() { // DirEntry.IsDir does not follow symlinks
				walk(filepath.Join(d, name), depth-1)
			}
		}
	}
	for _, p := range paths {
		info, err := os.Stat(p)
		if err != nil || !info.IsDir() {
			fail("not a directory: "+p, 1)
		}
		abs, _ := filepath.Abs(p)
		walk(abs, 3)
	}
	repos := make([]string, 0, len(found))
	for r := range found {
		repos = append(repos, r)
	}
	sort.Strings(repos)
	return repos
}

func git(repo string, args ...string) (int, string) {
	cmd := exec.Command("git", append([]string{"-C", repo}, args...)...)
	var stdout bytes.Buffer
	cmd.Stdout = &stdout
	err := cmd.Run()
	if err != nil {
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) {
			return exitErr.ExitCode(), stdout.String()
		}
		fail("git is not installed or not on PATH", 1)
	}
	return 0, stdout.String()
}

func isTerminal(f *os.File) bool {
	info, err := f.Stat()
	return err == nil && info.Mode()&os.ModeCharDevice != 0
}

func main() {
	opts := parseArgs(os.Args[1:])
	paths := opts.paths
	if len(paths) == 0 {
		paths = []string{"."}
	}
	repos := findRepos(paths)
	if len(repos) == 0 {
		fail("no git repositories found", 1)
	}
	year := opts.year
	if year == 0 {
		year = time.Now().Year()
	}

	everyone := opts.everyone
	query := strings.ToLower(opts.author)
	email := ""
	if !everyone && query == "" {
		_, out := git(repos[0], "config", "user.email")
		email = strings.ToLower(strings.TrimSpace(out))
		if email == "" {
			everyone = true
		}
	}

	names := make([]string, len(repos))
	for i, r := range repos {
		names[i] = filepath.Base(r)
	}
	yearStr := strconv.Itoa(year)
	var commits []Commit
	seen := map[string]bool{}
	for i, repo := range repos {
		code, out := git(repo, "-c", "core.quotepath=off", "log", "--branches", "--remotes", "--tags",
			"--no-merges", "--no-renames",
			fmt.Sprintf("--since=%d-12-25", year-1), fmt.Sprintf("--until=%d-01-31", year+1),
			"--pretty=format:%x1e%H%x1f%an%x1f%ae%x1f%aI%x1f%s", "--numstat")
		if code != 0 {
			fmt.Fprintf(os.Stderr, "git-wrapped: skipping %s (git log failed)\n", repo)
			continue
		}
		for _, c := range parseLog(out, names[i]) {
			if seen[c.Hash] {
				continue
			}
			seen[c.Hash] = true
			if c.Date[:4] != yearStr {
				continue
			}
			if !everyone {
				if query != "" && !strings.Contains(strings.ToLower(c.Name), query) && !strings.Contains(strings.ToLower(c.Email), query) {
					continue
				}
				if query == "" && strings.ToLower(c.Email) != email {
					continue
				}
			}
			commits = append(commits, c)
		}
	}

	stats := compute(commits, year, names, everyone)
	if opts.json {
		enc := json.NewEncoder(os.Stdout)
		enc.SetEscapeHTML(false)
		enc.SetIndent("", "  ")
		if err := enc.Encode(stats); err != nil {
			fail(err.Error(), 1)
		}
	} else {
		_, noColorEnv := os.LookupEnv("NO_COLOR")
		color := !opts.noColor && !noColorEnv && isTerminal(os.Stdout)
		fmt.Print(renderText(stats, color))
	}

	if opts.svg != "" {
		if err := os.WriteFile(opts.svg, []byte(renderSVG(stats)), 0o644); err != nil {
			fail(err.Error(), 1)
		}
		fmt.Fprintf(os.Stderr, "Saved share card to %s\n", opts.svg)
	}
}
