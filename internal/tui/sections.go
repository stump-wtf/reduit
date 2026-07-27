package tui

import "github.com/joestump/reduit/internal/tui/styles"

// sectionID identifies a top-level TUI destination. The foundation ships the
// menu shell and routes to placeholder bodies for each; the real views land in
// #169 (search) and #170 (insights).
type sectionID int

const (
	secSearch sectionID = iota
	secAttachments
	secContacts
	secMetadata
	secStats
)

// sectionMeta describes one menu entry: its title and its glyph (from the
// active glyph set, base or Nerd).
type sectionMeta struct {
	id    sectionID
	title string
	glyph func(styles.Glyphs) string
}

// sections is the ordered menu. The order is the reading order of the product:
// search first (the primary act), then the derived-insight views.
var sections = []sectionMeta{
	{
		id:    secSearch,
		title: "Search",
		glyph: func(g styles.Glyphs) string { return g.Search },
	},
	{
		id:    secAttachments,
		title: "Attachments",
		glyph: func(g styles.Glyphs) string { return g.Attach },
	},
	{
		id:    secContacts,
		title: "Contact Facts",
		glyph: func(g styles.Glyphs) string { return g.Contact },
	},
	{
		id:    secMetadata,
		title: "Metadata",
		glyph: func(g styles.Glyphs) string { return g.Meta },
	},
	{
		id:    secStats,
		title: "Stats",
		glyph: func(g styles.Glyphs) string { return g.Stats },
	},
}
