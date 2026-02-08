package epub

import (
	"encoding/xml"
	"strconv"
	"strings"
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

// XML structures for NCX parsing.
type ncxXML struct {
	XMLName  xml.Name       `xml:"ncx"`
	Head     ncxHead        `xml:"head"`
	DocTitle ncxDocTitle    `xml:"docTitle"`
	NavMap   ncxNavMap      `xml:"navMap"`
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
	ID        string         `xml:"id,attr"`
	PlayOrder string         `xml:"playOrder,attr"`
	NavLabel  ncxNavLabel    `xml:"navLabel"`
	Content   ncxContent     `xml:"content"`
	Children  []ncxNavPoint  `xml:"navPoint"`
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
