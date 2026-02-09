package converter

import (
	"bytes"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/PuerkitoBio/goquery"
	"github.com/yuanying/epub2azw3/internal/epub"
	"github.com/yuanying/epub2azw3/internal/mobi"
)

// ConvertOptions holds options for the conversion pipeline.
type ConvertOptions struct {
	InputPath  string
	OutputPath string
}

// Pipeline orchestrates the EPUB to AZW3 conversion.
type Pipeline struct {
	Options ConvertOptions
}

// NewPipeline creates a new conversion pipeline.
func NewPipeline(opts ConvertOptions) *Pipeline {
	return &Pipeline{Options: opts}
}

// Convert executes the conversion pipeline.
func (p *Pipeline) Convert() error {
	reader, opf, err := p.parseEPUB()
	if err != nil {
		return err
	}
	defer reader.Close()

	html, imageMapper, builder, err := p.buildHTML(reader, opf)
	if err != nil {
		return err
	}

	// Load NCX for TOC generation
	ncx, err := epub.LoadNCX(reader, opf)
	if err != nil {
		log.Printf("warning: failed to load NCX: %v", err)
	}

	// Generate inline TOC and insert into HTML (before image reference transformation)
	var tocGen *TOCGenerator
	if ncx != nil && len(ncx.NavPoints) > 0 {
		tocGen = NewTOCGenerator(ncx, builder.GetChapterIDs())
		html = tocGen.InsertInlineTOC(html)
	}

	// Transform image references to kindle:embed format
	html = mobi.TransformImageReferences(html, imageMapper)

	// Build NCX record
	var ncxRecord []byte
	if tocGen != nil {
		finalHTML := []byte(html)
		entries, err := tocGen.BuildTOCEntries(finalHTML)
		if err != nil {
			log.Printf("warning: failed to build TOC entries: %v", err)
		} else if len(entries) > 0 {
			ncxEntries := convertTOCEntries(entries)
			ncxRecord = mobi.GenerateNCXRecord(mobi.NCXRecordConfig{
				Title:   opf.Metadata.Title,
				Entries: ncxEntries,
				Guide:   buildGuideReferences(finalHTML),
			})
		}
	}

	return p.writeAZW3(html, &opf.Metadata, imageMapper, ncxRecord)
}

// parseEPUB opens the EPUB file and parses the OPF.
func (p *Pipeline) parseEPUB() (*epub.EPUBReader, *epub.OPF, error) {
	reader, err := epub.Open(p.Options.InputPath)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to open EPUB: %w", err)
	}

	opfData, err := reader.ReadFile(reader.OPFPath())
	if err != nil {
		reader.Close()
		return nil, nil, fmt.Errorf("failed to read OPF: %w", err)
	}

	opfDir := filepath.Dir(reader.OPFPath())
	opf, err := epub.ParseOPF(opfData, opfDir)
	if err != nil {
		reader.Close()
		return nil, nil, fmt.Errorf("failed to parse OPF: %w", err)
	}

	return reader, opf, nil
}

