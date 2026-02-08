package mobi

import (
	"bytes"
	"encoding/binary"
	"strings"
	"testing"
	"time"

	"github.com/yuanying/epub2azw3/internal/epub"
)

// --- Test helpers ---

func readUint16BE(data []byte, offset int) uint16 {
	return binary.BigEndian.Uint16(data[offset : offset+2])
}

func readUint32BE(data []byte, offset int) uint32 {
	return binary.BigEndian.Uint32(data[offset : offset+4])
}

func generateTestHTML(size int) []byte {
	prefix := "<html><body>"
	suffix := "</body></html>"
	padding := size - len(prefix) - len(suffix)
	if padding < 0 {
		padding = 0
	}
	return []byte(prefix + strings.Repeat("A", padding) + suffix)
}

func writeToBuffer(t *testing.T, w *AZW3Writer) []byte {
	t.Helper()
	var buf bytes.Buffer
	n, err := w.WriteTo(&buf)
	if err != nil {
		t.Fatalf("WriteTo failed: %v", err)
	}
	if n != int64(buf.Len()) {
		t.Fatalf("WriteTo returned %d bytes, but buffer has %d", n, buf.Len())
	}
	return buf.Bytes()
}

// extractRecord extracts the Nth record data from a PDB file binary.
func extractRecord(data []byte, index int) []byte {
	numRecords := int(readUint16BE(data, 76))
	if index >= numRecords {
		return nil
	}
	recordListStart := 78
	entryOffset := recordListStart + index*8
	recOffset := readUint32BE(data, entryOffset)

	// Next record offset or end of file
	var nextOffset uint32
	if index+1 < numRecords {
		nextEntryOffset := recordListStart + (index+1)*8
		nextOffset = readUint32BE(data, nextEntryOffset)
	} else {
		nextOffset = uint32(len(data))
	}
	return data[recOffset:nextOffset]
}

// --- Step 1: Constructor and validation ---

func TestNewAZW3Writer_MinimalConfig(t *testing.T) {
	html := generateTestHTML(100)
	uid := uint32(12345)
	cfg := AZW3WriterConfig{
		Title:    "Test Book",
		HTML:     html,
		UniqueID: &uid,
	}
	w, err := NewAZW3Writer(cfg)
	if err != nil {
		t.Fatalf("NewAZW3Writer failed: %v", err)
	}
	if w == nil {
		t.Fatal("NewAZW3Writer returned nil")
	}
}

func TestNewAZW3Writer_EmptyHTML(t *testing.T) {
	cfg := AZW3WriterConfig{
		Title: "Test Book",
		HTML:  []byte{},
	}
	_, err := NewAZW3Writer(cfg)
	if err == nil {
		t.Fatal("expected error for empty HTML")
	}
}

func TestNewAZW3Writer_NilHTML(t *testing.T) {
	cfg := AZW3WriterConfig{
		Title: "Test Book",
		HTML:  nil,
	}
	_, err := NewAZW3Writer(cfg)
	if err == nil {
		t.Fatal("expected error for nil HTML")
	}
}

func TestNewAZW3Writer_InvalidCompression(t *testing.T) {
	html := generateTestHTML(100)
	cfg := AZW3WriterConfig{
		Title:       "Test Book",
		HTML:        html,
		Compression: 99,
	}
	_, err := NewAZW3Writer(cfg)
	if err == nil {
		t.Fatal("expected error for invalid compression type")
	}
}

func TestNewAZW3Writer_PalmDocCompression(t *testing.T) {
	html := generateTestHTML(100)
	cfg := AZW3WriterConfig{
		Title:       "Test Book",
		HTML:        html,
		Compression: CompressionPalmDoc,
	}
	writer, err := NewAZW3Writer(cfg)
	if err != nil {
		t.Fatalf("NewAZW3Writer with PalmDoc: %v", err)
	}
	var buf bytes.Buffer
	if _, err := writer.WriteTo(&buf); err != nil {
		t.Fatalf("WriteTo with PalmDoc: %v", err)
	}
	if buf.Len() == 0 {
		t.Fatal("WriteTo produced empty output")
	}
}

