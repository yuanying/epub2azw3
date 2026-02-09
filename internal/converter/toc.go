package converter

import (
	"bytes"
	"fmt"
	"html"
	"log"
	"net/url"
	"path/filepath"
	"strings"

	"github.com/yuanying/epub2azw3/internal/epub"
)

// TOCEntry represents a resolved TOC entry with a byte offset in the final HTML.
type TOCEntry struct {
	Label    string
	FilePos  uint32
	Children []TOCEntry
}

// TOCGenerator generates inline HTML TOC and resolves filepos offsets.
type TOCGenerator struct {
	ncx        *epub.NCX
	chapterIDs map[string]string
}

// NewTOCGenerator creates a new TOCGenerator.
func NewTOCGenerator(ncx *epub.NCX, chapterIDs map[string]string) *TOCGenerator {
	return &TOCGenerator{
		ncx:        ncx,
		chapterIDs: chapterIDs,
	}
}

// GenerateInlineTOC generates an inline HTML TOC from the NCX data.
// Returns an empty string if there are no NavPoints.
func (g *TOCGenerator) GenerateInlineTOC() string {
	if g.ncx == nil || len(g.ncx.NavPoints) == 0 {
		return ""
	}

	title := g.ncx.DocTitle
	if title == "" {
		title = "Table of Contents"
	}

	var b strings.Builder
	b.WriteString(`<div id="toc">`)
	fmt.Fprintf(&b, "<h1>%s</h1>", html.EscapeString(title))
	g.writeInlineTOCEntries(&b, g.ncx.NavPoints)
	b.WriteString("</div>")

	return b.String()
}

// writeInlineTOCEntries recursively writes NavPoints as nested <ul>/<li> with links.
func (g *TOCGenerator) writeInlineTOCEntries(b *strings.Builder, points []epub.NavPoint) {
	b.WriteString("<ul>")
	for _, np := range points {
		b.WriteString("<li>")
		href := g.resolveHref(np.ContentPath, np.Fragment)
		fmt.Fprintf(b, `<a href="%s">%s</a>`, html.EscapeString(href), html.EscapeString(np.Label))
		if len(np.Children) > 0 {
			g.writeInlineTOCEntries(b, np.Children)
		}
		b.WriteString("</li>")
	}
	b.WriteString("</ul>")
}

// normalizePath cleans and slash-normalizes a path for consistent map lookups.
func normalizePath(p string) string {
	return filepath.ToSlash(filepath.Clean(p))
}

// resolveHref builds an internal href (#chapterID or #chapterID-fragment) from a NavPoint.
func (g *TOCGenerator) resolveHref(contentPath, fragment string) string {
	chapterID, ok := g.chapterIDs[normalizePath(contentPath)]
	if !ok {
		return "#"
	}
	if fragment == "" {
		return "#" + chapterID
	}
	sanitized := sanitizeFragmentForHTMLID(fragment)
	return "#" + chapterID + "-" + sanitized
}

// InsertInlineTOC inserts the inline TOC right after the opening <body> tag.
// Handles both <body> and <body ...attributes...> forms.
// If there are no NavPoints, returns the HTML unchanged.
func (g *TOCGenerator) InsertInlineTOC(htmlStr string) string {
	toc := g.GenerateInlineTOC()
	if toc == "" {
		return htmlStr
	}

	// Find <body> or <body ...> tag
	bodyIdx := strings.Index(htmlStr, "<body")
	if bodyIdx < 0 {
		return htmlStr
	}
	closeIdx := strings.Index(htmlStr[bodyIdx:], ">")
	if closeIdx < 0 {
		return htmlStr
	}
	insertPos := bodyIdx + closeIdx + 1

	return htmlStr[:insertPos] + toc + htmlStr[insertPos:]
}

// BuildTOCEntries converts NavPoints into TOCEntries with resolved filepos byte offsets.
func (g *TOCGenerator) BuildTOCEntries(finalHTML []byte) ([]TOCEntry, error) {
	if g.ncx == nil || len(g.ncx.NavPoints) == 0 {
		return nil, nil
	}
	return g.buildEntries(finalHTML, g.ncx.NavPoints), nil
}

// buildEntries recursively converts NavPoints to TOCEntries.
// If a fragment cannot be resolved, falls back to the chapter start.
// If the chapter itself cannot be resolved, the entry is skipped.
func (g *TOCGenerator) buildEntries(finalHTML []byte, points []epub.NavPoint) []TOCEntry {
	entries := make([]TOCEntry, 0, len(points))
	for _, np := range points {
		pos, _ := g.calculateFilePos(finalHTML, np.ContentPath, np.Fragment)
		if pos == 0 && np.Fragment != "" {
			// Fallback to chapter start when fragment is not found
			pos, _ = g.calculateFilePos(finalHTML, np.ContentPath, "")
			if pos > 0 {
				log.Printf("warning: fragment %q not found in %q, falling back to chapter start", np.Fragment, np.ContentPath)
			}
		}
		if pos == 0 {
			log.Printf("warning: skipping unresolved TOC entry %q (%s#%s)", np.Label, np.ContentPath, np.Fragment)
			continue
		}
		entry := TOCEntry{
			Label:   np.Label,
			FilePos: pos,
		}
		if len(np.Children) > 0 {
			entry.Children = g.buildEntries(finalHTML, np.Children)
		}
		entries = append(entries, entry)
	}
	return entries
}

// calculateFilePos finds the byte offset of the target element in the final HTML.
// Returns 0 if the target is not found.
func (g *TOCGenerator) calculateFilePos(finalHTML []byte, contentPath, fragment string) (uint32, error) {
	targetID := g.resolveTargetID(contentPath, fragment)
	if targetID == "" {
		return 0, nil
	}

	// Search for id="targetID" in the HTML bytes
	searchPattern := []byte(`id="` + targetID + `"`)
	idx := bytes.Index(finalHTML, searchPattern)
	if idx < 0 {
		return 0, nil
	}

	// Walk backwards to find the '<' that opens this tag
	tagStart := idx
	for tagStart > 0 && finalHTML[tagStart] != '<' {
		tagStart--
	}

	return uint32(tagStart), nil
}

// resolveTargetID constructs the search target ID from content path and fragment.
func (g *TOCGenerator) resolveTargetID(contentPath, fragment string) string {
	chapterID, ok := g.chapterIDs[normalizePath(contentPath)]
	if !ok {
		return ""
	}
	if fragment == "" {
		return chapterID
	}
	sanitized := url.QueryEscape(fragment)
	return chapterID + "-" + sanitized
}
