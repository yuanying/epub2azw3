// Test program for EPUB ZIP reader functionality
// Related: Issue #1, PR #10
//
// Usage:
//
//	go run ./cmd/test/epub_reader/main.go <epub-file-path>
//
// This program tests the following functionality:
// - Opening EPUB files (ZIP archive)
// - Validating mimetype file
// - Extracting OPF path from container.xml
// - Listing all files in the EPUB
// - Reading file contents
package main

import (
	"fmt"
	"log"
	"os"

	"github.com/yuanying/epub2azw3/internal/epub"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Println("Usage: go run ./cmd/test/epub_reader/main.go <epub-file> (<content-filename> ...)")
		os.Exit(1)
	}

	epubPath := os.Args[1]
	filePaths := os.Args[2:]

	// EPUBファイルを開く
	fmt.Printf("Opening EPUB file: %s\n", epubPath)
	reader, err := epub.Open(epubPath)
	if err != nil {
		log.Fatalf("Failed to open EPUB: %v", err)
	}
	defer reader.Close()

	// OPFパスを表示
	fmt.Printf("✓ EPUB opened successfully\n")
	fmt.Printf("OPF Path: %s\n\n", reader.OPFPath())

	// ファイル一覧を表示
	files := reader.Files()
	fmt.Printf("Total files: %d\n", len(files))
	fmt.Println("\nFile list:")
	for name := range files {
		fmt.Printf("  - %s\n", name)
	}

	// OPFファイルを読み込んでみる
	fmt.Println("\nReading OPF file...")
	opfContent, err := reader.ReadFile(reader.OPFPath())
	if err != nil {
		log.Fatalf("Failed to read OPF: %v", err)
	}
	fmt.Printf("✓ OPF file read successfully (%d bytes)\n", len(opfContent))

	// container.xmlを読み込んでみる
	fmt.Println("\nReading META-INF/container.xml...")
	containerContent, err := reader.ReadFile("META-INF/container.xml")
	if err != nil {
		log.Fatalf("Failed to read container.xml: %v", err)
	}
	fmt.Printf("✓ container.xml read successfully (%d bytes)\n", len(containerContent))

	// 指定されたコンテンツファイルを読み込む
	for _, filePath := range filePaths {
		fmt.Printf("\nReading content file: %s\n", filePath)
		content, err := reader.ReadFile(filePath)
		if err != nil {
			log.Fatalf("Failed to read content file %s: %v", filePath, err)
		}
		fmt.Printf("✓ Content file %s read successfully (%d bytes)\n", filePath, len(content))
		fmt.Printf("Content:\n%s\n", string(content))
	}

	fmt.Println("\n✓ All tests passed!")
}
