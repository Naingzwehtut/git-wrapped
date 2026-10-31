"""1200x630 SVG share card. Must stay byte-identical to the Go and JS versions."""

from .render import fmt, plural, truncate_end

REPO_URL = "github.com/Naingzwehtut/git-wrapped"


def esc(s):
    return (s.replace("&", "&amp;").replace("<", "&lt;").replace(">", "&gt;")
            .replace('"', "&quot;").replace("'", "&#39;"))


def render_svg(s):
    o = []
    a = o.append
    a('<svg xmlns="http://www.w3.org/2000/svg" width="1200" height="630" viewBox="0 0 1200 630">')
    a('<defs>')
    a('<linearGradient id="bg" x1="0" y1="0" x2="1" y2="1"><stop offset="0" stop-color="#0b0f1a"/><stop offset="1" stop-color="#26123f"/></linearGradient>')
    a('<linearGradient id="accent" x1="0" y1="0" x2="1" y2="0"><stop offset="0" stop-color="#ff5fa2"/><stop offset="1" stop-color="#ffb347"/></linearGradient>')
    a('</defs>')
    a('<rect width="1200" height="630" fill="url(#bg)"/>')
    a('<circle cx="1110" cy="70" r="230" fill="#ff5fa2" opacity="0.10"/>')
    a('<circle cx="1180" cy="610" r="170" fill="#7c5cff" opacity="0.16"/>')
    a('<g font-family="Inter, \'Segoe UI\', Helvetica, Arial, sans-serif">')

    # Left column: title and stat tiles
    a('<text x="64" y="88" fill="#ffb347" font-size="22" font-weight="700" letter-spacing="6">GIT WRAPPED</text>')
    a(f'<text x="60" y="196" fill="url(#accent)" font-size="120" font-weight="800">{s["year"]}</text>')
    a(f'<text x="64" y="250" fill="#ffffff" font-size="34" font-weight="700">{esc(truncate_end(s["author"] or "nobody", 26))}</text>')
    repos = s["repos"]
    where = plural(len(repos), "repo") if len(repos) > 1 else truncate_end(repos[0], 40)
    a(f'<text x="64" y="286" fill="#9aa4b2" font-size="22">{esc(where)}</text>')

    tiles = [
        (fmt(s["commits"]), "commits"),
        (fmt(s["active_days"]), "active days"),
        (fmt(s["longest_streak"]["days"]), "day streak"),
        ("+" + fmt(s["lines_added"]), "lines added"),
    ]
    for i, (value, label) in enumerate(tiles):
        x = 64 + (i % 2) * 290
        y = 326 + (i // 2) * 128
        a(f'<rect x="{x}" y="{y}" width="270" height="110" rx="18" fill="#ffffff" fill-opacity="0.06"/>')
        a(f'<text x="{x + 24}" y="{y + 58}" fill="#ffffff" font-size="44" font-weight="800">{esc(value)}</text>')
        a(f'<text x="{x + 24}" y="{y + 90}" fill="#9aa4b2" font-size="18">{esc(label)}</text>')

    # Right column: personality, hours chart, languages
    pers = s["personality"]
    a('<text x="680" y="88" fill="#9aa4b2" font-size="18" font-weight="700" letter-spacing="4">YOU ARE</text>')
    a(f'<text x="680" y="140" fill="url(#accent)" font-size="38" font-weight="800">{esc(pers["title"])}</text>')
    a(f'<text x="680" y="176" fill="#c9d1d9" font-size="20">{esc(pers["description"])}</text>')

    a('<text x="680" y="238" fill="#9aa4b2" font-size="16" font-weight="700" letter-spacing="3">WHEN YOU CODE</text>')
    hours = s["hours"]
    mx = max(hours)
    base = 350
    for h, n in enumerate(hours):
        x = 680 + h * 19
        if n == 0 or mx == 0:
            a(f'<rect x="{x}" y="{base - 3}" width="13" height="3" rx="1.5" fill="#ffffff" fill-opacity="0.18"/>')
        else:
            height = max(10, n * 90 // mx)
            a(f'<rect x="{x}" y="{base - height}" width="13" height="{height}" rx="4" fill="url(#accent)"/>')
    for h, label in ((0, "12am"), (6, "6am"), (12, "12pm"), (18, "6pm")):
        a(f'<text x="{680 + h * 19}" y="374" fill="#6e7681" font-size="14">{label}</text>')

    a('<text x="680" y="430" fill="#9aa4b2" font-size="16" font-weight="700" letter-spacing="3">TOP LANGUAGES</text>')
    langs = s["top_languages"][:3]
    if not langs:
        a('<text x="680" y="468" fill="#6e7681" font-size="20">No code detected</text>')
    for i, lang in enumerate(langs):
        y = 468 + i * 40
        width = max(4, lang["lines"] * 210 // langs[0]["lines"])
        a(f'<text x="680" y="{y}" fill="#ffffff" font-size="20" font-weight="600">{esc(truncate_end(lang["name"], 12))}</text>')
        a(f'<rect x="840" y="{y - 13}" width="210" height="12" rx="6" fill="#ffffff" fill-opacity="0.10"/>')
        a(f'<rect x="840" y="{y - 13}" width="{width}" height="12" rx="6" fill="url(#accent)"/>')
        a(f'<text x="1136" y="{y}" fill="#9aa4b2" font-size="18" text-anchor="end">{lang["percent"]}%</text>')

    a(f'<text x="64" y="604" fill="#6e7681" font-size="18">Get yours → {REPO_URL}</text>')
    a('</g>')
    a('</svg>')
    return "\n".join(o) + "\n"
