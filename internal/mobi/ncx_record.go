package mobi

import (
	"fmt"
	"html"
	"strings"
)

// NCXRecordConfig holds configuration for generating an NCX record HTML.
type NCXRecordConfig struct {
	Title   string
	Entries []NCXEntry
	Guide   []GuideReference
}

// NCXEntry represents a single TOC entry with a label, file position, and optional children.
type NCXEntry struct {
	Label    string
	FilePos  uint32
	Children []NCXEntry
}

// GuideReference represents a guide reference element in the NCX record.
type GuideReference struct {
	Type    string
	Title   string
	FilePos uint32
}

// GenerateNCXRecord generates the NCX record HTML used in AZW3 files.
func GenerateNCXRecord(cfg NCXRecordConfig) []byte {
	var b strings.Builder

	b.WriteString("<html>")
	b.WriteString("<head>")

	if len(cfg.Guide) > 0 {
		b.WriteString("<guide>")
		for _, ref := range cfg.Guide {
			fmt.Fprintf(&b, `<reference type="%s" title="%s" filepos="%08d"/>`,
				html.EscapeString(ref.Type),
				html.EscapeString(ref.Title),
				ref.FilePos)
		}
		b.WriteString("</guide>")
	}

	b.WriteString("</head>")
	b.WriteString("<body>")
	fmt.Fprintf(&b, "<h1>%s</h1>", html.EscapeString(cfg.Title))

	if len(cfg.Entries) > 0 {
		writeNCXEntries(&b, cfg.Entries)
	}

	b.WriteString("</body>")
	b.WriteString("</html>")

	return []byte(b.String())
}

// writeNCXEntries recursively writes NCX entries as nested <ul>/<li> HTML.
func writeNCXEntries(b *strings.Builder, entries []NCXEntry) {
	b.WriteString("<ul>")
	for _, e := range entries {
		b.WriteString("<li>")
		fmt.Fprintf(b, `<a filepos="%08d">%s</a>`,
			e.FilePos, html.EscapeString(e.Label))
		if len(e.Children) > 0 {
			writeNCXEntries(b, e.Children)
		}
		b.WriteString("</li>")
	}
	b.WriteString("</ul>")
}