func TestNewAZW3Writer_CompressionZeroDefaultsToNone(t *testing.T) {
	html := generateTestHTML(100)
	uid := uint32(12345)
	cfg := AZW3WriterConfig{
		Title:       "Test Book",
		HTML:        html,
		Compression: 0,
		UniqueID:    &uid,
	}
	w, err := NewAZW3Writer(cfg)
	if err != nil {
		t.Fatalf("NewAZW3Writer failed: %v", err)
	}
	if w == nil {
		t.Fatal("NewAZW3Writer returned nil")
	}
}

// --- Step 2: WriteTo minimal config ---

func TestWriteTo_MinimalOutput(t *testing.T) {
	html := generateTestHTML(100)
	uid := uint32(12345)
	creation := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	cfg := AZW3WriterConfig{
		Title:        "Test Book",
		HTML:         html,
		UniqueID:     &uid,
		CreationTime: creation,
	}
	w, err := NewAZW3Writer(cfg)
	if err != nil {
		t.Fatalf("NewAZW3Writer failed: %v", err)
	}
	data := writeToBuffer(t, w)

	// Check PDB Type = "BOOK"
	if string(data[60:64]) != "BOOK" {
		t.Errorf("PDB Type: got %q, want %q", string(data[60:64]), "BOOK")
	}
	// Check PDB Creator = "MOBI"
	if string(data[64:68]) != "MOBI" {
		t.Errorf("PDB Creator: got %q, want %q", string(data[64:68]), "MOBI")
	}

	// Record count = 6: Record0 + text×1 + FDST + FLIS + FCIS + EOF
	numRecords := readUint16BE(data, 76)
	if numRecords != 6 {
		t.Errorf("record count: got %d, want 6", numRecords)
	}
}

func TestWriteTo_OutputSizeConsistency(t *testing.T) {
	html := generateTestHTML(100)
	uid := uint32(12345)
	creation := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	cfg := AZW3WriterConfig{
		Title:        "Test Book",
		HTML:         html,
		UniqueID:     &uid,
		CreationTime: creation,
	}
	w, err := NewAZW3Writer(cfg)
	if err != nil {
		t.Fatalf("NewAZW3Writer failed: %v", err)
	}
	data := writeToBuffer(t, w)

	// Output should be non-empty and reasonable
	if len(data) < 78 {
		t.Errorf("output too small: %d bytes", len(data))
	}
}

// --- Step 3: Record offsets and file size integrity ---

func TestWriteTo_RecordOffsets(t *testing.T) {
	html := generateTestHTML(100)
	uid := uint32(12345)
	creation := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	cfg := AZW3WriterConfig{
		Title:        "Test Book",
		HTML:         html,
		UniqueID:     &uid,
		CreationTime: creation,
	}
	w, err := NewAZW3Writer(cfg)
	if err != nil {
		t.Fatalf("NewAZW3Writer failed: %v", err)
	}
	data := writeToBuffer(t, w)

	numRecords := int(readUint16BE(data, 76))
	recordListStart := 78

	// First offset = 78 + 8*recordCount + 2
	expectedFirstOffset := uint32(78 + 8*numRecords + 2)
	firstOffset := readUint32BE(data, recordListStart)
	if firstOffset != expectedFirstOffset {
		t.Errorf("first record offset: got %d, want %d", firstOffset, expectedFirstOffset)
	}

	// Offsets should be monotonically increasing
	prevOffset := firstOffset
	for i := 1; i < numRecords; i++ {
		off := readUint32BE(data, recordListStart+i*8)
		if off <= prevOffset {
			t.Errorf("record %d offset %d not greater than record %d offset %d", i, off, i-1, prevOffset)
		}
		prevOffset = off
	}

	// Last offset + last record size = total file size
	lastOffset := readUint32BE(data, recordListStart+(numRecords-1)*8)
	lastRecordSize := uint32(len(data)) - lastOffset
	if lastOffset+lastRecordSize != uint32(len(data)) {
		t.Errorf("file size mismatch: lastOffset(%d)+lastRecordSize(%d)=%d, fileSize=%d",
			lastOffset, lastRecordSize, lastOffset+lastRecordSize, len(data))
	}
}

