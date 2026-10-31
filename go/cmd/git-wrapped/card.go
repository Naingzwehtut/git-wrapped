package main

// 1200x630 SVG share card. Must stay byte-identical to the Python and JS versions.

import (
	"fmt"
	"strings"
)

const repoURL = "github.com/Naingzwehtut/git-wrapped"

var escaper = strings.NewReplacer("&", "&amp;", "<", "&lt;", ">", "&gt;", "\"", "&quot;", "'", "&#39;")

func esc(s string) string { return escaper.Replace(s) }

func renderSVG(s Stats) string {
	var o []string
	a := func(format string, args ...any) { o = append(o, fmt.Sprintf(format, args...)) }

	a(`<svg xmlns="http://www.w3.org/2000/svg" width="1200" height="630" viewBox="0 0 1200 630">`)
	a(`<defs>`)
	a(`<linearGradient id="bg" x1="0" y1="0" x2="1" y2="1"><stop offset="0" stop-color="#0b0f1a"/><stop offset="1" stop-color="#26123f"/></linearGradient>`)
	a(`<linearGradient id="accent" x1="0" y1="0" x2="1" y2="0"><stop offset="0" stop-color="#ff5fa2"/><stop offset="1" stop-color="#ffb347"/></linearGradient>`)
	a(`</defs>`)
	a(`<rect width="1200" height="630" fill="url(#bg)"/>`)
	a(`<circle cx="1110" cy="70" r="230" fill="#ff5fa2" opacity="0.10"/>`)
	a(`<circle cx="1180" cy="610" r="170" fill="#7c5cff" opacity="0.16"/>`)
	a(`<g font-family="Inter, 'Segoe UI', Helvetica, Arial, sans-serif">`)

	// Left column: title and stat tiles
	a(`<text x="64" y="88" fill="#ffb347" font-size="22" font-weight="700" letter-spacing="6">GIT WRAPPED</text>`)
	a(`<text x="60" y="196" fill="url(#accent)" font-size="120" font-weight="800">%d</text>`, s.Year)
	author := s.Author
	if author == "" {
		author = "nobody"
	}
	a(`<text x="64" y="250" fill="#ffffff" font-size="34" font-weight="700">%s</text>`, esc(truncateEnd(author, 26)))
	where := truncateEnd(s.Repos[0], 40)
	if len(s.Repos) > 1 {
		where = plural(len(s.Repos), "repo")
	}
	a(`<text x="64" y="286" fill="#9aa4b2" font-size="22">%s</text>`, esc(where))

	tiles := [][2]string{
		{fmtInt(s.Commits), "commits"},
		{fmtInt(s.ActiveDays), "active days"},
		{fmtInt(s.LongestStreak.Days), "day streak"},
		{"+" + fmtInt(s.LinesAdded), "lines added"},
	}
	for i, t := range tiles {
		x := 64 + (i%2)*290
		y := 326 + (i/2)*128
		a(`<rect x="%d" y="%d" width="270" height="110" rx="18" fill="#ffffff" fill-opacity="0.06"/>`, x, y)
		a(`<text x="%d" y="%d" fill="#ffffff" font-size="44" font-weight="800">%s</text>`, x+24, y+58, esc(t[0]))
		a(`<text x="%d" y="%d" fill="#9aa4b2" font-size="18">%s</text>`, x+24, y+90, esc(t[1]))
	}

	// Right column: personality, hours chart, languages
	a(`<text x="680" y="88" fill="#9aa4b2" font-size="18" font-weight="700" letter-spacing="4">YOU ARE</text>`)
	a(`<text x="680" y="140" fill="url(#accent)" font-size="38" font-weight="800">%s</text>`, esc(s.Personality.Title))
	a(`<text x="680" y="176" fill="#c9d1d9" font-size="20">%s</text>`, esc(s.Personality.Description))

	a(`<text x="680" y="238" fill="#9aa4b2" font-size="16" font-weight="700" letter-spacing="3">WHEN YOU CODE</text>`)
	mx := s.Hours[argmax(s.Hours)]
	base := 350
	for h, n := range s.Hours {
		x := 680 + h*19
		if n == 0 || mx == 0 {
			a(`<rect x="%d" y="%d" width="13" height="3" rx="1.5" fill="#ffffff" fill-opacity="0.18"/>`, x, base-3)
		} else {
			height := max(10, n*90/mx)
			a(`<rect x="%d" y="%d" width="13" height="%d" rx="4" fill="url(#accent)"/>`, x, base-height, height)
		}
	}
	for _, l := range []struct {
		h     int
		label string
	}{{0, "12am"}, {6, "6am"}, {12, "12pm"}, {18, "6pm"}} {
		a(`<text x="%d" y="374" fill="#6e7681" font-size="14">%s</text>`, 680+l.h*19, l.label)
	}

	a(`<text x="680" y="430" fill="#9aa4b2" font-size="16" font-weight="700" letter-spacing="3">TOP LANGUAGES</text>`)
	langs := s.TopLanguages
	if len(langs) > 3 {
		langs = langs[:3]
	}
	if len(langs) == 0 {
		a(`<text x="680" y="468" fill="#6e7681" font-size="20">No code detected</text>`)
	}
	for i, l := range langs {
		y := 468 + i*40
		width := max(4, l.Lines*210/langs[0].Lines)
		a(`<text x="680" y="%d" fill="#ffffff" font-size="20" font-weight="600">%s</text>`, y, esc(truncateEnd(l.Name, 12)))
		a(`<rect x="840" y="%d" width="210" height="12" rx="6" fill="#ffffff" fill-opacity="0.10"/>`, y-13)
		a(`<rect x="840" y="%d" width="%d" height="12" rx="6" fill="url(#accent)"/>`, y-13, width)
		a(`<text x="1136" y="%d" fill="#9aa4b2" font-size="18" text-anchor="end">%d%%</text>`, y, l.Percent)
	}

	a(`<text x="64" y="604" fill="#6e7681" font-size="18">Get yours → %s</text>`, repoURL)
	a(`</g>`)
	a(`</svg>`)
	return strings.Join(o, "\n") + "\n"
}
