package converter

import (
	"archive/zip"
	"encoding/binary"
	"os"
	"path/filepath"
	"testing"
)

// readUint16BE reads a big-endian uint16 from data at offset.
func readUint16BE(data []byte, offset int) uint16 {
	return binary.BigEndian.Uint16(data[offset : offset+2])
}

// readUint32BE reads a big-endian uint32 from data at offset.
func readUint32BE(data []byte, offset int) uint32 {
	return binary.BigEndian.Uint32(data[offset : offset+4])
}

// extractRecord extracts record data from AZW3 binary at the given record index.
// PDB record list starts at offset 78; each entry is 8 bytes (4 offset + 4 attrs).
func extractRecord(data []byte, recordIndex int) []byte {
	if len(data) < 78 {
		return nil
	}
	totalRecords := int(readUint16BE(data, 76))
	if recordIndex < 0 || recordIndex >= totalRecords {
		return nil
	}

	listBase := 78
	entryOffset := listBase + recordIndex*8
	if entryOffset+4 > len(data) {
		return nil
	}
	recOffset := int(readUint32BE(data, entryOffset))

	// Determine end of record
	var recEnd int
	if recordIndex+1 < totalRecords {
		nextEntryOffset := listBase + (recordIndex+1)*8
		if nextEntryOffset+4 > len(data) {
			return nil
		}
		recEnd = int(readUint32BE(data, nextEntryOffset))
	} else {
		recEnd = len(data)
	}

	if recOffset >= len(data) || recEnd > len(data) || recEnd < recOffset {
		return nil
	}
	return data[recOffset:recEnd]
}

// createMinimalTestEPUB creates a minimal valid EPUB ZIP file in the given directory.
// Returns the path to the created EPUB file.
func createMinimalTestEPUB(t *testing.T, dir string) string {
	t.Helper()
	epubPath := filepath.Join(dir, "test.epub")
	f, err := os.Create(epubPath)
	if err != nil {
		t.Fatalf("failed to create test EPUB: %v", err)
	}

	w := zip.NewWriter(f)

	// mimetype must be first entry, stored (not compressed)
	header := &zip.FileHeader{
		Name:   "mimetype",
		Method: zip.Store,
	}
	mw, err := w.CreateHeader(header)
	if err != nil {
		t.Fatalf("failed to create mimetype entry: %v", err)
	}
	mw.Write([]byte("application/epub+zip"))

	// META-INF/container.xml
	containerXML := `<?xml version="1.0" encoding="UTF-8"?>
<container version="1.0" xmlns="urn:oasis:names:tc:opendocument:xmlns:container">
  <rootfiles>
    <rootfile full-path="OEBPS/content.opf" media-type="application/oebps-package+xml"/>
  </rootfiles>
</container>`
	cw, _ := w.Create("META-INF/container.xml")
	cw.Write([]byte(containerXML))

	// OEBPS/content.opf
	opfXML := `<?xml version="1.0" encoding="UTF-8"?>
<package xmlns="http://www.idpf.org/2007/opf" version="3.0" unique-identifier="uid">
  <metadata xmlns:dc="http://purl.org/dc/elements/1.1/">
    <dc:title>Test Book</dc:title>
    <dc:language>en</dc:language>
    <dc:identifier id="uid">urn:uuid:12345</dc:identifier>
  </metadata>
  <manifest>
    <item id="ch1" href="chapter1.xhtml" media-type="application/xhtml+xml"/>
  </manifest>
  <spine>
    <itemref idref="ch1"/>
  </spine>
</package>`
	ow, _ := w.Create("OEBPS/content.opf")
	ow.Write([]byte(opfXML))

	// OEBPS/chapter1.xhtml
	xhtmlContent := `<?xml version="1.0" encoding="UTF-8"?>
<html xmlns="http://www.w3.org/1999/xhtml">
<head><title>Chapter 1</title></head>
<body><h1>Hello World</h1><p>This is a test chapter.</p></body>
</html>`
	xw, _ := w.Create("OEBPS/chapter1.xhtml")
	xw.Write([]byte(xhtmlContent))

	w.Close()
	f.Close()

	return epubPath
}