// --- Step 4: MOBI header record number references ---

func TestWriteTo_MOBIHeaderRecordNumbers(t *testing.T) {
	html := generateTestHTML(100)
	uid := uint32(12345)
	creation := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	cfg := AZW3WriterConfig{
		Title:        "Test Book",
		HTML:         html,
		UniqueID:     &uid,
		CreationTime: creation,
	}
	w, err := NewAZW3Writer(cfg)
	if err != nil {
		t.Fatalf("NewAZW3Writer failed: %v", err)
	}
	data := writeToBuffer(t, w)

	// Record 0 starts at firstOffset
	rec0 := extractRecord(data, 0)
	// PalmDOC header is 16 bytes, then MOBI header starts
	mobiStart := 16

	// MOBI header offset 160 (relative to MOBI start): FirstContentRecord (uint16)
	firstContent := readUint16BE(rec0, mobiStart+160)
	if firstContent != 1 {
		t.Errorf("FirstContentRecord: got %d, want 1", firstContent)
	}

	// MOBI header offset 162 (relative to MOBI start): LastContentRecord (uint16)
	lastContent := readUint16BE(rec0, mobiStart+162)
	// With 100-byte HTML, expect 1 text record
	if lastContent != 1 {
		t.Errorf("LastContentRecord: got %d, want 1", lastContent)
	}

	// MOBI header offset 176 (relative to MOBI start): FLISRecordNumber (uint32)
	flisNum := readUint32BE(rec0, mobiStart+176)
	// Record order: 0=Record0, 1=text, 2=FDST, 3=FLIS, 4=FCIS, 5=EOF
	if flisNum != 3 {
		t.Errorf("FLISRecordNumber: got %d, want 3", flisNum)
	}

	// MOBI header offset 168 (relative to MOBI start): FCISRecordNumber (uint32)
	fcisNum := readUint32BE(rec0, mobiStart+168)
	if fcisNum != 4 {
		t.Errorf("FCISRecordNumber: got %d, want 4", fcisNum)
	}

	// MOBI header offset 236 (relative to MOBI start): FDSTFlowCount (uint32)
	fdstFlowCount := readUint32BE(rec0, mobiStart+236)
	if fdstFlowCount != 1 {
		t.Errorf("FDSTFlowCount: got %d, want 1", fdstFlowCount)
	}
}

func TestWriteTo_FirstImageIndexNoImages(t *testing.T) {
	html := generateTestHTML(100)
	uid := uint32(12345)
	creation := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	cfg := AZW3WriterConfig{
		Title:        "Test Book",
		HTML:         html,
		UniqueID:     &uid,
		CreationTime: creation,
	}
	w, err := NewAZW3Writer(cfg)
	if err != nil {
		t.Fatalf("NewAZW3Writer failed: %v", err)
	}
	data := writeToBuffer(t, w)

	rec0 := extractRecord(data, 0)
	mobiStart := 16
	firstImageIdx := readUint32BE(rec0, mobiStart+80)
	if firstImageIdx != 0xFFFFFFFF {
		t.Errorf("FirstImageIndex (no images): got 0x%08X, want 0xFFFFFFFF", firstImageIdx)
	}
}

// --- Step 5: EXTH record values ---

