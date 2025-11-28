package epub

import (
	"archive/zip"
	"os"
	"path/filepath"
	"testing"
)

// createTestEPUB creates a minimal valid EPUB file for testing
func createTestEPUB(t *testing.T, dir string) string {
	t.Helper()
	epubPath := filepath.Join(dir, "test.epub")
	f, err := os.Create(epubPath)
	if err != nil {
		t.Fatalf("failed to create test epub: %v", err)
	}
	defer f.Close()

	w := zip.NewWriter(f)
	defer w.Close()

	// mimetype (must be uncompressed/stored)
	mw, err := w.CreateHeader(&zip.FileHeader{
		Name:   "mimetype",
		Method: zip.Store,
	})
	if err != nil {
		t.Fatalf("failed to create mimetype: %v", err)
	}
	mw.Write([]byte("application/epub+zip"))

	// META-INF/container.xml
	cw, err := w.Create("META-INF/container.xml")
	if err != nil {
		t.Fatalf("failed to create container.xml: %v", err)
	}
	cw.Write([]byte(`<?xml version="1.0" encoding="UTF-8"?>
<container version="1.0" xmlns="urn:oasis:names:tc:opendocument:xmlns:container">
  <rootfiles>
    <rootfile full-path="OEBPS/content.opf" media-type="application/oebps-package+xml"/>
  </rootfiles>
</container>`))

	// OEBPS/content.opf (minimal)
	ow, err := w.Create("OEBPS/content.opf")
	if err != nil {
		t.Fatalf("failed to create content.opf: %v", err)
	}
	ow.Write([]byte(`<?xml version="1.0" encoding="UTF-8"?>
<package xmlns="http://www.idpf.org/2007/opf" version="2.0">
  <metadata xmlns:dc="http://purl.org/dc/elements/1.1/">
    <dc:title>Test Book</dc:title>
    <dc:language>en</dc:language>
  </metadata>
  <manifest>
    <item id="chapter1" href="chapter1.xhtml" media-type="application/xhtml+xml"/>
  </manifest>
  <spine>
    <itemref idref="chapter1"/>
  </spine>
</package>`))

	// OEBPS/chapter1.xhtml
	chw, err := w.Create("OEBPS/chapter1.xhtml")
	if err != nil {
		t.Fatalf("failed to create chapter1.xhtml: %v", err)
	}
	chw.Write([]byte(`<?xml version="1.0" encoding="UTF-8"?>
<html xmlns="http://www.w3.org/1999/xhtml">
<head><title>Chapter 1</title></head>
<body><h1>Chapter 1</h1><p>Hello, World!</p></body>
</html>`))

	return epubPath
}

// createInvalidMimetypeEPUB creates an EPUB with wrong mimetype content
func createInvalidMimetypeEPUB(t *testing.T, dir string) string {
	t.Helper()
	epubPath := filepath.Join(dir, "invalid_mimetype.epub")
	f, err := os.Create(epubPath)
	if err != nil {
		t.Fatalf("failed to create test epub: %v", err)
	}
	defer f.Close()

	w := zip.NewWriter(f)
	defer w.Close()

	mw, err := w.CreateHeader(&zip.FileHeader{
		Name:   "mimetype",
		Method: zip.Store,
	})
	if err != nil {
		t.Fatalf("failed to create mimetype: %v", err)
	}
	mw.Write([]byte("text/plain"))

	return epubPath
}

// createCompressedMimetypeEPUB creates an EPUB with compressed mimetype
func createCompressedMimetypeEPUB(t *testing.T, dir string) string {
	t.Helper()
	epubPath := filepath.Join(dir, "compressed_mimetype.epub")
	f, err := os.Create(epubPath)
	if err != nil {
		t.Fatalf("failed to create test epub: %v", err)
	}
	defer f.Close()

	w := zip.NewWriter(f)
	defer w.Close()

	// mimetype with deflate compression (invalid)
	mw, err := w.CreateHeader(&zip.FileHeader{
		Name:   "mimetype",
		Method: zip.Deflate,
	})
	if err != nil {
		t.Fatalf("failed to create mimetype: %v", err)
	}
	mw.Write([]byte("application/epub+zip"))

	return epubPath
}

// createNoContainerEPUB creates an EPUB without container.xml
func createNoContainerEPUB(t *testing.T, dir string) string {
	t.Helper()
	epubPath := filepath.Join(dir, "no_container.epub")
	f, err := os.Create(epubPath)
	if err != nil {
		t.Fatalf("failed to create test epub: %v", err)
	}
	defer f.Close()

	w := zip.NewWriter(f)
	defer w.Close()

	mw, err := w.CreateHeader(&zip.FileHeader{
		Name:   "mimetype",
		Method: zip.Store,
	})
	if err != nil {
		t.Fatalf("failed to create mimetype: %v", err)
	}
	mw.Write([]byte("application/epub+zip"))

	return epubPath
}

func TestOpen(t *testing.T) {
	dir := t.TempDir()
	epubPath := createTestEPUB(t, dir)

	reader, err := Open(epubPath)
	if err != nil {
		t.Fatalf("Open() failed: %v", err)
	}
	defer reader.Close()

	if reader == nil {
		t.Fatal("Open() returned nil reader")
	}
}