// buildHTML loads spine items and builds the integrated HTML.
// It also collects images referenced in the content.
func (p *Pipeline) buildHTML(reader *epub.EPUBReader, opf *epub.OPF) (string, *mobi.ImageMapper, *HTMLBuilder, error) {
	builder := NewHTMLBuilder()
	cssCache := make(map[string]string)
	validChapters := 0

	type cssRef struct {
		chapterID string
		path      string
	}
	var orderedCSS []cssRef

	for _, spineItem := range opf.Spine {
		manifestItem, ok := opf.Manifest[spineItem.IDRef]
		if !ok {
			log.Printf("warning: spine item %q not found in manifest, skipping", spineItem.IDRef)
			continue
		}

		// Check if this is an XHTML file
		if !isXHTML(manifestItem.MediaType) {
			continue
		}

		data, err := reader.ReadFile(manifestItem.Href)
		if err != nil {
			log.Printf("warning: failed to read %q: %v, skipping", manifestItem.Href, err)
			continue
		}

		content, err := epub.LoadContent(manifestItem.ID, manifestItem.Href, data)
		if err != nil {
			log.Printf("warning: failed to parse %q: %v, skipping", manifestItem.Href, err)
			continue
		}

		// Resolve img src attributes to absolute EPUB paths
		// so they match manifest Href paths for image reference transformation
		chapterDir := filepath.Dir(manifestItem.Href)
		content.Document.Find("img[src]").Each(func(i int, s *goquery.Selection) {
			if src, exists := s.Attr("src"); exists {
				resolved := filepath.ToSlash(filepath.Clean(filepath.Join(chapterDir, src)))
				s.SetAttr("src", resolved)
			}
		})

		if err := builder.AddChapter(content); err != nil {
			log.Printf("warning: failed to add chapter %q: %v, skipping", manifestItem.Href, err)
			continue
		}

		validChapters++

		chapterID := builder.GetChapterID(manifestItem.Href)
		for _, cssPath := range content.CSSLinks {
			orderedCSS = append(orderedCSS, cssRef{
				chapterID: chapterID,
				path:      cssPath,
			})
		}
	}

	if validChapters == 0 {
		return "", nil, nil, fmt.Errorf("no valid XHTML chapters found")
	}

	// Load and add CSS with chapter namespacing
	for _, ref := range orderedCSS {
		cssText, ok := cssCache[ref.path]
		if !ok {
			cssData, err := reader.ReadFile(ref.path)
			if err != nil {
				log.Printf("warning: failed to read CSS %q: %v, skipping", ref.path, err)
				continue
			}
			cssText = string(cssData)
			cssCache[ref.path] = cssText
		}
		builder.AddChapterCSS(ref.chapterID, cssText)
	}

	// Collect images from manifest in document order
	imageMapper := mobi.NewImageMapper()
	for _, id := range opf.ManifestOrder {
		item, ok := opf.Manifest[id]
		if !ok {
			continue
		}
		if !isImage(item.MediaType) {
			continue
		}
		imgData, err := reader.ReadFile(item.Href)
		if err != nil {
			log.Printf("warning: failed to read image %q: %v, skipping", item.Href, err)
			continue
		}
		imageMapper.AddImage(item.Href, imgData, item.MediaType)
	}

	html, err := builder.Build()
	if err != nil {
		return "", nil, nil, fmt.Errorf("failed to build HTML: %w", err)
	}

	return html, imageMapper, builder, nil
}

// writeAZW3 creates the AZW3 file from the integrated HTML and metadata.
func (p *Pipeline) writeAZW3(html string, metadata *epub.Metadata, imageMapper *mobi.ImageMapper, ncxRecord []byte) error {
	title := metadata.Title
	if title == "" {
		title = "Untitled"
	}

	cfg := mobi.AZW3WriterConfig{
		Title:       title,
		HTML:        []byte(html),
		Metadata:    metadata,
		NCXRecord:   ncxRecord,
		Compression: mobi.CompressionPalmDoc,
	}

	if imageMapper != nil {
		cfg.ImageRecords = imageMapper.ImageRecordData()
	}

	writer, err := mobi.NewAZW3Writer(cfg)
	if err != nil {
		return fmt.Errorf("failed to create AZW3 writer: %w", err)
	}

	f, err := os.Create(p.Options.OutputPath)
	if err != nil {
		return fmt.Errorf("failed to create output file: %w", err)
	}
	defer f.Close()

	if _, err := writer.WriteTo(f); err != nil {
		return fmt.Errorf("failed to write AZW3: %w", err)
	}

	return nil
}

// convertTOCEntries converts converter.TOCEntry slice to mobi.NCXEntry slice.
func convertTOCEntries(entries []TOCEntry) []mobi.NCXEntry {
	result := make([]mobi.NCXEntry, len(entries))
	for i, e := range entries {
		result[i] = mobi.NCXEntry{
			Label:    e.Label,
			FilePos:  e.FilePos,
			Children: convertTOCEntries(e.Children),
		}
	}
	return result
}

// buildGuideReferences creates guide references for the NCX record.
// It includes a "toc" reference if an inline TOC div exists in the HTML.
func buildGuideReferences(finalHTML []byte) []mobi.GuideReference {
	var refs []mobi.GuideReference

	// Find the inline TOC position
	tocPattern := []byte(`id="toc"`)
	if idx := bytes.Index(finalHTML, tocPattern); idx >= 0 {
		// Walk backwards to find the '<' that opens this tag
		tagStart := idx
		for tagStart > 0 && finalHTML[tagStart] != '<' {
			tagStart--
		}
		refs = append(refs, mobi.GuideReference{
			Type:    "toc",
			Title:   "Table of Contents",
			FilePos: uint32(tagStart),
		})
	}

	return refs
}

// isXHTML checks if a media type indicates an XHTML content file.
func isXHTML(mediaType string) bool {
	return strings.Contains(mediaType, "html") || strings.Contains(mediaType, "xhtml")
}

// isImage checks if a media type indicates a raster image file.
// SVG (image/svg+xml) is excluded as Kindle does not support it.
func isImage(mediaType string) bool {
	if mediaType == "image/svg+xml" {
		return false
	}
	return strings.HasPrefix(mediaType, "image/")
}