func TestWriteTo_EXTHValues(t *testing.T) {
	html := generateTestHTML(100)
	uid := uint32(12345)
	creation := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	cfg := AZW3WriterConfig{
		Title:        "Test Book",
		HTML:         html,
		UniqueID:     &uid,
		CreationTime: creation,
	}
	w, err := NewAZW3Writer(cfg)
	if err != nil {
		t.Fatalf("NewAZW3Writer failed: %v", err)
	}
	data := writeToBuffer(t, w)

	rec0 := extractRecord(data, 0)
	// PalmDOC(16) + MOBI(248) = 264 -> EXTH starts at 264
	exthStart := 16 + MOBIHeaderSize

	// Verify EXTH magic
	if string(rec0[exthStart:exthStart+4]) != "EXTH" {
		t.Fatalf("EXTH magic: got %q, want %q", string(rec0[exthStart:exthStart+4]), "EXTH")
	}

	// Parse EXTH records to find types 121 and 125
	exthRecordCount := readUint32BE(rec0, exthStart+8)
	offset := exthStart + 12

	var boundary, recordCount uint32
	var foundBoundary, foundRecordCount bool

	for i := 0; i < int(exthRecordCount); i++ {
		recType := readUint32BE(rec0, offset)
		recLen := readUint32BE(rec0, offset+4)
		if recType == 121 {
			boundary = readUint32BE(rec0, offset+8)
			foundBoundary = true
		}
		if recType == 125 {
			recordCount = readUint32BE(rec0, offset+8)
			foundRecordCount = true
		}
		offset += int(recLen)
	}

	if !foundBoundary {
		t.Fatal("EXTH type 121 (boundary) not found")
	}
	if boundary != 0 {
		t.Errorf("EXTH 121 (boundary): got %d, want 0 (KF8-only)", boundary)
	}

	if !foundRecordCount {
		t.Fatal("EXTH type 125 (record count) not found")
	}
	// Total PDB records = 6
	if recordCount != 6 {
		t.Errorf("EXTH 125 (record count): got %d, want 6", recordCount)
	}
}

// --- Step 6: Record data content verification ---

func TestWriteTo_TextRecordContent(t *testing.T) {
	html := generateTestHTML(100)
	uid := uint32(12345)
	creation := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	cfg := AZW3WriterConfig{
		Title:        "Test Book",
		HTML:         html,
		UniqueID:     &uid,
		CreationTime: creation,
	}
	w, err := NewAZW3Writer(cfg)
	if err != nil {
		t.Fatalf("NewAZW3Writer failed: %v", err)
	}
	data := writeToBuffer(t, w)

	// Record 1 = text record (only one for small HTML)
	textRec := extractRecord(data, 1)
	if !bytes.Equal(textRec, html) {
		t.Errorf("text record does not match original HTML: got %d bytes, want %d", len(textRec), len(html))
	}
}

func TestWriteTo_FDSTContent(t *testing.T) {
	html := generateTestHTML(100)
	uid := uint32(12345)
	creation := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	cfg := AZW3WriterConfig{
		Title:        "Test Book",
		HTML:         html,
		UniqueID:     &uid,
		CreationTime: creation,
	}
	w, err := NewAZW3Writer(cfg)
	if err != nil {
		t.Fatalf("NewAZW3Writer failed: %v", err)
	}
	data := writeToBuffer(t, w)

	// FDST is at index 2 (Record0, text, FDST, FLIS, FCIS, EOF)
	fdstRec := extractRecord(data, 2)
	if string(fdstRec[:4]) != "FDST" {
		t.Errorf("FDST magic: got %q, want %q", string(fdstRec[:4]), "FDST")
	}
}

func TestWriteTo_FLISContent(t *testing.T) {
	html := generateTestHTML(100)
	uid := uint32(12345)
	creation := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	cfg := AZW3WriterConfig{
		Title:        "Test Book",
		HTML:         html,
		UniqueID:     &uid,
		CreationTime: creation,
	}
	w, err := NewAZW3Writer(cfg)
	if err != nil {
		t.Fatalf("NewAZW3Writer failed: %v", err)
	}
	data := writeToBuffer(t, w)

	// FLIS is at index 3
	flisRec := extractRecord(data, 3)
	if string(flisRec[:4]) != "FLIS" {
		t.Errorf("FLIS magic: got %q, want %q", string(flisRec[:4]), "FLIS")
	}
	if len(flisRec) != 36 {
		t.Errorf("FLIS size: got %d, want 36", len(flisRec))
	}
}

