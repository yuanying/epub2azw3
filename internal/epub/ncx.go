package epub

import (
	"bytes"
	"encoding/xml"
	"errors"
	"fmt"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/PuerkitoBio/goquery"
)

// NCX represents the parsed navigation control structure from NCX or NAV document.
type NCX struct {
	UID       string
	Depth     int
	DocTitle  string
	NavPoints []NavPoint
}

// NavPoint represents a single navigation point in the table of contents.
type NavPoint struct {
	ID          string
	PlayOrder   int
	Label       string
	ContentPath string // fragment-free, absolute path within EPUB
	Fragment    string // fragment identifier (without #)
	Children    []NavPoint
}

// LoadNCX loads and parses the table of contents from an EPUB.
// It prioritizes NCX over NAV. Returns nil, nil if neither exists.
func LoadNCX(reader *EPUBReader, opf *OPF) (*NCX, error) {
	// Try NCX first
	if opf.NCXPath != "" {
		data, err := reader.ReadFile(opf.NCXPath)
		if err == nil {
			ncxDir := filepath.ToSlash(filepath.Dir(opf.NCXPath))
			return parseNCX(data, ncxDir)
		}
		if !errors.Is(err, ErrFileNotFound) {
			return nil, fmt.Errorf("failed to read NCX file %s: %w", opf.NCXPath, err)
		}
	}

	// Fallback to NAV
	navPath, ok := findNAVPath(opf)
	if !ok {
		return nil, nil
	}

	data, err := reader.ReadFile(navPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read NAV file %s: %w", navPath, err)
	}

	navDir := filepath.ToSlash(filepath.Dir(navPath))
	return parseNAV(data, navDir)
}

// XML structures for NCX parsing.
type ncxXML struct {
	XMLName  xml.Name    `xml:"ncx"`
	Head     ncxHead     `xml:"head"`
	DocTitle ncxDocTitle `xml:"docTitle"`
	NavMap   ncxNavMap   `xml:"navMap"`
}

type ncxHead struct {
	Metas []ncxMeta `xml:"meta"`
}

type ncxMeta struct {
	Name    string `xml:"name,attr"`
	Content string `xml:"content,attr"`
}

type ncxDocTitle struct {
	Text string `xml:"text"`
}

type ncxNavMap struct {
	NavPoints []ncxNavPoint `xml:"navPoint"`
}

type ncxNavPoint struct {
	ID        string        `xml:"id,attr"`
	PlayOrder string        `xml:"playOrder,attr"`
	NavLabel  ncxNavLabel   `xml:"navLabel"`
	Content   ncxContent    `xml:"content"`
	Children  []ncxNavPoint `xml:"navPoint"`
}

type ncxNavLabel struct {
	Text string `xml:"text"`
}

type ncxContent struct {
	Src string `xml:"src,attr"`
}

// parseNCX parses NCX XML data and returns an NCX structure.
func parseNCX(data []byte, ncxDir string) (*NCX, error) {
	var raw ncxXML
	if err := xml.Unmarshal(data, &raw); err != nil {
		return nil, err
	}

	ncx := &NCX{
		DocTitle:  raw.DocTitle.Text,
		NavPoints: convertNCXNavPoints(raw.NavMap.NavPoints, ncxDir),
	}

	for _, m := range raw.Head.Metas {
		switch m.Name {
		case "dtb:uid":
			ncx.UID = m.Content
		case "dtb:depth":
			if d, err := strconv.Atoi(m.Content); err == nil {
				ncx.Depth = d
			}
		}
	}

	return ncx, nil
}

// convertNCXNavPoints recursively converts XML navPoints to NavPoint structs.
func convertNCXNavPoints(xmlPoints []ncxNavPoint, ncxDir string) []NavPoint {
	points := make([]NavPoint, 0, len(xmlPoints))
	for _, xp := range xmlPoints {
		path, fragment := splitFragment(xp.Content.Src)
		contentPath := ""
		if path != "" {
			contentPath = resolvePath(ncxDir, path)
		}

		playOrder := 0
		if xp.PlayOrder != "" {
			if po, err := strconv.Atoi(xp.PlayOrder); err == nil {
				playOrder = po
			}
		}

		np := NavPoint{
			ID:          xp.ID,
			PlayOrder:   playOrder,
			Label:       xp.NavLabel.Text,
			ContentPath: contentPath,
			Fragment:    fragment,
			Children:    convertNCXNavPoints(xp.Children, ncxDir),
		}
		points = append(points, np)
	}
	return points
}

