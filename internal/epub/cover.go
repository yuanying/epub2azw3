package epub

import (
	"path/filepath"
	"strings"
)

// CoverInfo holds information about the detected cover image.
type CoverInfo struct {
	ManifestID      string
	Href            string
	MediaType       string
	DetectionMethod string // "properties", "meta", "guide", "filename"
}

// DetectCover detects the cover image from the OPF manifest using multiple methods.
// Methods are tried in priority order:
//  1. properties="cover-image" (EPUB 3.0)
//  2. meta name="cover" (EPUB 2.0)
//  3. guide type="cover" (matched to image manifest items)
//  4. filename pattern (basename contains "cover", case-insensitive, SVG excluded)
//
// Returns nil if no cover image is found.
func (opf *OPF) DetectCover() *CoverInfo {
	// Method 1: EPUB 3.0 - check for cover-image property
	for _, id := range opf.ManifestOrder {
		item := opf.Manifest[id]
		for _, prop := range item.Properties {
			if prop == "cover-image" {
				return &CoverInfo{
					ManifestID:      item.ID,
					Href:            item.Href,
					MediaType:       item.MediaType,
					DetectionMethod: "properties",
				}
			}
		}
	}

	// Method 2: EPUB 2.0 - check for meta name="cover"
	if opf.Metadata.CoverID != "" {
		if item, ok := opf.Manifest[opf.Metadata.CoverID]; ok {
			return &CoverInfo{
				ManifestID:      item.ID,
				Href:            item.Href,
				MediaType:       item.MediaType,
				DetectionMethod: "meta",
			}
		}
	}

	// Method 3: guide type="cover" → match to image manifest items
	for _, ref := range opf.Guide {
		if ref.Type != "cover" {
			continue
		}
		// Strip fragment from Href for matching
		guideHref := ref.Href
		if idx := strings.Index(guideHref, "#"); idx >= 0 {
			guideHref = guideHref[:idx]
		}
		// Find matching image item in manifest
		for _, id := range opf.ManifestOrder {
			item := opf.Manifest[id]
			if !isImageMediaType(item.MediaType) {
				continue
			}
			if item.Href == guideHref {
				return &CoverInfo{
					ManifestID:      item.ID,
					Href:            item.Href,
					MediaType:       item.MediaType,
					DetectionMethod: "guide",
				}
			}
		}
		// Guide points to a non-image → skip to Method 4
	}

	// Method 4: filename pattern - image items with "cover" in basename (case-insensitive, SVG excluded)
	for _, id := range opf.ManifestOrder {
		item := opf.Manifest[id]
		if !isImageMediaType(item.MediaType) {
			continue
		}
		base := filepath.Base(item.Href)
		if strings.Contains(strings.ToLower(base), "cover") {
			return &CoverInfo{
				ManifestID:      item.ID,
				Href:            item.Href,
				MediaType:       item.MediaType,
				DetectionMethod: "filename",
			}
		}
	}

	return nil
}

// FindCoverImage finds the cover image in the manifest.
// This is a convenience wrapper around DetectCover.
func (opf *OPF) FindCoverImage() (string, bool) {
	if c := opf.DetectCover(); c != nil {
		return c.Href, true
	}
	return "", false
}

// isImageMediaType checks if a media type is a raster image (SVG excluded).
func isImageMediaType(mediaType string) bool {
	if mediaType == "image/svg+xml" {
		return false
	}
	return strings.HasPrefix(mediaType, "image/")
}