func TestWriteTo_FCISContent(t *testing.T) {
	html := generateTestHTML(100)
	uid := uint32(12345)
	creation := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	cfg := AZW3WriterConfig{
		Title:        "Test Book",
		HTML:         html,
		UniqueID:     &uid,
		CreationTime: creation,
	}
	w, err := NewAZW3Writer(cfg)
	if err != nil {
		t.Fatalf("NewAZW3Writer failed: %v", err)
	}
	data := writeToBuffer(t, w)

	// FCIS is at index 4
	fcisRec := extractRecord(data, 4)
	if string(fcisRec[:4]) != "FCIS" {
		t.Errorf("FCIS magic: got %q, want %q", string(fcisRec[:4]), "FCIS")
	}

	// textLength at offset 20 in FCIS
	textLength := readUint32BE(fcisRec, 20)
	if textLength != uint32(len(html)) {
		t.Errorf("FCIS textLength: got %d, want %d", textLength, len(html))
	}
}

func TestWriteTo_EOFContent(t *testing.T) {
	html := generateTestHTML(100)
	uid := uint32(12345)
	creation := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	cfg := AZW3WriterConfig{
		Title:        "Test Book",
		HTML:         html,
		UniqueID:     &uid,
		CreationTime: creation,
	}
	w, err := NewAZW3Writer(cfg)
	if err != nil {
		t.Fatalf("NewAZW3Writer failed: %v", err)
	}
	data := writeToBuffer(t, w)

	// EOF is at index 5
	eofRec := extractRecord(data, 5)
	if len(eofRec) != 4 {
		t.Fatalf("EOF size: got %d, want 4", len(eofRec))
	}
	eofVal := readUint32BE(eofRec, 0)
	if eofVal != 0xE98E0D0A {
		t.Errorf("EOF value: got 0x%08X, want 0xE98E0D0A", eofVal)
	}
}

// --- Step 7: Metadata support ---

func TestWriteTo_WithMetadata(t *testing.T) {
	html := generateTestHTML(100)
	uid := uint32(12345)
	creation := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	meta := &epub.Metadata{
		Title:    "Test Book",
		Creators: []epub.Creator{{Name: "Author"}},
		Language: "ja",
	}
	cfg := AZW3WriterConfig{
		Title:        "Test Book",
		HTML:         html,
		Metadata:     meta,
		UniqueID:     &uid,
		CreationTime: creation,
	}
	w, err := NewAZW3Writer(cfg)
	if err != nil {
		t.Fatalf("NewAZW3Writer failed: %v", err)
	}
	data := writeToBuffer(t, w)

	rec0 := extractRecord(data, 0)
	exthStart := 16 + MOBIHeaderSize

	if string(rec0[exthStart:exthStart+4]) != "EXTH" {
		t.Fatalf("EXTH magic missing")
	}

	exthRecordCount := readUint32BE(rec0, exthStart+8)
	offset := exthStart + 12

	var foundAuthor, foundTitle503, foundLang bool
	for i := 0; i < int(exthRecordCount); i++ {
		recType := readUint32BE(rec0, offset)
		recLen := readUint32BE(rec0, offset+4)
		dataBytes := rec0[offset+8 : offset+int(recLen)]

		switch recType {
		case 100:
			if string(dataBytes) == "Author" {
				foundAuthor = true
			}
		case 503:
			if string(dataBytes) == "Test Book" {
				foundTitle503 = true
			}
		case 524:
			if string(dataBytes) == "ja" {
				foundLang = true
			}
		}
		offset += int(recLen)
	}

	if !foundAuthor {
		t.Error("EXTH type 100 (author) not found")
	}
	if !foundTitle503 {
		t.Error("EXTH type 503 (title) not found")
	}
	if !foundLang {
		t.Error("EXTH type 524 (language) not found")
	}
}

