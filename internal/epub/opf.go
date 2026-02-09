package epub

import (
	"encoding/xml"
	"fmt"
	"path/filepath"
	"regexp"
	"strings"
)

var opfISBNPattern = regexp.MustCompile(`(?:^|\D)(\d{13}|\d{10})(?:\D|$)`)

// opfPackage represents the OPF XML structure
type opfPackage struct {
	XMLName  xml.Name    `xml:"package"`
	Version  string      `xml:"version,attr"`
	UniqueID string      `xml:"unique-identifier,attr"`
	Metadata opfMetadata `xml:"metadata"`
	Manifest opfManifest `xml:"manifest"`
	Spine    opfSpine    `xml:"spine"`
	Guide    opfGuide    `xml:"guide"`
}

// opfMetadata represents the metadata section
type opfMetadata struct {
	Title       []string        `xml:"http://purl.org/dc/elements/1.1/ title"`
	Creator     []opfCreator    `xml:"http://purl.org/dc/elements/1.1/ creator"`
	Language    []string        `xml:"http://purl.org/dc/elements/1.1/ language"`
	Identifier  []opfIdentifier `xml:"http://purl.org/dc/elements/1.1/ identifier"`
	Publisher   []string        `xml:"http://purl.org/dc/elements/1.1/ publisher"`
	Date        []string        `xml:"http://purl.org/dc/elements/1.1/ date"`
	Description []string        `xml:"http://purl.org/dc/elements/1.1/ description"`
	Subject     []string        `xml:"http://purl.org/dc/elements/1.1/ subject"`
	Rights      []string        `xml:"http://purl.org/dc/elements/1.1/ rights"`
	Meta        []opfMeta       `xml:"meta"`
}

// opfCreator represents a creator element
type opfCreator struct {
	Name string `xml:",chardata"`
	Role string `xml:"http://www.idpf.org/2007/opf role,attr"`
	Lang string `xml:"http://www.w3.org/XML/1998/namespace lang,attr"`
	ID   string `xml:"id,attr"`
}

// opfIdentifier represents an identifier element
type opfIdentifier struct {
	Value     string `xml:",chardata"`
	ID        string `xml:"id,attr"`
	Scheme    string `xml:"scheme,attr"`
	OPFScheme string `xml:"http://www.idpf.org/2007/opf scheme,attr"`
}

// opfMeta represents a meta element (EPUB 2.0 and 3.0)
type opfMeta struct {
	Name     string `xml:"name,attr"`
	Content  string `xml:"content,attr"` // EPUB 2.0: attribute value
	Value    string `xml:",chardata"`    // EPUB 3.0: element text content
	Property string `xml:"property,attr"`
	Refines  string `xml:"refines,attr"`
	Scheme   string `xml:"scheme,attr"`
}

// opfManifest represents the manifest section
type opfManifest struct {
	Items []opfManifestItem `xml:"item"`
}

// opfManifestItem represents an item in the manifest
type opfManifestItem struct {
	ID         string `xml:"id,attr"`
	Href       string `xml:"href,attr"`
	MediaType  string `xml:"media-type,attr"`
	Properties string `xml:"properties,attr"`
}

// opfSpine represents the spine section
type opfSpine struct {
	Toc                      string       `xml:"toc,attr"`
	PageProgressionDirection string       `xml:"page-progression-direction,attr"`
	ItemRefs                 []opfItemRef `xml:"itemref"`
}

// opfItemRef represents an itemref in the spine
type opfItemRef struct {
	IDRef  string `xml:"idref,attr"`
	Linear string `xml:"linear,attr"`
}

// opfGuide represents the guide section (EPUB 2.0).
type opfGuide struct {
	References []opfGuideReference `xml:"reference"`
}

// opfGuideReference represents a reference in the OPF guide.
type opfGuideReference struct {
	Type  string `xml:"type,attr"`
	Title string `xml:"title,attr"`
	Href  string `xml:"href,attr"`
}