// createBrokenXHTMLTestEPUB creates an EPUB with one valid and one broken XHTML file.
func createBrokenXHTMLTestEPUB(t *testing.T, dir string) string {
	t.Helper()
	epubPath := filepath.Join(dir, "broken.epub")
	f, err := os.Create(epubPath)
	if err != nil {
		t.Fatalf("failed to create test EPUB: %v", err)
	}

	w := zip.NewWriter(f)

	header := &zip.FileHeader{
		Name:   "mimetype",
		Method: zip.Store,
	}
	mw, _ := w.CreateHeader(header)
	mw.Write([]byte("application/epub+zip"))

	cw, _ := w.Create("META-INF/container.xml")
	cw.Write([]byte(`<?xml version="1.0" encoding="UTF-8"?>
<container version="1.0" xmlns="urn:oasis:names:tc:opendocument:xmlns:container">
  <rootfiles>
    <rootfile full-path="OEBPS/content.opf" media-type="application/oebps-package+xml"/>
  </rootfiles>
</container>`))

	// OPF with two chapters: ch1 (valid), ch2 (missing file)
	ow, _ := w.Create("OEBPS/content.opf")
	ow.Write([]byte(`<?xml version="1.0" encoding="UTF-8"?>
<package xmlns="http://www.idpf.org/2007/opf" version="3.0" unique-identifier="uid">
  <metadata xmlns:dc="http://purl.org/dc/elements/1.1/">
    <dc:title>Broken XHTML Test</dc:title>
    <dc:language>en</dc:language>
    <dc:identifier id="uid">urn:uuid:broken</dc:identifier>
  </metadata>
  <manifest>
    <item id="ch1" href="chapter1.xhtml" media-type="application/xhtml+xml"/>
    <item id="ch2" href="missing.xhtml" media-type="application/xhtml+xml"/>
  </manifest>
  <spine>
    <itemref idref="ch1"/>
    <itemref idref="ch2"/>
  </spine>
</package>`))

	// Only chapter1.xhtml exists (chapter2 is missing)
	xw, _ := w.Create("OEBPS/chapter1.xhtml")
	xw.Write([]byte(`<?xml version="1.0" encoding="UTF-8"?>
<html xmlns="http://www.w3.org/1999/xhtml">
<head><title>Chapter 1</title></head>
<body><h1>Valid Chapter</h1><p>This chapter is valid.</p></body>
</html>`))

	w.Close()
	f.Close()

	return epubPath
}

// createAllBrokenTestEPUB creates an EPUB where all XHTML chapters are missing.
func createAllBrokenTestEPUB(t *testing.T, dir string) string {
	t.Helper()
	epubPath := filepath.Join(dir, "allbroken.epub")
	f, err := os.Create(epubPath)
	if err != nil {
		t.Fatalf("failed to create test EPUB: %v", err)
	}

	w := zip.NewWriter(f)

	header := &zip.FileHeader{
		Name:   "mimetype",
		Method: zip.Store,
	}
	mw, _ := w.CreateHeader(header)
	mw.Write([]byte("application/epub+zip"))

	cw, _ := w.Create("META-INF/container.xml")
	cw.Write([]byte(`<?xml version="1.0" encoding="UTF-8"?>
<container version="1.0" xmlns="urn:oasis:names:tc:opendocument:xmlns:container">
  <rootfiles>
    <rootfile full-path="OEBPS/content.opf" media-type="application/oebps-package+xml"/>
  </rootfiles>
</container>`))

	// OPF with chapters that all point to missing files
	ow, _ := w.Create("OEBPS/content.opf")
	ow.Write([]byte(`<?xml version="1.0" encoding="UTF-8"?>
<package xmlns="http://www.idpf.org/2007/opf" version="3.0" unique-identifier="uid">
  <metadata xmlns:dc="http://purl.org/dc/elements/1.1/">
    <dc:title>All Broken Test</dc:title>
    <dc:language>en</dc:language>
    <dc:identifier id="uid">urn:uuid:allbroken</dc:identifier>
  </metadata>
  <manifest>
    <item id="ch1" href="missing1.xhtml" media-type="application/xhtml+xml"/>
    <item id="ch2" href="missing2.xhtml" media-type="application/xhtml+xml"/>
  </manifest>
  <spine>
    <itemref idref="ch1"/>
    <itemref idref="ch2"/>
  </spine>
</package>`))

	w.Close()
	f.Close()

	return epubPath
}

func TestPipeline_Convert_FileNotFound(t *testing.T) {
	dir := t.TempDir()
	outputPath := filepath.Join(dir, "output.azw3")

	p := NewPipeline(ConvertOptions{
		InputPath:  filepath.Join(dir, "nonexistent.epub"),
		OutputPath: outputPath,
	})

	err := p.Convert()
	if err == nil {
		t.Fatal("expected error for nonexistent file, got nil")
	}
}

