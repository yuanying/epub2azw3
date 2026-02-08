package converter

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"

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

	html, err := p.buildHTML(reader, opf)
	if err != nil {
		return err
	}

	return p.writeAZW3(html, &opf.Metadata)
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
func (p *Pipeline) buildHTML(reader *epub.EPUBReader, opf *epub.OPF) (string, error) {
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
		return "", fmt.Errorf("no valid XHTML chapters found")
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

	html, err := builder.Build()
	if err != nil {
		return "", fmt.Errorf("failed to build HTML: %w", err)
	}

	return html, nil
}

// writeAZW3 creates the AZW3 file from the integrated HTML and metadata.
func (p *Pipeline) writeAZW3(html string, metadata *epub.Metadata) error {
	title := metadata.Title
	if title == "" {
		title = "Untitled"
	}

	cfg := mobi.AZW3WriterConfig{
		Title:    title,
		HTML:     []byte(html),
		Metadata: metadata,
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

// isXHTML checks if a media type indicates an XHTML content file.
func isXHTML(mediaType string) bool {
	return strings.Contains(mediaType, "html") || strings.Contains(mediaType, "xhtml")
}