// findNAVPath searches the OPF manifest for an item with the "nav" property.
func findNAVPath(opf *OPF) (string, bool) {
	for _, item := range opf.Manifest {
		for _, prop := range item.Properties {
			if prop == "nav" {
				return item.Href, true
			}
		}
	}
	return "", false
}

// hasEpubTypeTOC checks if the epub:type attribute value contains "toc" token.
func hasEpubTypeTOC(value string) bool {
	for _, token := range strings.Fields(value) {
		if token == "toc" {
			return true
		}
	}
	return false
}

// parseNAV parses an EPUB 3 NAV document and returns an NCX-compatible structure.
func parseNAV(data []byte, navDir string) (*NCX, error) {
	doc, err := goquery.NewDocumentFromReader(bytes.NewReader(data))
	if err != nil {
		return nil, fmt.Errorf("failed to parse NAV document: %w", err)
	}

	ncx := &NCX{}

	// Find first nav element with epub:type containing "toc"
	doc.Find("nav").EachWithBreak(func(_ int, s *goquery.Selection) bool {
		epubType, _ := s.Attr("epub:type")
		if !hasEpubTypeTOC(epubType) {
			return true // continue
		}

		counter := 0
		ol := s.Find("ol").First()
		ncx.NavPoints = parseNAVList(ol, navDir, &counter)
		return false // break after first toc
	})

	return ncx, nil
}

// parseNAVList recursively parses an ol/li/a structure into NavPoints.
func parseNAVList(ol *goquery.Selection, navDir string, counter *int) []NavPoint {
	var points []NavPoint

	ol.Children().Filter("li").Each(func(_ int, li *goquery.Selection) {
		// Search for <a> in li's children excluding nested <ol>
		a := li.Children().Not("ol").Find("a").First()
		if a.Length() == 0 {
			// Also check direct child <a>
			a = li.Children().Filter("a").First()
		}

		nestedOL := li.Children().Filter("ol").First()

		if a.Length() == 0 {
			// No link found — extract label from li text excluding nested ol
			label := strings.TrimSpace(cloneWithoutOL(li).Text())
			if label != "" || nestedOL.Length() > 0 {
				*counter++
				np := NavPoint{
					ID:        fmt.Sprintf("nav-%d", *counter),
					PlayOrder: *counter,
					Label:     label,
				}
				if nestedOL.Length() > 0 {
					np.Children = parseNAVList(nestedOL, navDir, counter)
				}
				points = append(points, np)
			}
			return
		}

		*counter++

		href, _ := a.Attr("href")
		path, fragment := splitFragment(href)
		contentPath := ""
		if path != "" {
			contentPath = resolvePath(navDir, path)
		}

		np := NavPoint{
			ID:          fmt.Sprintf("nav-%d", *counter),
			PlayOrder:   *counter,
			Label:       strings.TrimSpace(a.Text()),
			ContentPath: contentPath,
			Fragment:    fragment,
		}

		if nestedOL.Length() > 0 {
			np.Children = parseNAVList(nestedOL, navDir, counter)
		}

		points = append(points, np)
	})

	return points
}

// cloneWithoutOL creates a clone of the selection with nested ol elements removed.
func cloneWithoutOL(li *goquery.Selection) *goquery.Selection {
	clone := li.Clone()
	clone.Find("ol").Remove()
	return clone
}

// splitFragment splits a source path into the path and fragment identifier.
func splitFragment(src string) (path, fragment string) {
	if src == "" {
		return "", ""
	}
	parts := strings.SplitN(src, "#", 2)
	path = parts[0]
	if len(parts) == 2 {
		fragment = parts[1]
	}
	return path, fragment
}