func TestWriteTo_NilMetadata(t *testing.T) {
	html := generateTestHTML(100)
	uid := uint32(12345)
	creation := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	cfg := AZW3WriterConfig{
		Title:        "Test Book",
		HTML:         html,
		Metadata:     nil,
		UniqueID:     &uid,
		CreationTime: creation,
	}
	w, err := NewAZW3Writer(cfg)
	if err != nil {
		t.Fatalf("NewAZW3Writer failed: %v", err)
	}
	data := writeToBuffer(t, w)

	// Should produce valid output even without metadata
	numRecords := readUint16BE(data, 76)
	if numRecords != 6 {
		t.Errorf("record count: got %d, want 6", numRecords)
	}
}

// --- Step 8: Multiple text records ---

func TestWriteTo_MultipleTextRecords(t *testing.T) {
	// Create HTML larger than 4096 bytes to force multiple text records
	html := generateTestHTML(5000)
	uid := uint32(12345)
	creation := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	cfg := AZW3WriterConfig{
		Title:        "Test Book",
		HTML:         html,
		UniqueID:     &uid,
		CreationTime: creation,
	}
	w, err := NewAZW3Writer(cfg)
	if err != nil {
		t.Fatalf("NewAZW3Writer failed: %v", err)
	}
	data := writeToBuffer(t, w)

	// Expected text records: ceil(5000/4096) = 2
	// Total records: Record0 + text×2 + FDST + FLIS + FCIS + EOF = 7
	numRecords := readUint16BE(data, 76)
	if numRecords != 7 {
		t.Errorf("record count: got %d, want 7", numRecords)
	}

	// Verify MOBI header record numbers
	rec0 := extractRecord(data, 0)
	mobiStart := 16

	firstContent := readUint16BE(rec0, mobiStart+160)
	if firstContent != 1 {
		t.Errorf("FirstContentRecord: got %d, want 1", firstContent)
	}

	lastContent := readUint16BE(rec0, mobiStart+162)
	if lastContent != 2 {
		t.Errorf("LastContentRecord: got %d, want 2", lastContent)
	}

	// FLIS should be at index 4 (0=Record0, 1-2=text, 3=FDST, 4=FLIS)
	flisNum := readUint32BE(rec0, mobiStart+176)
	if flisNum != 4 {
		t.Errorf("FLISRecordNumber: got %d, want 4", flisNum)
	}

	// FCIS should be at index 5
	fcisNum := readUint32BE(rec0, mobiStart+168)
	if fcisNum != 5 {
		t.Errorf("FCISRecordNumber: got %d, want 5", fcisNum)
	}

	// Concatenate text records should equal original HTML
	var textData []byte
	textRecordCount := TextRecordCount(html)
	for i := 1; i <= textRecordCount; i++ {
		textData = append(textData, extractRecord(data, i)...)
	}
	if !bytes.Equal(textData, html) {
		t.Errorf("concatenated text records don't match original HTML: got %d bytes, want %d", len(textData), len(html))
	}
}

// --- Step 9: Image records ---

