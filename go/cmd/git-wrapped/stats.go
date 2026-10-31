package main

// Parse `git log` output and compute the yearly stats. See SPEC.md.

import (
	"sort"
	"strconv"
	"strings"
	"unicode/utf8"
)

var extensions = map[string]string{
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

var filenames = map[string]string{"dockerfile": "Docker", "makefile": "Makefile"}

var lockfiles = map[string]bool{
	"package-lock.json": true, "yarn.lock": true, "pnpm-lock.yaml": true, "go.sum": true,
	"cargo.lock": true, "poetry.lock": true, "composer.lock": true, "gemfile.lock": true,
	"bun.lockb": true,
}

var ignoredDirs = map[string]bool{"node_modules": true, "vendor": true}

var personalities = map[string][2]string{
	"ghost":           {"The Ghost", "A mysterious and powerful silence."},
	"night_owl":       {"The Night Owl", "Your best code happens after dark."},
	"early_bird":      {"The Early Bird", "Shipping before the coffee kicks in."},
	"weekend_warrior": {"The Weekend Warrior", "Saturdays are for side projects."},
	"code_janitor":    {"The Code Janitor", "Deleted more than you wrote. Legend."},
	"commit_machine":  {"The Commit Machine", "Small commits, all day long."},
	"bug_hunter":      {"The Bug Hunter", "Nothing escapes you. Especially bugs."},
	"steady_shipper":  {"The Steady Shipper", "Consistent, reliable, always shipping."},
}

var weekdayNames = []string{"Monday", "Tuesday", "Wednesday", "Thursday", "Friday", "Saturday", "Sunday"}

type FileChange struct {
	Added, Deleted int
	Path           string
}

type Commit struct {
	Hash, Name, Email, Date, Subject, Repo string
	Files                                  []FileChange
}

type DayCount struct {
	Date    string `json:"date"`
	Commits int    `json:"commits"`
}

type Streak struct {
	Days  int    `json:"days"`
	Start string `json:"start"`
	End   string `json:"end"`
}

type Language struct {
	Name    string `json:"name"`
	Lines   int    `json:"lines"`
	Percent int    `json:"percent"`
}

type FileCount struct {
	Path    string `json:"path"`
	Commits int    `json:"commits"`
}

type Message struct {
	Length  int    `json:"length"`
	Subject string `json:"subject"`
}

type Personality struct {
	ID          string `json:"id"`
	Title       string `json:"title"`
	Description string `json:"description"`
}

type Stats struct {
	Year           int         `json:"year"`
	Author         string      `json:"author"`
	Everyone       bool        `json:"everyone"`
	Repos          []string    `json:"repos"`
	Commits        int         `json:"commits"`
	ActiveDays     int         `json:"active_days"`
	LinesAdded     int         `json:"lines_added"`
	LinesDeleted   int         `json:"lines_deleted"`
	FilesTouched   int         `json:"files_touched"`
	BusiestDay     DayCount    `json:"busiest_day"`
	BusiestWeekday string      `json:"busiest_weekday"`
	BusiestHour    int         `json:"busiest_hour"`
	LongestStreak  Streak      `json:"longest_streak"`
	Weekdays       []int       `json:"weekdays"`
	Hours          []int       `json:"hours"`
	Months         []int       `json:"months"`
	TopLanguages   []Language  `json:"top_languages"`
	TopFiles       []FileCount `json:"top_files"`
	FixCommits     int         `json:"fix_commits"`
	LongestMessage Message     `json:"longest_message"`
	Personality    Personality `json:"personality"`
}

func isIgnored(path string) bool {
	parts := strings.Split(path, "/")
	if lockfiles[strings.ToLower(parts[len(parts)-1])] {
		return true
	}
	for _, p := range parts[:len(parts)-1] {
		if ignoredDirs[p] {
			return true
		}
	}
	return false
}

func languageOf(path string) string {
	base := strings.ToLower(path[strings.LastIndex(path, "/")+1:])
	if name, ok := filenames[base]; ok {
		return name
	}
	i := strings.LastIndex(base, ".")
	if i <= 0 {
		return ""
	}
	return extensions[base[i+1:]]
}

func floorDiv(a, b int) int {
	q := a / b
	if (a%b != 0) && ((a < 0) != (b < 0)) {
		q--
	}
	return q
}

// daysFromCivil returns days since 1970-01-01 (proleptic Gregorian).
func daysFromCivil(y, m, d int) int {
	if m <= 2 {
		y--
	}
	era := floorDiv(y, 400)
	yoe := y - era*400
	mm := m + 9
	if m > 2 {
		mm = m - 3
	}
	doy := (153*mm+2)/5 + d - 1
	doe := yoe*365 + yoe/4 - yoe/100 + doy
	return era*146097 + doe - 719468
}

func atoi(s string) int {
	n, _ := strconv.Atoi(s)
	return n
}

func isDigits(s string) bool {
	if s == "" {
		return false
	}
	for _, r := range s {
		if r < '0' || r > '9' {
			return false
		}
	}
	return true
}

func parseLog(output, repo string) []Commit {
	var commits []Commit
	records := strings.Split(output, "\x1e")
	for _, record := range records[1:] {
		lines := strings.Split(record, "\n")
		fields := strings.SplitN(lines[0], "\x1f", 5)
		if len(fields) < 5 {
			continue
		}
		c := Commit{Hash: fields[0], Name: fields[1], Email: fields[2], Date: fields[3], Subject: fields[4], Repo: repo}
		for _, line := range lines[1:] {
			if line == "" {
				continue
			}
			parts := strings.SplitN(line, "\t", 3)
			if len(parts) < 3 {
				continue
			}
			fc := FileChange{Path: parts[2]}
			if isDigits(parts[0]) {
				fc.Added = atoi(parts[0])
			}
			if isDigits(parts[1]) {
				fc.Deleted = atoi(parts[1])
			}
			c.Files = append(c.Files, fc)
		}
		commits = append(commits, c)
	}
	return commits
}

type kv struct {
	Key string
	N   int
}

func top(counter map[string]int, limit int) []kv {
	items := make([]kv, 0, len(counter))
	for k, v := range counter {
		items = append(items, kv{k, v})
	}
	sort.Slice(items, func(i, j int) bool {
		if items[i].N != items[j].N {
			return items[i].N > items[j].N
		}
		return items[i].Key < items[j].Key
	})
	if len(items) > limit {
		items = items[:limit]
	}
	return items
}

func sum(xs []int) int {
	t := 0
	for _, x := range xs {
		t += x
	}
	return t
}

func argmax(xs []int) int {
	best := 0
	for i, x := range xs {
		if x > xs[best] {
			best = i
		}
	}
	return best
}

func compute(commits []Commit, year int, repoNames []string, everyone bool) Stats {
	total := len(commits)
	hours, weekdays, months := make([]int, 24), make([]int, 7), make([]int, 12)
	perDay, names, langs, files := map[string]int{}, map[string]int{}, map[string]int{}, map[string]int{}
	added, deleted, fix := 0, 0, 0
	longest, longestLen := "", -1
	multi := len(repoNames) > 1

	for _, c := range commits {
		day := c.Date[:10]
		y, m, d := atoi(day[:4]), atoi(day[5:7]), atoi(day[8:10])
		hours[atoi(c.Date[11:13])]++
		weekdays[(daysFromCivil(y, m, d)+3)%7]++
		months[m-1]++
		perDay[day]++
		names[c.Name]++
		low := strings.ToLower(c.Subject)
		if strings.Contains(low, "fix") || strings.Contains(low, "bug") {
			fix++
		}
		if n := utf8.RuneCountInString(c.Subject); n > longestLen {
			longest, longestLen = c.Subject, n
		}
		seen := map[string]bool{}
		for _, f := range c.Files {
			if isIgnored(f.Path) {
				continue
			}
			added += f.Added
			deleted += f.Deleted
			if lang := languageOf(f.Path); lang != "" {
				langs[lang] += f.Added + f.Deleted
			}
			key := f.Path
			if multi {
				key = c.Repo + "/" + f.Path
			}
			if !seen[key] {
				seen[key] = true
				files[key]++
			}
		}
	}

	busiest := DayCount{}
	if len(perDay) > 0 {
		t := top(perDay, 1)[0]
		busiest = DayCount{t.Key, t.N}
	}

	days := make([]string, 0, len(perDay))
	for d := range perDay {
		days = append(days, d)
	}
	sort.Strings(days)
	streak := Streak{}
	run, start, prev, hasPrev := 0, "", 0, false
	for _, day := range days {
		n := daysFromCivil(atoi(day[:4]), atoi(day[5:7]), atoi(day[8:10]))
		if hasPrev && n == prev+1 {
			run++
		} else {
			run, start = 1, day
		}
		if run > streak.Days {
			streak = Streak{run, start, day}
		}
		prev, hasPrev = n, true
	}

	langTotal := 0
	for _, v := range langs {
		langTotal += v
	}
	topLanguages := []Language{}
	for _, t := range top(langs, 5) {
		if t.N > 0 {
			topLanguages = append(topLanguages, Language{t.Key, t.N, t.N * 100 / langTotal})
		}
	}
	topFiles := []FileCount{}
	for _, t := range top(files, 5) {
		topFiles = append(topFiles, FileCount{t.Key, t.N})
	}

	author := ""
	if everyone {
		author = "All contributors"
	} else if len(names) > 0 {
		author = top(names, 1)[0].Key
	}

	s := Stats{
		Year: year, Author: author, Everyone: everyone, Repos: append([]string{}, repoNames...),
		Commits: total, ActiveDays: len(perDay), LinesAdded: added, LinesDeleted: deleted,
		FilesTouched: len(files), BusiestDay: busiest, BusiestHour: -1, LongestStreak: streak,
		Weekdays: weekdays, Hours: hours, Months: months, TopLanguages: topLanguages,
		TopFiles: topFiles, FixCommits: fix,
		LongestMessage: Message{max(longestLen, 0), longest},
		Personality:    personality(total, len(perDay), hours, weekdays, added, deleted, fix),
	}
	if total > 0 {
		s.BusiestWeekday = weekdayNames[argmax(weekdays)]
		s.BusiestHour = argmax(hours)
	}
	return s
}

func personality(total, activeDays int, hours, weekdays []int, added, deleted, fix int) Personality {
	night := sum(hours[22:]) + sum(hours[:4])
	early := sum(hours[5:9])
	weekend := weekdays[5] + weekdays[6]
	var id string
	switch {
	case total == 0:
		id = "ghost"
	case night*100 >= total*30:
		id = "night_owl"
	case early*100 >= total*25:
		id = "early_bird"
	case weekend*100 >= total*30:
		id = "weekend_warrior"
	case deleted > added:
		id = "code_janitor"
	case total >= activeDays*8:
		id = "commit_machine"
	case fix*100 >= total*30:
		id = "bug_hunter"
	default:
		id = "steady_shipper"
	}
	p := personalities[id]
	return Personality{id, p[0], p[1]}
}
