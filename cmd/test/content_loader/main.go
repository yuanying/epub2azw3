// Test program for XHTML content loading functionality
// Related: Issue #3, PR #12
//
// Usage:
//   go run ./cmd/test/content_loader/main.go <epub-file-path>
//
// This program:
// 1. Opens the specified EPUB file
// 2. Parses the OPF file to get spine items
// 3. Loads each XHTML content file using LoadContent
// 4. Displays CSS links and image references found in each file
//
// Verification points:
// - ✓ XHTML files are loaded without errors
// - ✓ CSS references are correctly collected
// - ✓ Image references are correctly collected
// - ✓ Relative paths are correctly resolved
// - ✓ Document structure is accessible

package main

import (
	"fmt"
	"log"
	"os"
	"path/filepath"

	"github.com/yuanying/epub2azw3/internal/epub"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Fprintf(os.Stderr, "Usage: %s <epub-file-path>\n", filepath.Base(os.Args[0]))
		os.Exit(1)
	}

	epubPath := os.Args[1]

	fmt.Printf("=== XHTML Content Loader Test ===\n")
	fmt.Printf("EPUB file: %s\n\n", epubPath)

	// Open EPUB file
	reader, err := epub.Open(epubPath)
	if err != nil {
		log.Fatalf("Failed to open EPUB: %v", err)
	}
	defer reader.Close()

	fmt.Printf("✓ EPUB opened successfully\n")
	fmt.Printf("OPF path: %s\n\n", reader.OPFPath())

	// Read and parse OPF
	opfContent, err := reader.ReadFile(reader.OPFPath())
	if err != nil {
		log.Fatalf("Failed to read OPF: %v", err)
	}

	opfDir := filepath.Dir(reader.OPFPath())
	opf, err := epub.ParseOPF(opfContent, opfDir)
	if err != nil {
		log.Fatalf("Failed to parse OPF: %v", err)
	}

	fmt.Printf("✓ OPF parsed successfully\n")
	fmt.Printf("Title: %s\n", opf.Metadata.Title)
	fmt.Printf("Spine items: %d\n\n", len(opf.Spine))

	// Load each spine item
	fmt.Println("=== Loading Spine Items ===")

	successCount := 0
	errorCount := 0
	totalCSS := 0
	totalImages := 0

	for i, spineItem := range opf.Spine {
		manifestItem, ok := opf.Manifest[spineItem.IDRef]
		if !ok {
			fmt.Printf("[%d] ⚠ Spine item '%s' not found in manifest\n\n", i+1, spineItem.IDRef)
			errorCount++
			continue
		}

		// Only process XHTML files
		if manifestItem.MediaType != "application/xhtml+xml" {
			fmt.Printf("[%d] ⏭ Skipping non-XHTML item: %s (%s)\n\n",
				i+1, spineItem.IDRef, manifestItem.MediaType)
			continue
		}

		fmt.Printf("[%d] Processing: %s\n", i+1, spineItem.IDRef)
		fmt.Printf("    Path: %s\n", manifestItem.Href)

		// Read XHTML content
		xhtmlContent, err := reader.ReadFile(manifestItem.Href)
		if err != nil {
			fmt.Printf("    ✗ Failed to read file: %v\n\n", err)
			errorCount++
			continue
		}

		// Load content
		content, err := epub.LoadContent(spineItem.IDRef, manifestItem.Href, xhtmlContent)
		if err != nil {
			fmt.Printf("    ✗ Failed to load content: %v\n\n", err)
			errorCount++
			continue
		}

		fmt.Printf("    ✓ Content loaded successfully\n")

		// Display CSS links
		if len(content.CSSLinks) > 0 {
			fmt.Printf("    CSS links (%d):\n", len(content.CSSLinks))
			for _, css := range content.CSSLinks {
				fmt.Printf("      - %s\n", css)
			}
			totalCSS += len(content.CSSLinks)
		} else {
			fmt.Printf("    CSS links: none\n")
		}

		// Display image references
		if len(content.ImageRefs) > 0 {
			fmt.Printf("    Images (%d):\n", len(content.ImageRefs))
			for _, img := range content.ImageRefs {
				fmt.Printf("      - %s\n", img)
			}
			totalImages += len(content.ImageRefs)
		} else {
			fmt.Printf("    Images: none\n")
		}

		// Display document structure info
		if content.Document != nil {
			bodyText := content.Document.Find("body").Text()
			textLength := len(bodyText)
			fmt.Printf("    Body text length: %d characters\n", textLength)

			// Count some common elements
			h1Count := content.Document.Find("h1").Length()
			h2Count := content.Document.Find("h2").Length()
			pCount := content.Document.Find("p").Length()

			if h1Count > 0 || h2Count > 0 {
				fmt.Printf("    Headings: h1=%d, h2=%d\n", h1Count, h2Count)
			}
			if pCount > 0 {
				fmt.Printf("    Paragraphs: %d\n", pCount)
			}
		}

		fmt.Println()
		successCount++
	}

	// Summary
	fmt.Println("=== Summary ===")
	fmt.Printf("Successfully loaded: %d files\n", successCount)
	if errorCount > 0 {
		fmt.Printf("Errors: %d files\n", errorCount)
	}
	fmt.Printf("Total CSS references: %d\n", totalCSS)
	fmt.Printf("Total image references: %d\n", totalImages)

	if errorCount > 0 {
		os.Exit(1)
	}
}
