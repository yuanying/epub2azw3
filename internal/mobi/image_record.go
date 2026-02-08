package mobi

import (
	"fmt"
	"regexp"
)

// ImageRecord holds data for a single image record in the AZW3 file.
type ImageRecord struct {
	Data         []byte
	OriginalPath string
	MediaType    string
}

// ImageMapper manages image records and their path-to-index mappings.
type ImageMapper struct {
	Images      []ImageRecord
	PathToIndex map[string]int
}

// NewImageMapper creates a new empty ImageMapper.
func NewImageMapper() *ImageMapper {
	return &ImageMapper{
		Images:      nil,
		PathToIndex: make(map[string]int),
	}
}

// AddImage adds an image to the mapper. Duplicate paths are skipped.
func (m *ImageMapper) AddImage(path string, data []byte, mediaType string) {
	if _, exists := m.PathToIndex[path]; exists {
		return
	}

	idx := len(m.Images)
	m.PathToIndex[path] = idx
	m.Images = append(m.Images, ImageRecord{
		Data:         data,
		OriginalPath: path,
		MediaType:    mediaType,
	})
}

// KindleEmbedRef returns the kindle:embed:XXXX reference for a given image path.
// The XXXX is the 1-based index as a 4-digit zero-padded hexadecimal number.
func (m *ImageMapper) KindleEmbedRef(path string) (string, bool) {
	idx, ok := m.PathToIndex[path]
	if !ok {
		return "", false
	}
	// kindle:embed uses 1-based indexing
	return fmt.Sprintf("kindle:embed:%04X", idx+1), true
}

// ImageRecordData returns the raw image data for each image record,
// ready to be written to the AZW3 file.
func (m *ImageMapper) ImageRecordData() [][]byte {
	if len(m.Images) == 0 {
		return nil
	}
	records := make([][]byte, len(m.Images))
	for i, img := range m.Images {
		records[i] = img.Data
	}
	return records
}

// imgSrcRe matches <img src="..."> attributes in HTML.
var imgSrcRe = regexp.MustCompile(`(<img\s[^>]*?)src="([^"]*)"`)

// TransformImageReferences replaces img src attributes in HTML with
// kindle:embed:XXXX references using the provided ImageMapper.
func TransformImageReferences(html string, mapper *ImageMapper) string {
	if mapper == nil || len(mapper.Images) == 0 {
		return html
	}

	return imgSrcRe.ReplaceAllStringFunc(html, func(match string) string {
		submatch := imgSrcRe.FindStringSubmatch(match)
		if len(submatch) < 3 {
			return match
		}
		prefix := submatch[1]
		srcPath := submatch[2]

		ref, ok := mapper.KindleEmbedRef(srcPath)
		if !ok {
			return match
		}
		return prefix + `src="` + ref + `"`
	})
}