func TestOpen_FileNotFound(t *testing.T) {
	_, err := Open("/nonexistent/file.epub")
	if err == nil {
		t.Fatal("Open() should fail for nonexistent file")
	}
}

func TestOpen_InvalidMimetype(t *testing.T) {
	dir := t.TempDir()
	epubPath := createInvalidMimetypeEPUB(t, dir)

	_, err := Open(epubPath)
	if err == nil {
		t.Fatal("Open() should fail for invalid mimetype")
	}
}

func TestOpen_CompressedMimetype(t *testing.T) {
	dir := t.TempDir()
	epubPath := createCompressedMimetypeEPUB(t, dir)

	_, err := Open(epubPath)
	if err == nil {
		t.Fatal("Open() should fail for compressed mimetype")
	}
}

func TestOpen_NoContainer(t *testing.T) {
	dir := t.TempDir()
	epubPath := createNoContainerEPUB(t, dir)

	_, err := Open(epubPath)
	if err == nil {
		t.Fatal("Open() should fail when container.xml is missing")
	}
}

func TestEPUBReader_OPFPath(t *testing.T) {
	dir := t.TempDir()
	epubPath := createTestEPUB(t, dir)

	reader, err := Open(epubPath)
	if err != nil {
		t.Fatalf("Open() failed: %v", err)
	}
	defer reader.Close()

	expected := "OEBPS/content.opf"
	if reader.OPFPath() != expected {
		t.Errorf("OPFPath() = %q, want %q", reader.OPFPath(), expected)
	}
}

func TestEPUBReader_Files(t *testing.T) {
	dir := t.TempDir()
	epubPath := createTestEPUB(t, dir)

	reader, err := Open(epubPath)
	if err != nil {
		t.Fatalf("Open() failed: %v", err)
	}
	defer reader.Close()

	files := reader.Files()

	expectedFiles := []string{
		"mimetype",
		"META-INF/container.xml",
		"OEBPS/content.opf",
		"OEBPS/chapter1.xhtml",
	}

	for _, name := range expectedFiles {
		if _, ok := files[name]; !ok {
			t.Errorf("Files() missing %q", name)
		}
	}
}

func TestEPUBReader_ReadFile(t *testing.T) {
	dir := t.TempDir()
	epubPath := createTestEPUB(t, dir)

	reader, err := Open(epubPath)
	if err != nil {
		t.Fatalf("Open() failed: %v", err)
	}
	defer reader.Close()

	content, err := reader.ReadFile("mimetype")
	if err != nil {
		t.Fatalf("ReadFile() failed: %v", err)
	}

	expected := "application/epub+zip"
	if string(content) != expected {
		t.Errorf("ReadFile() = %q, want %q", string(content), expected)
	}
}

func TestEPUBReader_ReadFile_NotFound(t *testing.T) {
	dir := t.TempDir()
	epubPath := createTestEPUB(t, dir)

	reader, err := Open(epubPath)
	if err != nil {
		t.Fatalf("Open() failed: %v", err)
	}
	defer reader.Close()

	_, err = reader.ReadFile("nonexistent.txt")
	if err == nil {
		t.Fatal("ReadFile() should fail for nonexistent file")
	}
}

// Test path normalization (handling of ./ prefix)
func TestOpen_PathNormalization(t *testing.T) {
	dir := t.TempDir()
	epubPath := filepath.Join(dir, "normalized.epub")
	f, err := os.Create(epubPath)
	if err != nil {
		t.Fatalf("failed to create test epub: %v", err)
	}

	w := zip.NewWriter(f)

	// mimetype
	mw, _ := w.CreateHeader(&zip.FileHeader{
		Name:   "mimetype",
		Method: zip.Store,
	})
	mw.Write([]byte("application/epub+zip"))

	// container.xml with ./ prefix in path
	cw, _ := w.Create("META-INF/container.xml")
	cw.Write([]byte(`<?xml version="1.0" encoding="UTF-8"?>
<container version="1.0" xmlns="urn:oasis:names:tc:opendocument:xmlns:container">
  <rootfiles>
    <rootfile full-path="./OEBPS/content.opf" media-type="application/oebps-package+xml"/>
  </rootfiles>
</container>`))

	// content.opf
	ow, _ := w.Create("OEBPS/content.opf")
	ow.Write([]byte(`<?xml version="1.0" encoding="UTF-8"?>
<package xmlns="http://www.idpf.org/2007/opf" version="2.0">
  <metadata xmlns:dc="http://purl.org/dc/elements/1.1/">
    <dc:title>Test</dc:title>
    <dc:language>en</dc:language>
  </metadata>
  <manifest></manifest>
  <spine></spine>
</package>`))

	w.Close()
	f.Close()

	reader, err := Open(epubPath)
	if err != nil {
		t.Fatalf("Open() failed: %v", err)
	}
	defer reader.Close()

	// Should normalize ./OEBPS/content.opf to OEBPS/content.opf
	expected := "OEBPS/content.opf"
	if reader.OPFPath() != expected {
		t.Errorf("OPFPath() = %q, want %q (path should be normalized)", reader.OPFPath(), expected)
	}
}
