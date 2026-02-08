package epub

import (
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
