package converter

import (
	"fmt"
	"net/url"
	"path/filepath"
	"strings"

	"github.com/PuerkitoBio/goquery"
	"github.com/yuanying/epub2azw3/internal/epub"
)

// HTMLBuilder builds a single integrated HTML document from multiple XHTML files
type HTMLBuilder struct {
	chapters   []*ChapterContent
	cssContent []string
	chapterIDs map[string]string // file path -> chapter ID (e.g., "text/ch01.xhtml" -> "ch01")
}

// ChapterContent represents the content of a single chapter
type ChapterContent struct {
	ID           string            // Chapter ID (e.g., "ch01")
	OriginalPath string            // Original file path in EPUB
	Document     *goquery.Document // Parsed HTML document
}

// NewHTMLBuilder creates a new HTMLBuilder instance
func NewHTMLBuilder() *HTMLBuilder {
	return &HTMLBuilder{
		chapters:   make([]*ChapterContent, 0),
		cssContent: make([]string, 0),
		chapterIDs: make(map[string]string),
	}
}

// AddChapter adds a chapter to the builder
func (h *HTMLBuilder) AddChapter(content *epub.Content) error {
	// Generate chapter ID (ch01, ch02, etc.)
	chapterNum := len(h.chapters) + 1
	chapterID := fmt.Sprintf("ch%02d", chapterNum)

	// Store the mapping from file path to chapter ID
	h.chapterIDs[content.Path] = chapterID

	// Create chapter content
	chapter := &ChapterContent{
		ID:           chapterID,
		OriginalPath: content.Path,
		Document:     content.Document,
	}

	h.chapters = append(h.chapters, chapter)
	return nil
}

// AddCSS adds CSS content to the builder
func (h *HTMLBuilder) AddCSS(css string) {
	h.cssContent = append(h.cssContent, css)
}

// GetChapterID returns the chapter ID for a given file path
func (h *HTMLBuilder) GetChapterID(path string) string {
	return h.chapterIDs[path]
}

// Build generates the integrated HTML document
func (h *HTMLBuilder) Build() (string, error) {
	// Create a new HTML document from a template
	templateHTML := `<html xmlns="http://www.w3.org/1999/xhtml"><head></head><body></body></html>`
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(templateHTML))
	if err != nil {
		return "", fmt.Errorf("failed to create document: %w", err)
	}

	head := doc.Find("head")
	body := doc.Find("body")

	// Add CSS to head
	if len(h.cssContent) > 0 {
		cssText := strings.Join(h.cssContent, "\n")
		// Escape any </style> tags in CSS to prevent breaking the HTML structure
		// Use <\/style> which is safe in CSS context
		cssText = strings.ReplaceAll(cssText, "</style>", "<\\/style>")
		head.AppendHtml(fmt.Sprintf("<style>%s</style>", cssText))
	}

	// Process each chapter
	for _, chapter := range h.chapters {
		// Extract body content from the chapter
		chapterBody := chapter.Document.Find("body")

		// Namespace all element IDs to avoid collisions across chapters
		chapterBody.Find("[id]").Each(func(i int, s *goquery.Selection) {
			origID, exists := s.Attr("id")
			if !exists || origID == "" {
				return
			}
			sanitized := sanitizeFragmentForHTMLID(origID)
			if sanitized == "" {
				return
			}
			s.SetAttr("id", chapter.ID+"-"+sanitized)
		})

		// Build the chapter div HTML
		var chapterHTML strings.Builder
		chapterHTML.WriteString(fmt.Sprintf(`<div id="%s">`, chapter.ID))
		chapterHTML.WriteString("<mbp:pagebreak/>")

		// Copy all children from the chapter body to the chapter div
		var htmlErr error
		chapterBody.Children().Each(func(i int, s *goquery.Selection) {
			html, err := goquery.OuterHtml(s)
			if err != nil {
				htmlErr = fmt.Errorf("failed to get outer HTML for chapter '%s', child %d: %w", chapter.ID, i, err)
				return
			}
			chapterHTML.WriteString(html)
		})

		if htmlErr != nil {
			return "", htmlErr
		}

		chapterHTML.WriteString("</div>")

		// Append the complete chapter div to body
		body.AppendHtml(chapterHTML.String())
	}

	// Resolve links in the integrated document
	h.resolveLinks(body)

	// Get the final HTML
	html, err := doc.Html()
	if err != nil {
		return "", fmt.Errorf("failed to generate HTML: %w", err)
	}

	return html, nil
}

