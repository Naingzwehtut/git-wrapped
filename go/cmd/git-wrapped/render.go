package main

// Terminal output. Must stay byte-identical to the Python and JS versions (see SPEC.md).

import (
	"fmt"
	"strconv"
	"strings"
	"unicode/utf8"
)

var months = []string{"Jan", "Feb", "Mar", "Apr", "May", "Jun", "Jul", "Aug", "Sep", "Oct", "Nov", "Dec"}
var shortWeekdays = []string{"Mon", "Tue", "Wed", "Thu", "Fri", "Sat", "Sun"}
var sparkChars = []rune("▁▂▃▄▅▆▇█")

func fmtInt(n int) string {
	s := strconv.Itoa(n)
	neg := strings.HasPrefix(s, "-")
	if neg {
		s = s[1:]
	}
	var b strings.Builder
	for i, r := range s {
		if i > 0 && (len(s)-i)%3 == 0 {
			b.WriteByte(',')
		}
		b.WriteRune(r)
	}
	if neg {
		return "-" + b.String()
	}
	return b.String()
}

func plural(n int, word string) string {
	if n == 1 {
		return fmtInt(n) + " " + word
	}
	return fmtInt(n) + " " + word + "s"
}

func hourLabel(h int) string {
	switch {
	case h == 0:
		return "12 AM"
	case h < 12:
		return fmt.Sprintf("%d AM", h)
	case h == 12:
		return "12 PM"
	}
	return fmt.Sprintf("%d PM", h-12)
}

func shortDate(day string) string {
	return fmt.Sprintf("%s %d", months[atoi(day[5:7])-1], atoi(day[8:10]))
}

func bar(n, mx, width int) string {
	filled := 0
	if mx != 0 {
		filled = n * width / mx
	}
	if n > 0 && filled == 0 {
		filled = 1
	}
	return strings.Repeat("█", filled) + strings.Repeat("░", width-filled)
}

func spark(values []int) string {
	mx := values[argmax(values)]
	var b strings.Builder
	for _, v := range values {
		if v == 0 {
			b.WriteByte(' ')
		} else {
			b.WriteRune(sparkChars[v*7/mx])
		}
	}
	return b.String()
}

func truncateStart(s string, n int) string {
	r := []rune(s)
	if len(r) <= n {
		return s
	}
	return "…" + string(r[len(r)-(n-1):])
}

func truncateEnd(s string, n int) string {
	r := []rune(s)
	if len(r) <= n {
		return s
	}
	return string(r[:n-1]) + "…"
}

func padEnd(s string, n int) string {
	return s + strings.Repeat(" ", max(0, n-utf8.RuneCountInString(s)))
}

func padStart(s string, n int) string {
	return strings.Repeat(" ", max(0, n-utf8.RuneCountInString(s))) + s
}

type palette struct{ enabled bool }

func (p palette) wrap(code, t string) string {
	if !p.enabled {
		return t
	}
	return "\x1b[" + code + "m" + t + "\x1b[0m"
}
func (p palette) bold(t string) string    { return p.wrap("1", t) }
func (p palette) dim(t string) string     { return p.wrap("2", t) }
func (p palette) magenta(t string) string { return p.wrap("35", t) }
func (p palette) cyan(t string) string    { return p.wrap("36", t) }
func (p palette) yellow(t string) string  { return p.wrap("33", t) }

func renderText(s Stats, color bool) string {
	p := palette{color}
	out := []string{""}
	add := func(line string) { out = append(out, line) }

	add("  " + p.bold(p.magenta(fmt.Sprintf("★ GIT WRAPPED %d ★", s.Year))))
	where := s.Repos[0]
	if len(s.Repos) > 1 {
		where = plural(len(s.Repos), "repo")
	}
	author := s.Author
	if author == "" {
		author = "nobody"
	}
	add("  " + p.dim(author+" · "+where))
	add("")

	if s.Commits == 0 {
		add("  No commits found for this year.")
		add("  Try --everyone, --author <name> or --year <year>.")
		add("")
		return strings.Join(out, "\n") + "\n"
	}

	row := func(value, label string) {
		add("  " + p.bold(value) + strings.Repeat(" ", max(1, 20-utf8.RuneCountInString(value))) + p.dim(label))
	}
	st, bd := s.LongestStreak, s.BusiestDay
	row(plural(s.Commits, "commit"), "on "+plural(s.ActiveDays, "active day"))
	row("+"+fmtInt(s.LinesAdded)+" / -"+fmtInt(s.LinesDeleted), "lines added / deleted")
	row(plural(st.Days, "day"), "longest streak ("+shortDate(st.Start)+" – "+shortDate(st.End)+")")
	row(s.BusiestWeekday, "busiest weekday")
	row(hourLabel(s.BusiestHour), "busiest hour")
	row(shortDate(bd.Date), "busiest day ("+plural(bd.Commits, "commit")+")")
	add("")

	add("  " + p.bold("When you code"))
	mx := s.Weekdays[argmax(s.Weekdays)]
	for i, n := range s.Weekdays {
		add("  " + shortWeekdays[i] + " " + p.cyan(bar(n, mx, 24)) + " " + fmtInt(n))
	}
	add("")
	add("  Hours  " + p.cyan(spark(s.Hours)))
	add("         " + p.dim("0     6     12    18"))
	add("  Months " + p.cyan(spark(s.Months)))
	add("         " + p.dim("JFMAMJJASOND"))

	if len(s.TopLanguages) > 0 {
		add("")
		add("  " + p.bold("Top languages"))
		mx = s.TopLanguages[0].Lines
		for _, l := range s.TopLanguages {
			add("  " + padEnd(l.Name, 12) + " " + p.magenta(bar(l.Lines, mx, 20)) + " " + padStart(strconv.Itoa(l.Percent)+"%", 4))
		}
	}

	if len(s.TopFiles) > 0 {
		add("")
		add("  " + p.bold("Most edited files"))
		for i, f := range s.TopFiles {
			add(fmt.Sprintf("  %d. %s ", i+1, truncateStart(f.Path, 50)) + p.dim("("+plural(f.Commits, "commit")+")"))
		}
	}

	lm := s.LongestMessage
	add("")
	add("  " + p.bold("Fun facts"))
	add("  · " + fmtInt(s.FixCommits) + " of " + plural(s.Commits, "commit message") + " mention fix or bug")
	add("  · " + plural(s.FilesTouched, "file") + " touched across " + plural(len(s.Repos), "repo"))
	add("  · Longest commit message: " + plural(lm.Length, "character"))
	add("    " + p.dim("\""+truncateEnd(lm.Subject, 60)+"\""))

	add("")
	add("  You are " + p.bold(p.yellow(strings.ToUpper(s.Personality.Title))))
	add("  " + p.dim(s.Personality.Description))
	add("")
	add("  " + p.dim("Make your share card:  git-wrapped --svg wrapped.svg"))
	add("")
	return strings.Join(out, "\n") + "\n"
}
