package converter

import (
	"bytes"
	"context"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"

	"github.com/PuerkitoBio/goquery"
	"github.com/yuanying/epub2azw3/internal/epub"
	"github.com/yuanying/epub2azw3/internal/mobi"
)

// ConvertOptions holds options for the conversion pipeline.
type ConvertOptions struct {
	InputPath         string
	OutputPath        string
	MaxImageWidth     int
	JPEGQuality       int
	MaxImageSizeBytes int
	NoImages          bool
	Strict            bool
	Logger            *slog.Logger
}

type ErrorLevel string

const (
	ErrorLevelFatal       ErrorLevel = "Fatal"
	ErrorLevelRecoverable ErrorLevel = "Recoverable"
	ErrorLevelAcceptable  ErrorLevel = "Acceptable"
)

// ConvertError represents a structured conversion error.
type ConvertError struct {
	Level   ErrorLevel
	Context string
	Message string
	Cause   error
}

// discardHandler is a slog.Handler that discards all log records.
type discardHandler struct{}

func (discardHandler) Enabled(context.Context, slog.Level) bool  { return false }
func (discardHandler) Handle(context.Context, slog.Record) error { return nil }
func (h discardHandler) WithAttrs([]slog.Attr) slog.Handler      { return h }
func (h discardHandler) WithGroup(string) slog.Handler           { return h }

// Pipeline orchestrates the EPUB to AZW3 conversion.
type Pipeline struct {
	Options ConvertOptions
	errors  []ConvertError
	logger  *slog.Logger
}

// NewPipeline creates a new conversion pipeline.
func NewPipeline(opts ConvertOptions) *Pipeline {
	logger := opts.Logger
	if logger == nil {
		logger = slog.New(discardHandler{})
	}
	return &Pipeline{
		Options: opts,
		logger:  logger,
		errors:  make([]ConvertError, 0),
	}
}

// Convert executes the conversion pipeline.
func (p *Pipeline) Convert() error {
	p.stageStart("parse", "parse EPUB")
	reader, opf, err := p.parseEPUB()
	if err != nil {
		return p.fatal("parse", "failed to parse EPUB", err)
	}
	defer reader.Close()
	p.stageDone("parse", "parse EPUB")

	if err := p.validateRequiredMetadata(&opf.Metadata); err != nil {
		return p.fatal("metadata", "required metadata is missing", err)
	}

	cover := DetectCoverInfo(opf, reader)
	if cover == nil {
		p.acceptable("cover", "cover image not found", nil)
	}

	p.stageStart("build", "build integrated HTML")
	html, imageMapper, builder, err := p.buildHTML(reader, opf, cover)
	if err != nil {
		return p.fatal("build", "failed to build HTML", err)
	}
	p.stageDone("build", "build integrated HTML")

	var coverOffset *uint32
	if cover != nil && !p.Options.NoImages {
		if offset, ok := ComputeCoverOffset(cover, imageMapper); ok {
			coverOffset = &offset
		} else {
			p.recoverable(
				"cover",
				fmt.Sprintf("cover image detected (%s) but not found in image records", cover.DetectionMethod),
				fmt.Errorf("%s", cover.Href),
			)
		}
	}

	p.stageStart("toc", "load NCX and generate TOC")
	ncx, err := epub.LoadNCX(reader, opf)
	if err != nil {
		p.recoverable("toc", "failed to load NCX", err)
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
		entries, buildErr := tocGen.BuildTOCEntries(finalHTML)
		if buildErr != nil {
			p.recoverable("toc", "failed to build TOC entries", buildErr)
		} else if len(entries) > 0 {
			ncxEntries := convertTOCEntries(entries)
			ncxRecord = mobi.GenerateNCXRecord(mobi.NCXRecordConfig{
				Title:   opf.Metadata.Title,
				Entries: ncxEntries,
				Guide:   buildGuideReferences(finalHTML),
			})
		}
	}
	p.stageDone("toc", "load NCX and generate TOC")

	p.stageStart("write", "write AZW3")
	if err := p.writeAZW3(html, &opf.Metadata, imageMapper, ncxRecord, coverOffset); err != nil {
		return p.fatal("write", "failed to write AZW3", err)
	}
	p.stageDone("write", "write AZW3")

	if stat, err := os.Stat(p.Options.OutputPath); err == nil {
		p.logger.Info(fmt.Sprintf("output size: %d bytes", stat.Size()), "stage", "result")
	}

	if p.Options.Strict {
		if err := p.strictFailureIfNeeded(); err != nil {
			return err
		}
	}

	return nil
}