// resolveLinks resolves internal chapter links to fragment identifiers
func (h *HTMLBuilder) resolveLinks(body *goquery.Selection) {
	body.Find("a[href]").Each(func(i int, s *goquery.Selection) {
		href, exists := s.Attr("href")
		if !exists {
			return
		}

		// Parse the URL
		u, err := url.Parse(href)
		if err != nil {
			return
		}

		// Skip absolute URLs (http://, https://, etc.)
		if u.IsAbs() {
			return
		}

		// If it's a fragment-only reference (e.g., #section1), keep it as-is
		if u.Path == "" && u.Fragment != "" {
			chapterID := h.findChapterIDForLink(s)
			if chapterID == "" {
				return
			}

			sanitizedFragment := sanitizeFragmentForHTMLID(u.Fragment)
			if sanitizedFragment == "" {
				return
			}

			s.SetAttr("href", "#"+chapterID+"-"+sanitizedFragment)
			return
		}

		// If it's a relative path (e.g., chapter02.xhtml or chapter02.xhtml#section1)
		if u.Path != "" {
			// Resolve the path to get the target chapter
			targetPath := h.resolveRelativePath(s, u.Path)

			// Get the chapter ID for this path
			chapterID, exists := h.chapterIDs[targetPath]
			if !exists {
				// Try just the filename without directory
				filename := filepath.Base(targetPath)
				for path, id := range h.chapterIDs {
					if filepath.Base(path) == filename {
						chapterID = id
						exists = true
						break
					}
				}
			}

			if exists {
				// Transform the link
				newHref := "#" + chapterID
				if u.Fragment != "" {
					// If there's a fragment, append it with a separator
					// Sanitize the fragment to ensure valid HTML ID references
					sanitizedFragment := sanitizeFragmentForHTMLID(u.Fragment)
					if sanitizedFragment != "" {
						newHref = "#" + chapterID + "-" + sanitizedFragment
					}
				}
				s.SetAttr("href", newHref)
			}
		}
	})
}

// sanitizeFragmentForHTMLID sanitizes a URL fragment for use as an HTML ID
// URL-encodes the fragment to ensure HTML attribute safety while preserving all characters
// This handles quotes, angle brackets, and other special characters that could break HTML attributes
func sanitizeFragmentForHTMLID(fragment string) string {
	if fragment == "" {
		return ""
	}

	// URL-encode to ensure HTML attribute safety
	// This handles quotes, angle brackets, and other special characters
	// while preserving non-ASCII characters in encoded form
	return url.QueryEscape(fragment)
}

	// resolveRelativePath resolves a relative path from a link element
	func (h *HTMLBuilder) resolveRelativePath(link *goquery.Selection, relativePath string) string {
		// Find the chapter this link is in
		chapterPath, _ := h.chapterPathForLink(link)

		if chapterPath == "" {
			return relativePath
		}

	// Resolve relative path against the chapter's directory
	chapterDir := filepath.Dir(chapterPath)
	resolved := filepath.Join(chapterDir, relativePath)
	cleaned := filepath.Clean(resolved)
	normalized := filepath.ToSlash(cleaned)

	return normalized
}

// findChapterIDForLink returns the chapter ID for a link's enclosing chapter div
func (h *HTMLBuilder) findChapterIDForLink(link *goquery.Selection) string {
	_, chapterID := h.chapterPathForLink(link)
	return chapterID
}

// chapterPathForLink finds the original path and chapter ID for the link's enclosing chapter
func (h *HTMLBuilder) chapterPathForLink(link *goquery.Selection) (string, string) {
	chapterDiv := link.Closest("[id^='ch']")
	if chapterDiv.Length() == 0 {
		return "", ""
	}

	chapterID, exists := chapterDiv.Attr("id")
	if !exists {
		return "", ""
	}

	for path, id := range h.chapterIDs {
		if id == chapterID {
			return path, chapterID
		}
	}

	return "", chapterID
}
