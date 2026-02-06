// Test program for HTML integration functionality
// Related: Issue #4, PR #13
//
// Usage:
//   go run ./cmd/test/html_builder/main.go <epub-file-path> [output-html-path]
//
// This program:
// 1. Opens the specified EPUB file
// 2. Parses the OPF file to get spine items and metadata
// 3. Loads all XHTML content files
// 4. Collects referenced CSS files
// 5. Uses HTMLBuilder to integrate all chapters into a single HTML document
// 6. Saves the integrated HTML to output file (default: integrated.html)
// 7. Displays statistics about the integration
//
// Verification points:
// - ✓ All chapters are loaded and integrated in spine order
// - ✓ Each chapter has a unique ID (ch01, ch02, ...)
// - ✓ Pagebreaks are inserted before each chapter
// - ✓ CSS files are integrated into <style> tag
// - ✓ Internal links are converted to fragment identifiers
// - ✓ External links are preserved
// - ✓ Output HTML is valid and readable

package main

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/PuerkitoBio/goquery"
	"github.com/yuanying/epub2azw3/internal/converter"
	"github.com/yuanying/epub2azw3/internal/epub"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Fprintf(os.Stderr, "Usage: %s <epub-file-path> [output-html-path]\n", filepath.Base(os.Args[0]))
		os.Exit(1)
	}

	epubPath := os.Args[1]
	outputPath := "integrated.html"
	if len(os.Args) >= 3 {
		outputPath = os.Args[2]
	}

	fmt.Printf("=== HTML Integration Test ===\n")
	fmt.Printf("EPUB file: %s\n", epubPath)
	fmt.Printf("Output file: %s\n\n", outputPath)

	// Open EPUB file
	reader, err := epub.Open(epubPath)
	if err != nil {
		log.Fatalf("Failed to open EPUB: %v", err)
	}
	defer reader.Close()

	fmt.Printf("✓ EPUB opened successfully\n")

	// Parse OPF
	opfPath := reader.OPFPath()
	opfData, err := reader.ReadFile(opfPath)
	if err != nil {
		log.Fatalf("Failed to read OPF: %v", err)
	}

	// Get OPF directory for relative path resolution
	opfDir := filepath.Dir(opfPath)

	opf, err := epub.ParseOPF(opfData, opfDir)
	if err != nil {
		log.Fatalf("Failed to parse OPF: %v", err)
	}

	fmt.Printf("✓ OPF parsed successfully\n")
	fmt.Printf("Title: %s\n", opf.Metadata.Title)
	fmt.Printf("Spine items: %d\n\n", len(opf.Spine))

	// Create HTMLBuilder
	builder := converter.NewHTMLBuilder()

	fmt.Println("=== Loading and Integrating Chapters ===")

	// Track statistics
	totalChapters := 0
	cssFiles := make(map[string]bool)
	type cssRef struct {
		chapterID string
		path      string
	}
	orderedCSS := make([]cssRef, 0, 64)
	totalLinks := 0

	// Process each spine item
	for i, spineItem := range opf.Spine {
		manifestItem, exists := opf.Manifest[spineItem.IDRef]
		if !exists {
			log.Printf("Warning: Spine item %s not found in manifest", spineItem.IDRef)
			continue
		}

		// Only process XHTML files
		if !strings.Contains(manifestItem.MediaType, "html") &&
			!strings.Contains(manifestItem.MediaType, "xhtml") {
			continue
		}

		fmt.Printf("[%d/%d] Loading: %s\n", i+1, len(opf.Spine), manifestItem.Href)

		// Read the content file
		contentData, err := reader.ReadFile(manifestItem.Href)
		if err != nil {
			log.Printf("  ✗ Failed to read: %v", err)
			continue
		}

		// Load content using epub.LoadContent
		content, err := epub.LoadContent(spineItem.IDRef, manifestItem.Href, contentData)
		if err != nil {
			log.Printf("  ✗ Failed to parse: %v", err)
			continue
		}

		// Add chapter to builder
		err = builder.AddChapter(content)
		if err != nil {
			log.Printf("  ✗ Failed to add chapter: %v", err)
			continue
		}

		totalChapters++

		// Collect CSS references
		for _, cssPath := range content.CSSLinks {
			cssFiles[cssPath] = true
		}

		// Count links in this chapter
		linkCount := content.Document.Find("a[href]").Length()
		totalLinks += linkCount

		fmt.Printf("  ✓ Added as %s (CSS refs: %d, Links: %d)\n",
			builder.GetChapterID(manifestItem.Href),
			len(content.CSSLinks),
			linkCount)

		chapterID := builder.GetChapterID(manifestItem.Href)
		for _, cssPath := range content.CSSLinks {
			orderedCSS = append(orderedCSS, cssRef{
				chapterID: chapterID,
				path:      cssPath,
			})
		}
	}

	fmt.Printf("\n✓ Loaded %d chapters\n\n", totalChapters)

	// Load and add CSS files
	if len(orderedCSS) > 0 {
		fmt.Println("=== Loading CSS Files ===")

		cssCache := make(map[string]string)
		for _, ref := range orderedCSS {
			cssPath := ref.path
			fmt.Printf("Loading: %s (chapter %s)\n", cssPath, ref.chapterID)

			cssText, ok := cssCache[cssPath]
			if !ok {
				cssData, err := reader.ReadFile(cssPath)
				if err != nil {
					log.Printf("  ✗ Failed to read CSS: %v", err)
					continue
				}
				cssText = string(cssData)
				cssCache[cssPath] = cssText
			}

			builder.AddChapterCSS(ref.chapterID, cssText)
			fmt.Printf("  ✓ Added (%d bytes)\n", len(cssText))
		}

		fmt.Printf("\n✓ Integrated %d CSS references (%d unique files)\n\n", len(orderedCSS), len(cssCache))
	}

	// Build integrated HTML
	fmt.Println("=== Building Integrated HTML ===")

	integratedHTML, err := builder.Build()
	if err != nil {
		log.Fatalf("Failed to build HTML: %v", err)
	}

	fmt.Printf("✓ HTML built successfully (%d bytes)\n\n", len(integratedHTML))

	// Analyze the integrated HTML
	fmt.Println("=== Analyzing Integrated HTML ===")

	doc, err := goquery.NewDocumentFromReader(strings.NewReader(integratedHTML))
	if err != nil {
		log.Fatalf("Failed to parse integrated HTML: %v", err)
	}

	// Count chapters
	chapterDivs := doc.Find("body > div[id^='ch']")
	fmt.Printf("Chapters: %d\n", chapterDivs.Length())

	// Count pagebreaks
	pagebreaks := doc.Find("mbp\\:pagebreak")
	fmt.Printf("Pagebreaks: %d\n", pagebreaks.Length())

	// Check CSS
	styleTags := doc.Find("head style")
	fmt.Printf("Style tags: %d\n", styleTags.Length())
	if styleTags.Length() > 0 {
		styleContent := styleTags.Text()
		fmt.Printf("CSS size: %d bytes\n", len(styleContent))
	}

	// Count links
	allLinks := doc.Find("a[href]")
	fmt.Printf("Total links: %d\n", allLinks.Length())

	// Analyze link types
	internalLinks := 0
	externalLinks := 0
	fragmentLinks := 0

	allLinks.Each(func(i int, s *goquery.Selection) {
		href, _ := s.Attr("href")
		if strings.HasPrefix(href, "http://") || strings.HasPrefix(href, "https://") {
			externalLinks++
		} else if strings.HasPrefix(href, "#") {
			fragmentLinks++
			// Check if it's a chapter reference
			if strings.HasPrefix(href, "#ch") {
				internalLinks++
			}
		}
	})

	fmt.Printf("  - Fragment links: %d\n", fragmentLinks)
	fmt.Printf("  - Chapter links: %d\n", internalLinks)
	fmt.Printf("  - External links: %d\n", externalLinks)

	// Save to file
	fmt.Printf("\n=== Saving Output ===\n")

	err = os.WriteFile(outputPath, []byte(integratedHTML), 0644)
	if err != nil {
		log.Fatalf("Failed to write output file: %v", err)
	}

	fmt.Printf("✓ Saved to: %s\n", outputPath)

	// Display chapter IDs
	fmt.Printf("\n=== Chapter ID Mapping ===\n")
	chapterDivs.Each(func(i int, s *goquery.Selection) {
		id, _ := s.Attr("id")
		// Get the first heading in this chapter
		heading := s.Find("h1, h2, h3").First().Text()
		if heading != "" {
			heading = strings.TrimSpace(heading)
			if len(heading) > 50 {
				heading = heading[:50] + "..."
			}
			fmt.Printf("%s: %s\n", id, heading)
		} else {
			fmt.Printf("%s: (no heading found)\n", id)
		}
	})

	fmt.Printf("\n✓ Integration test completed successfully!\n")
}