func (p *Pipeline) strictFailureIfNeeded() error {
	recoverables := make([]ConvertError, 0)
	for _, ce := range p.errors {
		if ce.Level == ErrorLevelRecoverable {
			recoverables = append(recoverables, ce)
		}
	}
	if len(recoverables) == 0 {
		return nil
	}

	p.logger.Error(fmt.Sprintf("%d recoverable errors collected", len(recoverables)), "stage", "strict")
	for i, ce := range recoverables {
		if ce.Cause != nil {
			p.logger.Error(fmt.Sprintf("[%d] %s: %s (%v)", i+1, ce.Context, ce.Message, ce.Cause), "stage", "strict")
			continue
		}
		p.logger.Error(fmt.Sprintf("[%d] %s: %s", i+1, ce.Context, ce.Message), "stage", "strict")
	}

	return fmt.Errorf("strict mode failed: %d recoverable errors", len(recoverables))
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

func (p *Pipeline) validateRequiredMetadata(metadata *epub.Metadata) error {
	missing := make([]string, 0, 3)
	if strings.TrimSpace(metadata.Title) == "" {
		missing = append(missing, "title")
	}
	if strings.TrimSpace(metadata.Language) == "" {
		missing = append(missing, "language")
	}
	if strings.TrimSpace(metadata.Identifier) == "" {
		missing = append(missing, "identifier")
	}
	if len(missing) == 0 {
		return nil
	}
	return fmt.Errorf("missing required metadata: %s", strings.Join(missing, ", "))
}

// buildHTML loads spine items and builds the integrated HTML.
// It also collects images referenced in the content.
func (p *Pipeline) buildHTML(reader *epub.EPUBReader, opf *epub.OPF, cover *CoverInfo) (string, *mobi.ImageMapper, *HTMLBuilder, error) {
	builder := NewHTMLBuilder()
	cssCache := make(map[string]string)
	validChapters := 0
	totalChapters := 0

	type cssRef struct {
		chapterID string
		path      string
	}
	var orderedCSS []cssRef

	for _, spineItem := range opf.Spine {
		manifestItem, ok := opf.Manifest[spineItem.IDRef]
		if ok && isXHTML(manifestItem.MediaType) {
			totalChapters++
		}
	}

	chapterProgress := 0
	for _, spineItem := range opf.Spine {
		manifestItem, ok := opf.Manifest[spineItem.IDRef]
		if !ok {
			p.recoverable("html", fmt.Sprintf("spine item %q not found in manifest, skipping", spineItem.IDRef), nil)
			continue
		}

		if !isXHTML(manifestItem.MediaType) {
			continue
		}

		chapterProgress++
		p.logger.Info(fmt.Sprintf("chapter %d/%d: %s", chapterProgress, totalChapters, manifestItem.Href), "stage", "progress")

		data, err := reader.ReadFile(manifestItem.Href)
		if err != nil {
			p.recoverable("html", fmt.Sprintf("failed to read %q, skipping", manifestItem.Href), err)
			continue
		}

		content, err := epub.LoadContent(manifestItem.ID, manifestItem.Href, data)
		if err != nil {
			p.recoverable("html", fmt.Sprintf("failed to parse %q, skipping", manifestItem.Href), err)
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
			p.recoverable("html", fmt.Sprintf("failed to add chapter %q, skipping", manifestItem.Href), err)
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
				p.recoverable("css", fmt.Sprintf("failed to read CSS %q, skipping", ref.path), err)
				continue
			}
			cssText = string(cssData)
			cssCache[ref.path] = cssText
		}
		builder.AddChapterCSS(ref.chapterID, cssText)
	}

	imageMapper := mobi.NewImageMapper()
	if p.Options.NoImages {
		p.logger.Info("--no-images enabled; removing all img tags", "stage", "images")
		builder.RemoveImages()
		html, err := builder.Build()
		if err != nil {
			return "", nil, nil, fmt.Errorf("failed to build HTML: %w", err)
		}
		return html, imageMapper, builder, nil
	}

	// Collect images from manifest in document order
	optimizer := NewImageOptimizer(p.Options)
	totalImages := 0
	for _, id := range opf.ManifestOrder {
		item, ok := opf.Manifest[id]
		if ok && isImage(item.MediaType) {
			totalImages++
		}
	}

	imageProgress := 0
	for _, id := range opf.ManifestOrder {
		item, ok := opf.Manifest[id]
		if !ok {
			continue
		}
		if isSVG(item.MediaType) {
			p.acceptable("images", fmt.Sprintf("SVG image %q is not supported and will be skipped", item.Href), nil)
			continue
		}
		if !isImage(item.MediaType) {
			continue
		}

		imageProgress++
		p.logger.Info(fmt.Sprintf("image %d/%d: %s", imageProgress, totalImages, item.Href), "stage", "progress")

		imgData, err := reader.ReadFile(item.Href)
		if err != nil {
			p.recoverable("images", fmt.Sprintf("failed to read image %q, skipping", item.Href), err)
			continue
		}

		isCover := cover != nil && cover.Href == item.Href
		optimized, optErr := optimizer.Optimize(item.Href, item.MediaType, imgData, isCover)
		if optErr != nil {
			p.recoverable("images", fmt.Sprintf("image optimization failed for %q; using original", item.Href), optErr)
		}
		if optimized.Warning != "" {
			p.recoverable("images", fmt.Sprintf("image optimization warning for %q: %s", item.Href, optimized.Warning), nil)
		}

		mediaType := item.MediaType
		if optimized.Format != "" {
			switch strings.ToLower(optimized.Format) {
			case "jpeg", "jpg":
				mediaType = "image/jpeg"
			case "png":
				mediaType = "image/png"
			case "gif":
				mediaType = "image/gif"
			}
		}
		imageMapper.AddImage(item.Href, optimized.Data, mediaType)
	}

	html, err := builder.Build()
	if err != nil {
		return "", nil, nil, fmt.Errorf("failed to build HTML: %w", err)
	}

	return html, imageMapper, builder, nil
}