func TestPipeline_Convert_InvalidFile(t *testing.T) {
	dir := t.TempDir()

	// Create a non-EPUB file
	invalidPath := filepath.Join(dir, "invalid.epub")
	os.WriteFile(invalidPath, []byte("not a zip file"), 0o644)

	outputPath := filepath.Join(dir, "output.azw3")
	p := NewPipeline(ConvertOptions{
		InputPath:  invalidPath,
		OutputPath: outputPath,
	})

	err := p.Convert()
	if err == nil {
		t.Fatal("expected error for invalid file, got nil")
	}
}

func TestPipeline_Convert_MinimalEPUB(t *testing.T) {
	dir := t.TempDir()
	epubPath := createMinimalTestEPUB(t, dir)
	outputPath := filepath.Join(dir, "output.azw3")

	p := NewPipeline(ConvertOptions{
		InputPath:  epubPath,
		OutputPath: outputPath,
	})

	err := p.Convert()
	if err != nil {
		t.Fatalf("Convert() failed: %v", err)
	}

	// Verify output file exists and is non-empty
	info, err := os.Stat(outputPath)
	if err != nil {
		t.Fatalf("output file not found: %v", err)
	}
	if info.Size() == 0 {
		t.Fatal("output file is empty")
	}
}

func TestPipeline_Convert_PDBHeader(t *testing.T) {
	dir := t.TempDir()
	epubPath := createMinimalTestEPUB(t, dir)
	outputPath := filepath.Join(dir, "output.azw3")

	p := NewPipeline(ConvertOptions{
		InputPath:  epubPath,
		OutputPath: outputPath,
	})
	if err := p.Convert(); err != nil {
		t.Fatalf("Convert() failed: %v", err)
	}

	data, err := os.ReadFile(outputPath)
	if err != nil {
		t.Fatalf("failed to read output: %v", err)
	}

	// PDB Type at offset 60-64: "BOOK"
	if len(data) < 68 {
		t.Fatal("output too small for PDB header")
	}
	pdbType := string(data[60:64])
	if pdbType != "BOOK" {
		t.Errorf("PDB type = %q, want %q", pdbType, "BOOK")
	}

	// PDB Creator at offset 64-68: "MOBI"
	pdbCreator := string(data[64:68])
	if pdbCreator != "MOBI" {
		t.Errorf("PDB creator = %q, want %q", pdbCreator, "MOBI")
	}
}

func TestPipeline_Convert_TextRecordExists(t *testing.T) {
	dir := t.TempDir()
	epubPath := createMinimalTestEPUB(t, dir)
	outputPath := filepath.Join(dir, "output.azw3")

	p := NewPipeline(ConvertOptions{
		InputPath:  epubPath,
		OutputPath: outputPath,
	})
	if err := p.Convert(); err != nil {
		t.Fatalf("Convert() failed: %v", err)
	}

	data, err := os.ReadFile(outputPath)
	if err != nil {
		t.Fatalf("failed to read output: %v", err)
	}

	// Record 1 is the first text record; it should not be empty
	rec1 := extractRecord(data, 1)
	if len(rec1) == 0 {
		t.Fatal("text record (record 1) is empty")
	}
}

func TestPipeline_Convert_FixedRecords(t *testing.T) {
	dir := t.TempDir()
	epubPath := createMinimalTestEPUB(t, dir)
	outputPath := filepath.Join(dir, "output.azw3")

	p := NewPipeline(ConvertOptions{
		InputPath:  epubPath,
		OutputPath: outputPath,
	})
	if err := p.Convert(); err != nil {
		t.Fatalf("Convert() failed: %v", err)
	}

	data, err := os.ReadFile(outputPath)
	if err != nil {
		t.Fatalf("failed to read output: %v", err)
	}

	// Get total record count from PDB header (offset 76, 2 bytes big-endian)
	totalRecords := int(readUint16BE(data, 76))
	if totalRecords < 4 {
		t.Fatalf("too few records: %d", totalRecords)
	}

	// FLIS is at totalRecords-3
	flisRec := extractRecord(data, totalRecords-3)
	if len(flisRec) < 4 || string(flisRec[:4]) != "FLIS" {
		t.Errorf("FLIS magic not found, got %q", string(flisRec[:4]))
	}

	// FCIS is at totalRecords-2
	fcisRec := extractRecord(data, totalRecords-2)
	if len(fcisRec) < 4 || string(fcisRec[:4]) != "FCIS" {
		t.Errorf("FCIS magic not found, got %q", string(fcisRec[:4]))
	}

	// EOF is at totalRecords-1
	eofRec := extractRecord(data, totalRecords-1)
	if len(eofRec) < 4 {
		t.Fatalf("EOF record too short: %d bytes", len(eofRec))
	}
	eofMagic := readUint32BE(eofRec, 0)
	if eofMagic != 0xE98E0D0A {
		t.Errorf("EOF magic = 0x%08X, want 0xE98E0D0A", eofMagic)
	}
}