func TestWriteTo_WithImageRecords(t *testing.T) {
	html := generateTestHTML(100)
	uid := uint32(12345)
	creation := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)

	img1 := bytes.Repeat([]byte{0xFF}, 100)
	img2 := bytes.Repeat([]byte{0xAA}, 200)

	cfg := AZW3WriterConfig{
		Title:        "Test Book",
		HTML:         html,
		UniqueID:     &uid,
		CreationTime: creation,
		ImageRecords: [][]byte{img1, img2},
	}
	w, err := NewAZW3Writer(cfg)
	if err != nil {
		t.Fatalf("NewAZW3Writer failed: %v", err)
	}
	data := writeToBuffer(t, w)

	// Record count: Record0 + text×1 + image×2 + FDST + FLIS + FCIS + EOF = 8
	numRecords := readUint16BE(data, 76)
	if numRecords != 8 {
		t.Errorf("record count: got %d, want 8", numRecords)
	}

	// Verify FirstImageIndex in MOBI header
	rec0 := extractRecord(data, 0)
	mobiStart := 16
	firstImageIdx := readUint32BE(rec0, mobiStart+80)
	// firstImageIndex = 1 (text records end) + 1 = 2
	if firstImageIdx != 2 {
		t.Errorf("FirstImageIndex: got %d, want 2", firstImageIdx)
	}

	// Verify image records content
	imgRec1 := extractRecord(data, 2)
	if !bytes.Equal(imgRec1, img1) {
		t.Error("image record 1 content mismatch")
	}
	imgRec2 := extractRecord(data, 3)
	if !bytes.Equal(imgRec2, img2) {
		t.Error("image record 2 content mismatch")
	}

	// FDST should be at index 4
	fdstRec := extractRecord(data, 4)
	if string(fdstRec[:4]) != "FDST" {
		t.Errorf("FDST at index 4: got %q, want %q", string(fdstRec[:4]), "FDST")
	}

	// FLIS at index 5
	flisRec := extractRecord(data, 5)
	if string(flisRec[:4]) != "FLIS" {
		t.Errorf("FLIS at index 5: got %q, want %q", string(flisRec[:4]), "FLIS")
	}

	// FCIS at index 6
	fcisRec := extractRecord(data, 6)
	if string(fcisRec[:4]) != "FCIS" {
		t.Errorf("FCIS at index 6: got %q, want %q", string(fcisRec[:4]), "FCIS")
	}

	// EOF at index 7
	eofRec := extractRecord(data, 7)
	eofVal := readUint32BE(eofRec, 0)
	if eofVal != 0xE98E0D0A {
		t.Errorf("EOF value: got 0x%08X, want 0xE98E0D0A", eofVal)
	}
}

func TestWriteTo_ImageRecordsShiftRecordNumbers(t *testing.T) {
	html := generateTestHTML(100)
	uid := uint32(12345)
	creation := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)

	img1 := bytes.Repeat([]byte{0xFF}, 50)

	cfg := AZW3WriterConfig{
		Title:        "Test Book",
		HTML:         html,
		UniqueID:     &uid,
		CreationTime: creation,
		ImageRecords: [][]byte{img1},
	}
	w, err := NewAZW3Writer(cfg)
	if err != nil {
		t.Fatalf("NewAZW3Writer failed: %v", err)
	}
	data := writeToBuffer(t, w)

	rec0 := extractRecord(data, 0)
	mobiStart := 16

	// With 1 image: Record0, text×1, image×1, FDST, FLIS, FCIS, EOF = 7 records
	numRecords := readUint16BE(data, 76)
	if numRecords != 7 {
		t.Errorf("record count: got %d, want 7", numRecords)
	}

	// FLIS should be at index 4 (0=Record0, 1=text, 2=image, 3=FDST, 4=FLIS)
	flisNum := readUint32BE(rec0, mobiStart+176)
	if flisNum != 4 {
		t.Errorf("FLISRecordNumber: got %d, want 4", flisNum)
	}

	// FCIS should be at index 5
	fcisNum := readUint32BE(rec0, mobiStart+168)
	if fcisNum != 5 {
		t.Errorf("FCISRecordNumber: got %d, want 5", fcisNum)
	}

	// EXTH 125 should be 7
	exthStart := 16 + MOBIHeaderSize
	exthRecordCount := readUint32BE(rec0, exthStart+8)
	offset := exthStart + 12
	for i := 0; i < int(exthRecordCount); i++ {
		recType := readUint32BE(rec0, offset)
		recLen := readUint32BE(rec0, offset+4)
		if recType == 125 {
			val := readUint32BE(rec0, offset+8)
			if val != 7 {
				t.Errorf("EXTH 125 (record count): got %d, want 7", val)
			}
		}
		offset += int(recLen)
	}
}
