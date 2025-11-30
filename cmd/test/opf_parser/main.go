// Test program for OPF parser functionality
// Related: Issue #2, PR #11
//
// Usage:
//   go run ./cmd/test/opf_parser/main.go <epub-file-path>
//
// Example:
//   go run ./cmd/test/opf_parser/main.go ~/Downloads/sample.epub
//
// This program will:
// - Open the EPUB file
// - Parse the OPF file
// - Display metadata (title, authors, language, etc.)
// - List manifest items
// - Show spine order
// - Display NCX path
// - Show cover image if found

package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/yuanying/epub2azw3/internal/epub"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Fprintf(os.Stderr, "Usage: %s <epub-file-path>\n", os.Args[0])
		os.Exit(1)
	}

	epubPath := os.Args[1]

	fmt.Println("=== EPUB OPF Parser Test ===")
	fmt.Printf("File: %s\n\n", epubPath)

	// Open EPUB file
	reader, err := epub.Open(epubPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error opening EPUB: %v\n", err)
		os.Exit(1)
	}
	defer reader.Close()

	fmt.Printf("✓ EPUB opened successfully\n")
	fmt.Printf("OPF Path: %s\n\n", reader.OPFPath())

	// Read OPF file
	opfContent, err := reader.ReadFile(reader.OPFPath())
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error reading OPF file: %v\n", err)
		os.Exit(1)
	}

	// Parse OPF
	// Extract directory from OPF path for proper manifest item path resolution
	opfDir := filepath.Dir(reader.OPFPath())
	if opfDir == "." {
		opfDir = ""
	} else {
		opfDir = opfDir + "/"
	}
	opf, err := epub.ParseOPF(opfContent, opfDir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error parsing OPF: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("✓ OPF parsed successfully")

	// Display metadata
	fmt.Println("--- Metadata ---")
	fmt.Printf("Title:       %s\n", opf.Metadata.Title)
	fmt.Printf("Language:    %s\n", opf.Metadata.Language)
	fmt.Printf("Identifier:  %s\n", opf.Metadata.Identifier)

	if len(opf.Metadata.Creators) > 0 {
		fmt.Println("Creators:")
		for i, creator := range opf.Metadata.Creators {
			role := creator.Role
			if role == "" {
				role = "unknown"
			}
			fmt.Printf("  %d. %s (role: %s)\n", i+1, creator.Name, role)
		}
	}

	if opf.Metadata.Publisher != "" {
		fmt.Printf("Publisher:   %s\n", opf.Metadata.Publisher)
	}

	if opf.Metadata.Date != "" {
		fmt.Printf("Date:        %s\n", opf.Metadata.Date)
	}

	if opf.Metadata.Description != "" {
		fmt.Printf("Description: %s\n", opf.Metadata.Description)
	}

	if len(opf.Metadata.Subjects) > 0 {
		fmt.Println("Subjects:")
		for i, subject := range opf.Metadata.Subjects {
			fmt.Printf("  %d. %s\n", i+1, subject)
		}
	}

	if opf.Metadata.Rights != "" {
		fmt.Printf("Rights:      %s\n", opf.Metadata.Rights)
	}

	// Display manifest summary
	fmt.Printf("\n--- Manifest ---\n")
	fmt.Printf("Total items: %d\n\n", len(opf.Manifest))

	// Count by media type
	mediaTypes := make(map[string]int)
	for _, item := range opf.Manifest {
		mediaTypes[item.MediaType]++
	}

	fmt.Println("Items by media type:")
	for mediaType, count := range mediaTypes {
		fmt.Printf("  %s: %d\n", mediaType, count)
	}

	// Display cover image
	coverHref, hasCover := opf.FindCoverImage()
	if hasCover {
		fmt.Printf("\nCover Image: %s\n", coverHref)
	} else {
		fmt.Println("\nCover Image: (not found)")
	}

	// Display spine
	fmt.Printf("\n--- Spine ---\n")
	fmt.Printf("Total items: %d\n\n", len(opf.Spine))

	fmt.Println("Reading order:")
	for i, spineItem := range opf.Spine {
		linear := "yes"
		if !spineItem.Linear {
			linear = "no"
		}

		manifestItem, ok := opf.Manifest[spineItem.IDRef]
		if ok {
			fmt.Printf("  %d. %s (linear: %s)\n", i+1, manifestItem.Href, linear)
		} else {
			fmt.Printf("  %d. [ID: %s - not found in manifest] (linear: %s)\n", i+1, spineItem.IDRef, linear)
		}
	}

	// Display NCX path
	if opf.NCXPath != "" {
		fmt.Printf("\n--- Navigation ---\n")
		fmt.Printf("NCX Path: %s\n", opf.NCXPath)
	}

	// Display some manifest items with properties
	fmt.Println("\n--- Special Items ---")
	hasSpecial := false
	for id, item := range opf.Manifest {
		if len(item.Properties) > 0 {
			if !hasSpecial {
				hasSpecial = true
			}
			fmt.Printf("  %s: %s (properties: %v)\n", id, item.Href, item.Properties)
		}
	}
	if !hasSpecial {
		fmt.Println("  (no items with special properties)")
	}

	fmt.Println("\n=== Test Completed Successfully ===")
}
