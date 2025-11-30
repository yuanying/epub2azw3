package epub

// OPF represents the parsed Open Package Format document
type OPF struct {
	Metadata Metadata
	Manifest map[string]ManifestItem // id -> item
	Spine    []SpineItem
	NCXPath  string
}

// Metadata represents the metadata section of the OPF
type Metadata struct {
	Title       string
	Creators    []Creator
	Language    string
	Identifier  string
	Publisher   string
	Date        string
	Description string
	Subjects    []string
	Rights      string
	CoverID     string // EPUB 2.0 cover image manifest item ID (from meta name="cover")
}

// Creator represents a creator (author, editor, etc.) of the book
type Creator struct {
	Name string
	Role string // e.g., "aut" for author, "edt" for editor
	Lang string // xml:lang attribute
}

// ManifestItem represents an item in the manifest
type ManifestItem struct {
	ID         string
	Href       string
	MediaType  string
	Properties []string
}

// SpineItem represents an item reference in the spine
type SpineItem struct {
	IDRef  string
	Linear bool
}