// writeAZW3 creates the AZW3 file from the integrated HTML and metadata.
func (p *Pipeline) writeAZW3(html string, metadata *epub.Metadata, imageMapper *mobi.ImageMapper, ncxRecord []byte, coverOffset *uint32) error {
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
		CoverOffset: coverOffset,
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

func (p *Pipeline) stageStart(stage, message string) {
	p.logger.Info("start: "+message, "stage", stage)
}

func (p *Pipeline) stageDone(stage, message string) {
	p.logger.Info("done: "+message, "stage", stage)
}

func (p *Pipeline) fatal(stage, message string, cause error) error {
	p.errors = append(p.errors, ConvertError{
		Level:   ErrorLevelFatal,
		Context: stage,
		Message: message,
		Cause:   cause,
	})
	if cause != nil {
		p.logger.Error(message, "stage", stage, "error", cause)
		return fmt.Errorf("%s: %w", message, cause)
	}
	p.logger.Error(message, "stage", stage)
	return fmt.Errorf("%s", message)
}

func (p *Pipeline) recoverable(stage, message string, cause error) {
	p.errors = append(p.errors, ConvertError{
		Level:   ErrorLevelRecoverable,
		Context: stage,
		Message: message,
		Cause:   cause,
	})
	if cause != nil {
		p.logger.Warn(message, "stage", stage, "error", cause)
		return
	}
	p.logger.Warn(message, "stage", stage)
}

func (p *Pipeline) acceptable(stage, message string, cause error) {
	p.errors = append(p.errors, ConvertError{
		Level:   ErrorLevelAcceptable,
		Context: stage,
		Message: message,
		Cause:   cause,
	})
	if cause != nil {
		p.logger.Info(message, "stage", stage, "error", cause)
		return
	}
	p.logger.Info(message, "stage", stage)
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

// normalizeMediaType extracts the base MIME type, lowercased, without parameters.
func normalizeMediaType(mt string) string {
	mt = strings.ToLower(strings.TrimSpace(mt))
	base, _, _ := strings.Cut(mt, ";")
	return strings.TrimSpace(base)
}

// isSVG checks if a media type indicates an SVG image.
func isSVG(mediaType string) bool {
	return normalizeMediaType(mediaType) == "image/svg+xml"
}

// isImage checks if a media type indicates a raster image file.
// SVG (image/svg+xml) is excluded as Kindle does not support it.
func isImage(mediaType string) bool {
	if isSVG(mediaType) {
		return false
	}
	return strings.HasPrefix(normalizeMediaType(mediaType), "image/")
}
