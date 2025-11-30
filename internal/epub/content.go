package epub

import (
	"bytes"
	"fmt"
	"path/filepath"

	"github.com/PuerkitoBio/goquery"
)

// Content represents a parsed XHTML content file
type Content struct {
	ID        string            // Manifest ID
	Path      string            // File path
	Document  *goquery.Document // Parsed HTML document
	CSSLinks  []string          // Referenced CSS file paths
	ImageRefs []string          // Referenced image paths
}

// LoadContent loads and parses an XHTML content file
// id: manifest item ID
// path: file path within EPUB (used for relative path resolution)
// content: XHTML file content
func LoadContent(id, path string, content []byte) (*Content, error) {
	// Parse XHTML using goquery
	doc, err := goquery.NewDocumentFromReader(bytes.NewReader(content))
	if err != nil {
		return nil, fmt.Errorf("failed to parse XHTML: %w", err)
	}

	c := &Content{
		ID:        id,
		Path:      path,
		Document:  doc,
		CSSLinks:  []string{},
		ImageRefs: []string{},
	}

	// Get base directory for resolving relative paths
	baseDir := filepath.Dir(path)

	// Collect CSS links
	doc.Find("link[rel='stylesheet']").Each(func(i int, s *goquery.Selection) {
		if href, exists := s.Attr("href"); exists {
			resolved := resolvePath(baseDir, href)
			c.CSSLinks = append(c.CSSLinks, resolved)
		}
	})

	// Collect image references
	doc.Find("img").Each(func(i int, s *goquery.Selection) {
		if src, exists := s.Attr("src"); exists {
			resolved := resolvePath(baseDir, src)
			c.ImageRefs = append(c.ImageRefs, resolved)
		}
	})

	return c, nil
}

// resolvePath resolves a relative path against a base directory
// baseDir: base directory (e.g., "text" for "text/chapter1.xhtml")
// relPath: relative path (e.g., "../images/photo.jpg")
// returns: resolved path (e.g., "images/photo.jpg")
func resolvePath(baseDir, relPath string) string {
	// Join base directory with relative path
	joined := filepath.Join(baseDir, relPath)

	// Clean the path to resolve .. and . segments
	cleaned := filepath.Clean(joined)

	// Normalize path separators to forward slashes (EPUB standard)
	normalized := filepath.ToSlash(cleaned)

	return normalized
}