func TestPipeline_Convert_MOBIHeader(t *testing.T) {
	dir := t.TempDir()
	epubPath := createMinimalTestEPUB(t, dir)
	outputPath := filepath.Join(dir, "output.azw3")

	p := NewPipeline(ConvertOptions{
		InputPath:  epubPath,
		OutputPath: outputPath,
	})
	if err := p.Convert(); err != nil {
		t.Fatalf("Convert() failed: %v", err)
	}

	data, err := os.ReadFile(outputPath)
	if err != nil {
		t.Fatalf("failed to read output: %v", err)
	}

	// Record 0 contains the MOBI header
	rec0 := extractRecord(data, 0)
	if len(rec0) < 20 {
		t.Fatal("Record 0 too short")
	}

	// MOBI magic at offset 16 in Record 0
	mobiMagic := string(rec0[16:20])
	if mobiMagic != "MOBI" {
		t.Errorf("MOBI magic = %q, want %q", mobiMagic, "MOBI")
	}

	// Encoding at offset 28 (uint32 BE) should be 65001 (UTF-8)
	if len(rec0) < 32 {
		t.Fatal("Record 0 too short for encoding field")
	}
	encoding := readUint32BE(rec0, 28)
	if encoding != 65001 {
		t.Errorf("encoding = %d, want 65001", encoding)
	}
}

func TestPipeline_Convert_XHTMLReadError_Skips(t *testing.T) {
	dir := t.TempDir()
	epubPath := createBrokenXHTMLTestEPUB(t, dir)
	outputPath := filepath.Join(dir, "output.azw3")

	p := NewPipeline(ConvertOptions{
		InputPath:  epubPath,
		OutputPath: outputPath,
	})

	// Should succeed with warning (skipping broken chapter)
	err := p.Convert()
	if err != nil {
		t.Fatalf("Convert() failed: %v (expected success with skipped chapter)", err)
	}

	// Verify output file exists
	info, err := os.Stat(outputPath)
	if err != nil {
		t.Fatalf("output file not found: %v", err)
	}
	if info.Size() == 0 {
		t.Fatal("output file is empty")
	}
}

func TestPipeline_Convert_NoValidChapters(t *testing.T) {
	dir := t.TempDir()
	epubPath := createAllBrokenTestEPUB(t, dir)
	outputPath := filepath.Join(dir, "output.azw3")

	p := NewPipeline(ConvertOptions{
		InputPath:  epubPath,
		OutputPath: outputPath,
	})

	err := p.Convert()
	if err == nil {
		t.Fatal("expected error when all chapters are invalid, got nil")
	}
}

func TestPipeline_Convert_WithTestdataEPUB(t *testing.T) {
	// Use the project's testdata/test.epub for an E2E test
	epubPath := filepath.Join("..", "..", "testdata", "test.epub")
	if _, err := os.Stat(epubPath); os.IsNotExist(err) {
		t.Skip("testdata/test.epub not found, skipping E2E test")
	}

	dir := t.TempDir()
	outputPath := filepath.Join(dir, "test.azw3")

	p := NewPipeline(ConvertOptions{
		InputPath:  epubPath,
		OutputPath: outputPath,
	})

	err := p.Convert()
	if err != nil {
		t.Fatalf("Convert() failed: %v", err)
	}

	data, err := os.ReadFile(outputPath)
	if err != nil {
		t.Fatalf("failed to read output: %v", err)
	}

	// Basic validation: PDB header
	if len(data) < 68 {
		t.Fatal("output too small for PDB header")
	}
	pdbType := string(data[60:64])
	if pdbType != "BOOK" {
		t.Errorf("PDB type = %q, want %q", pdbType, "BOOK")
	}
	pdbCreator := string(data[64:68])
	if pdbCreator != "MOBI" {
		t.Errorf("PDB creator = %q, want %q", pdbCreator, "MOBI")
	}

	// Verify MOBI magic in Record 0
	rec0 := extractRecord(data, 0)
	if len(rec0) >= 20 {
		mobiMagic := string(rec0[16:20])
		if mobiMagic != "MOBI" {
			t.Errorf("MOBI magic = %q, want %q", mobiMagic, "MOBI")
		}
	}

	// Verify text record exists
	rec1 := extractRecord(data, 1)
	if len(rec1) == 0 {
		t.Error("text record (record 1) is empty")
	}
}
