package converter

import (
	"path/filepath"
	"strings"

	"github.com/yuanying/epub2azw3/internal/epub"
	"github.com/yuanying/epub2azw3/internal/mobi"
)

// CoverInfo represents the detected cover image details.
type CoverInfo struct {
	ManifestID      string
	Href            string
	MediaType       string
	DetectionMethod string
}

type fileReader interface {
	ReadFile(path string) ([]byte, error)
}

// DetectCoverInfo detects cover image using prioritized methods.
func DetectCoverInfo(opf *epub.OPF, reader fileReader) *CoverInfo {
	if opf == nil {
		return nil
	}

	if info := detectCoverByProperty(opf); info != nil {
		return info
	}
	if info := detectCoverByMetadata(opf); info != nil {
		return info
	}
	if info := detectCoverByGuide(opf, reader); info != nil {
		return info
	}
	if info := detectCoverByFilename(opf); info != nil {
		return info
	}

	return nil
}

// ComputeCoverOffset calculates EXTH 131 cover offset from ImageMapper.
func ComputeCoverOffset(cover *CoverInfo, mapper *mobi.ImageMapper) (uint32, bool) {
	if cover == nil || mapper == nil {
		return 0, false
	}
	idx, ok := mapper.PathToIndex[cover.Href]
	if !ok {
		return 0, false
	}
	return uint32(idx), true
}

func detectCoverByProperty(opf *epub.OPF) *CoverInfo {
	for _, item := range orderedManifestItems(opf) {
		if !isImage(item.MediaType) {
			continue
		}
		for _, prop := range item.Properties {
			if strings.EqualFold(prop, "cover-image") {
				return &CoverInfo{
					ManifestID:      item.ID,
					Href:            item.Href,
					MediaType:       item.MediaType,
					DetectionMethod: "manifest-property",
				}
			}
		}
	}
	return nil
}

func detectCoverByMetadata(opf *epub.OPF) *CoverInfo {
	if opf.Metadata.CoverID == "" {
		return nil
	}
	item, ok := opf.Manifest[opf.Metadata.CoverID]
	if !ok || !isImage(item.MediaType) {
		return nil
	}
	return &CoverInfo{
		ManifestID:      item.ID,
		Href:            item.Href,
		MediaType:       item.MediaType,
		DetectionMethod: "metadata-cover",
	}
}

func detectCoverByGuide(opf *epub.OPF, reader fileReader) *CoverInfo {
	for _, ref := range opf.Guide {
		if !strings.EqualFold(ref.Type, "cover") {
			continue
		}

		targetPath := stripFragment(normalizePath(ref.Href))
		if targetPath == "" {
			continue
		}

		if item, ok := findManifestByHref(opf, targetPath); ok && isImage(item.MediaType) {
			return &CoverInfo{
				ManifestID:      item.ID,
				Href:            item.Href,
				MediaType:       item.MediaType,
				DetectionMethod: "guide-reference",
			}
		}

		if reader == nil {
			continue
		}

		manifestItem, manifestOK := findManifestByHref(opf, targetPath)
		if !manifestOK && !looksLikeXHTML(targetPath) {
			continue
		}
		if manifestOK && !isXHTML(manifestItem.MediaType) {
			continue
		}

		data, err := reader.ReadFile(targetPath)
		if err != nil {
			continue
		}

		content, err := epub.LoadContent(manifestItem.ID, targetPath, data)
		if err != nil || len(content.ImageRefs) == 0 {
			continue
		}

		firstImage := normalizePath(content.ImageRefs[0])
		if imageItem, ok := findManifestByHref(opf, firstImage); ok && isImage(imageItem.MediaType) {
			return &CoverInfo{
				ManifestID:      imageItem.ID,
				Href:            imageItem.Href,
				MediaType:       imageItem.MediaType,
				DetectionMethod: "guide-xhtml-first-img",
			}
		}
	}
	return nil
}

func detectCoverByFilename(opf *epub.OPF) *CoverInfo {
	for _, item := range orderedManifestItems(opf) {
		if !isImage(item.MediaType) {
			continue
		}
		base := strings.ToLower(filepath.Base(item.Href))
		if strings.Contains(base, "cover") {
			return &CoverInfo{
				ManifestID:      item.ID,
				Href:            item.Href,
				MediaType:       item.MediaType,
				DetectionMethod: "filename-pattern",
			}
		}
	}
	return nil
}

func orderedManifestItems(opf *epub.OPF) []epub.ManifestItem {
	seen := make(map[string]struct{}, len(opf.Manifest))
	items := make([]epub.ManifestItem, 0, len(opf.Manifest))

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

func findManifestByHref(opf *epub.OPF, href string) (epub.ManifestItem, bool) {
	normalized := stripFragment(normalizePath(href))
	for _, item := range opf.Manifest {
		if stripFragment(normalizePath(item.Href)) == normalized {
			return item, true
		}
	}
	return epub.ManifestItem{}, false
}

func stripFragment(href string) string {
	pathPart, _, _ := strings.Cut(href, "#")
	return pathPart
}

func normalizePath(path string) string {
	if strings.TrimSpace(path) == "" {
		return ""
	}
	return filepath.ToSlash(filepath.Clean(path))
}

func looksLikeXHTML(path string) bool {
	lower := strings.ToLower(path)
	return strings.HasSuffix(lower, ".xhtml") || strings.HasSuffix(lower, ".html") || strings.HasSuffix(lower, ".htm")
}