// ParseOPF parses an OPF file content and returns the OPF structure
// opfDir is the directory containing the OPF file (e.g., "OEBPS/")
func ParseOPF(content []byte, opfDir string) (*OPF, error) {
	var pkg opfPackage
	if err := xml.Unmarshal(content, &pkg); err != nil {
		return nil, fmt.Errorf("failed to parse OPF XML: %w", err)
	}

	opf := &OPF{
		Manifest: make(map[string]ManifestItem),
	}

	// Parse metadata
	opf.Metadata = parseMetadata(&pkg.Metadata, pkg.UniqueID)

	// Parse manifest
	for _, item := range pkg.Manifest.Items {
		manifestItem := ManifestItem{
			ID:        item.ID,
			Href:      joinPath(opfDir, item.Href),
			MediaType: item.MediaType,
		}

		// Parse properties (space-separated)
		if item.Properties != "" {
			manifestItem.Properties = strings.Fields(item.Properties)
		}

		opf.Manifest[item.ID] = manifestItem
		opf.ManifestOrder = append(opf.ManifestOrder, item.ID)
	}

	// Parse spine
	for _, itemRef := range pkg.Spine.ItemRefs {
		linear := true
		if itemRef.Linear == "no" {
			linear = false
		}

		opf.Spine = append(opf.Spine, SpineItem{
			IDRef:  itemRef.IDRef,
			Linear: linear,
		})
	}

	// Parse page-progression-direction
	opf.PageProgressionDirection = pkg.Spine.PageProgressionDirection

	// Resolve NCX path from toc attribute
	if pkg.Spine.Toc != "" {
		if ncxItem, ok := opf.Manifest[pkg.Spine.Toc]; ok {
			opf.NCXPath = ncxItem.Href
		}
	}

	// Parse guide references
	for _, ref := range pkg.Guide.References {
		opf.Guide = append(opf.Guide, GuideReference{
			Type:  ref.Type,
			Title: ref.Title,
			Href:  joinPathWithFragment(opfDir, ref.Href),
		})
	}

	return opf, nil
}

// parseMetadata parses the metadata section
func parseMetadata(meta *opfMetadata, uniqueID string) Metadata {
	md := Metadata{
		Subjects: []string{},
		Creators: []Creator{},
	}

	// Title (use first one)
	if len(meta.Title) > 0 {
		md.Title = meta.Title[0]
	}

	// Language (use first one)
	if len(meta.Language) > 0 {
		md.Language = meta.Language[0]
	}

	md.Identifier = selectIdentifier(meta.Identifier, uniqueID)

	// Publisher (use first one)
	if len(meta.Publisher) > 0 {
		md.Publisher = meta.Publisher[0]
	}

	// Date (use first one)
	if len(meta.Date) > 0 {
		md.Date = meta.Date[0]
	}

	// Description (use first one)
	if len(meta.Description) > 0 {
		md.Description = meta.Description[0]
	}

	// Subjects (all)
	md.Subjects = meta.Subject

	// Rights (use first one)
	if len(meta.Rights) > 0 {
		md.Rights = meta.Rights[0]
	}

	// Creators
	for _, creator := range meta.Creator {
		md.Creators = append(md.Creators, Creator{
			Name: creator.Name,
			Role: creator.Role,
			Lang: creator.Lang,
		})
	}

	// Process EPUB 3.0 meta elements for creator roles
	processCreatorRoles(&md, meta)

	// Process EPUB 2.0 cover meta element
	for _, m := range meta.Meta {
		if m.Name == "cover" && m.Content != "" {
			md.CoverID = m.Content
			break
		}
	}

	return md
}

// processCreatorRoles processes EPUB 3.0 meta elements to refine creator roles
func processCreatorRoles(md *Metadata, meta *opfMetadata) {
	// Build a map of creator IDs to indices
	creatorMap := make(map[string]int)
	for i, creator := range md.Creators {
		// Find the corresponding creator in the original metadata
		for _, origCreator := range meta.Creator {
			if origCreator.Name == creator.Name && origCreator.ID != "" {
				creatorMap["#"+origCreator.ID] = i
				break
			}
		}
	}

	// Process meta elements that refine creators
	for _, m := range meta.Meta {
		if m.Property == "role" && m.Refines != "" {
			if idx, ok := creatorMap[m.Refines]; ok {
				// EPUB 3.0 uses chardata (Value), EPUB 2.0 uses content attribute (Content)
				if m.Value != "" {
					md.Creators[idx].Role = m.Value
				} else {
					md.Creators[idx].Role = m.Content
				}
			}
		}
	}
}

func selectIdentifier(identifiers []opfIdentifier, uniqueID string) string {
	trimmedUniqueID := strings.TrimSpace(uniqueID)

	// 1) Prefer explicit ISBN scheme.
	for _, id := range identifiers {
		value := strings.TrimSpace(id.Value)
		if value == "" {
			continue
		}
		scheme := identifierScheme(id)
		if strings.EqualFold(strings.TrimSpace(scheme), "isbn") {
			return value
		}
	}

	// 2) Then prefer identifier values that contain an ISBN token.
	for _, id := range identifiers {
		value := strings.TrimSpace(id.Value)
		if value == "" {
			continue
		}
		if hasISBN(value) {
			return value
		}
	}

	// 3) Fallback to unique-identifier.
	if trimmedUniqueID != "" {
		for _, id := range identifiers {
			if strings.TrimSpace(id.ID) == trimmedUniqueID {
				if value := strings.TrimSpace(id.Value); value != "" {
					return value
				}
			}
		}
	}

	// 4) Last fallback: first non-empty identifier.
	for _, id := range identifiers {
		if value := strings.TrimSpace(id.Value); value != "" {
			return value
		}
	}

	return ""
}

func identifierScheme(id opfIdentifier) string {
	if id.OPFScheme != "" {
		return id.OPFScheme
	}
	return id.Scheme
}

func hasISBN(value string) bool {
	stripped := strings.ReplaceAll(value, "-", "")
	return opfISBNPattern.MatchString(stripped)
}

// joinPath joins OPF directory with a relative path
// Always returns forward-slash separated paths (EPUB standard)
func joinPath(base, rel string) string {
	if base == "" {
		return rel
	}
	return filepath.ToSlash(filepath.Join(base, rel))
}

func joinPathWithFragment(base, rel string) string {
	if rel == "" {
		return ""
	}
	pathPart, frag, hasFrag := strings.Cut(rel, "#")
	if pathPart == "" {
		return rel
	}
	joined := joinPath(base, pathPart)
	if hasFrag {
		return joined + "#" + frag
	}
	return joined
}

// FindCoverImage finds a cover image using OPF-only lightweight detection.
// It does not parse XHTML files; that richer detection lives in converter layer.
func (opf *OPF) FindCoverImage() (string, bool) {
	if opf == nil {
		return "", false
	}

	orderedItems := orderedManifestItems(opf)

	// Method 1: EPUB 3.0 - check for cover-image property
	for _, item := range orderedItems {
		if !isImageMediaType(item.MediaType) {
			continue
		}
		for _, prop := range item.Properties {
			if strings.EqualFold(prop, "cover-image") {
				return item.Href, true
			}
		}
	}

	// Method 2: EPUB 2.0 - check for meta name="cover"
	if opf.Metadata.CoverID != "" {
		if item, ok := opf.Manifest[opf.Metadata.CoverID]; ok && isImageMediaType(item.MediaType) {
			return item.Href, true
		}
	}

	// Method 3: guide type="cover" -> matching image manifest item (ignore fragment)
	for _, ref := range opf.Guide {
		if !strings.EqualFold(ref.Type, "cover") {
			continue
		}
		guideHref := stripFragment(ref.Href)
		if guideHref == "" {
			continue
		}
		for _, item := range orderedItems {
			if !isImageMediaType(item.MediaType) {
				continue
			}
			if stripFragment(item.Href) == guideHref {
				return item.Href, true
			}
		}
	}

	// Method 4: filename pattern (contains "cover", case-insensitive, SVG excluded)
	for _, item := range orderedItems {
		if !isImageMediaType(item.MediaType) {
			continue
		}
		if strings.Contains(strings.ToLower(filepath.Base(item.Href)), "cover") {
			return item.Href, true
		}
	}

	return "", false
}

func orderedManifestItems(opf *OPF) []ManifestItem {
	seen := make(map[string]struct{}, len(opf.Manifest))
	items := make([]ManifestItem, 0, len(opf.Manifest))

	for _, id := range opf.ManifestOrder {
		item, ok := opf.Manifest[id]
		if !ok {
			continue
		}
		items = append(items, item)
		seen[id] = struct{}{}
	}

	for id, item := range opf.Manifest {
		if _, ok := seen[id]; ok {
			continue
		}
		items = append(items, item)
	}

	return items
}

func stripFragment(href string) string {
	pathPart, _, _ := strings.Cut(href, "#")
	return pathPart
}

func isImageMediaType(mediaType string) bool {
	if mediaType == "image/svg+xml" {
		return false
	}
	return strings.HasPrefix(mediaType, "image/")
}
