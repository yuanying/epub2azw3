package mobi

import (
	"bytes"
	"encoding/binary"
	"strings"
	"testing"
	"time"

	"github.com/yuanying/epub2azw3/internal/epub"
)

// --- MOBIWriter test helpers ---

func mobiWriteToBuffer(t *testing.T, w *MOBIWriter) []byte {
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

// mobiExtractRecord extracts the Nth record data from a PDB file binary.
func mobiExtractRecord(data []byte, index int) []byte {
	numRecords := int(readUint16BE(data, 76))
	if index >= numRecords {
		return nil
	}
	recordListStart := 78
	entryOffset := recordListStart + index*8
	recOffset := readUint32BE(data, entryOffset)

	var nextOffset uint32
	if index+1 < numRecords {
		nextEntryOffset := recordListStart + (index+1)*8
		nextOffset = readUint32BE(data, nextEntryOffset)
	} else {
		nextOffset = uint32(len(data))
	}
	return data[recOffset:nextOffset]
}

// findEXTHValue searches EXTH records for a specific type and returns the uint32 value.
func findEXTHValue(rec0 []byte, exthStart int, targetType uint32) (uint32, bool) {
	exthRecordCount := readUint32BE(rec0, exthStart+8)
	offset := exthStart + 12

	for i := 0; i < int(exthRecordCount); i++ {
		recType := readUint32BE(rec0, offset)
		recLen := readUint32BE(rec0, offset+4)
		if recType == targetType {
			return readUint32BE(rec0, offset+8), true
		}
		offset += int(recLen)
	}
	return 0, false
}

func newMinimalMOBIWriter(t *testing.T) *MOBIWriter {
	t.Helper()
	html := generateTestHTML(100)
	uid := uint32(12345)
	creation := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	cfg := MOBIWriterConfig{
		Title:        "Test Book",
		HTML:         html,
		UniqueID:     &uid,
		CreationTime: creation,
	}
	w, err := NewMOBIWriter(cfg)
	if err != nil {
		t.Fatalf("NewMOBIWriter failed: %v", err)
	}
	return w
}

// --- Step 1: Constructor and validation ---

func TestNewMOBIWriter_MinimalConfig(t *testing.T) {
	html := generateTestHTML(100)
	uid := uint32(12345)
	cfg := MOBIWriterConfig{
		Title:    "Test Book",
		HTML:     html,
		UniqueID: &uid,
	}
	w, err := NewMOBIWriter(cfg)
	if err != nil {
		t.Fatalf("NewMOBIWriter failed: %v", err)
	}
	if w == nil {
		t.Fatal("NewMOBIWriter returned nil")
	}
}

func TestNewMOBIWriter_EmptyHTML(t *testing.T) {
	cfg := MOBIWriterConfig{
		Title: "Test Book",
		HTML:  []byte{},
	}
	_, err := NewMOBIWriter(cfg)
	if err == nil {
		t.Fatal("expected error for empty HTML")
	}
}

// --- Step 2: Record count ---

func TestMOBIWriter_RecordCount(t *testing.T) {
	// 100B HTML, no images
	// MOBI7 section: Record0 + text×1 + FLIS + FCIS + BoundaryEOF = 5
	// KF8 section:   Record0 + text×1 + FDST + FLIS + FCIS + EOF = 6
	// Total: 11
	w := newMinimalMOBIWriter(t)
	data := mobiWriteToBuffer(t, w)

	numRecords := readUint16BE(data, 76)
	if numRecords != 11 {
		t.Errorf("record count: got %d, want 11", numRecords)
	}
}

// --- Step 3: PDB header ---

func TestMOBIWriter_PDBHeader(t *testing.T) {
	w := newMinimalMOBIWriter(t)
	data := mobiWriteToBuffer(t, w)

	// Check PDB Type = "BOOK"
	if string(data[60:64]) != "BOOK" {
		t.Errorf("PDB Type: got %q, want %q", string(data[60:64]), "BOOK")
	}
	// Check PDB Creator = "MOBI"
	if string(data[64:68]) != "MOBI" {
		t.Errorf("PDB Creator: got %q, want %q", string(data[64:68]), "MOBI")
	}
}

// --- Step 4: MOBI7 Record 0 ---

func TestMOBIWriter_MOBI7Record0(t *testing.T) {
	w := newMinimalMOBIWriter(t)
	data := mobiWriteToBuffer(t, w)

	rec0 := mobiExtractRecord(data, 0)
	mobiStart := PalmDOCHeaderSize // 16

	// Verify MOBI magic
	if string(rec0[mobiStart:mobiStart+4]) != "MOBI" {
		t.Fatalf("MOBI magic: got %q, want %q", string(rec0[mobiStart:mobiStart+4]), "MOBI")
	}

	// Header size = 232 (MOBI7)
	headerLen := readUint32BE(rec0, mobiStart+4)
	if headerLen != MOBI7HeaderSize {
		t.Errorf("MOBI7 header size = %d, want %d", headerLen, MOBI7HeaderSize)
	}

	// MOBI type = 2 (MOBI7)
	mobiType := readUint32BE(rec0, mobiStart+8)
	if mobiType != MOBITypeMOBI7 {
		t.Errorf("MOBI type = %d, want %d", mobiType, MOBITypeMOBI7)
	}

	// File version = 6 (MOBI7)
	fileVersion := readUint32BE(rec0, mobiStart+20)
	if fileVersion != FileVersionMOBI7 {
		t.Errorf("file version = %d, want %d", fileVersion, FileVersionMOBI7)
	}

	// FirstContentRecord = 1
	firstContent := readUint16BE(rec0, mobiStart+160)
	if firstContent != 1 {
		t.Errorf("MOBI7 FirstContentRecord: got %d, want 1", firstContent)
	}

	// LastContentRecord = 1 (100B HTML = 1 text record)
	lastContent := readUint16BE(rec0, mobiStart+162)
	if lastContent != 1 {
		t.Errorf("MOBI7 LastContentRecord: got %d, want 1", lastContent)
	}
}

// --- Step 5: Boundary record ---

func TestMOBIWriter_BoundaryRecord(t *testing.T) {
	w := newMinimalMOBIWriter(t)
	data := mobiWriteToBuffer(t, w)

	// With 100B HTML, no images:
	// MOBI7: Record0(0), text(1), FLIS(2), FCIS(3), BoundaryEOF(4)
	// KF8:   Record0(5), text(6), FDST(7), FLIS(8), FCIS(9), EOF(10)
	boundaryIdx := 4
	boundaryRec := mobiExtractRecord(data, boundaryIdx)

	if len(boundaryRec) != 4 {
		t.Fatalf("boundary record size: got %d, want 4", len(boundaryRec))
	}

	val := binary.BigEndian.Uint32(boundaryRec)
	if val != 0xE98E0D0A {
		t.Errorf("boundary value: got 0x%08X, want 0xE98E0D0A", val)
	}
}

// --- Step 6: KF8 Record 0 ---

func TestMOBIWriter_KF8Record0(t *testing.T) {
	w := newMinimalMOBIWriter(t)
	data := mobiWriteToBuffer(t, w)

	// KF8 Record 0 is at index 5 (after MOBI7: 0,1,2,3,4)
	kf8Rec0Idx := 5
	kf8Rec0 := mobiExtractRecord(data, kf8Rec0Idx)
	mobiStart := PalmDOCHeaderSize

	// Verify MOBI magic
	if string(kf8Rec0[mobiStart:mobiStart+4]) != "MOBI" {
		t.Fatalf("KF8 MOBI magic: got %q, want %q", string(kf8Rec0[mobiStart:mobiStart+4]), "MOBI")
	}

	// Header size = 248 (KF8)
	headerLen := readUint32BE(kf8Rec0, mobiStart+4)
	if headerLen != MOBIHeaderSize {
		t.Errorf("KF8 header size = %d, want %d", headerLen, MOBIHeaderSize)
	}

	// MOBI type = 248 (KF8)
	mobiType := readUint32BE(kf8Rec0, mobiStart+8)
	if mobiType != MOBITypeKF8 {
		t.Errorf("KF8 MOBI type = %d, want %d", mobiType, MOBITypeKF8)
	}

	// File version = 8 (KF8)
	fileVersion := readUint32BE(kf8Rec0, mobiStart+20)
	if fileVersion != FileVersionKF8 {
		t.Errorf("KF8 file version = %d, want %d", fileVersion, FileVersionKF8)
	}

	// FirstContentRecord and LastContentRecord should use global PDB record numbers
	// KF8 text starts at index 6 (kf8Rec0Idx + 1)
	firstContent := readUint16BE(kf8Rec0, mobiStart+160)
	expectedFirst := uint16(kf8Rec0Idx + 1)
	if firstContent != expectedFirst {
		t.Errorf("KF8 FirstContentRecord: got %d, want %d", firstContent, expectedFirst)
	}

	lastContent := readUint16BE(kf8Rec0, mobiStart+162)
	expectedLast := uint16(kf8Rec0Idx + 1) // only 1 text record
	if lastContent != expectedLast {
		t.Errorf("KF8 LastContentRecord: got %d, want %d", lastContent, expectedLast)
	}
}

// --- Step 7: EXTH 121 values ---

func TestMOBIWriter_MOBI7EXTH121(t *testing.T) {
	w := newMinimalMOBIWriter(t)
	data := mobiWriteToBuffer(t, w)

	rec0 := mobiExtractRecord(data, 0)
	exthStart := PalmDOCHeaderSize + MOBI7HeaderSize

	// MOBI7 EXTH 121 should be the boundary record index
	// Boundary is at index 4
	val, found := findEXTHValue(rec0, exthStart, 121)
	if !found {
		t.Fatal("MOBI7 EXTH 121 not found")
	}
	if val != 4 {
		t.Errorf("MOBI7 EXTH 121 = %d, want 4 (boundary index)", val)
	}
}

func TestMOBIWriter_KF8EXTH121(t *testing.T) {
	w := newMinimalMOBIWriter(t)
	data := mobiWriteToBuffer(t, w)

	// KF8 Record 0 is at index 5
	kf8Rec0 := mobiExtractRecord(data, 5)
	exthStart := PalmDOCHeaderSize + MOBIHeaderSize

	// KF8 EXTH 121 should be 0
	val, found := findEXTHValue(kf8Rec0, exthStart, 121)
	if !found {
		t.Fatal("KF8 EXTH 121 not found")
	}
	if val != 0 {
		t.Errorf("KF8 EXTH 121 = %d, want 0", val)
	}
}

// --- Step 8: Image records ---

func TestMOBIWriter_WithImages(t *testing.T) {
	html := generateTestHTML(100)
	uid := uint32(12345)
	creation := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)

	img1 := bytes.Repeat([]byte{0xFF}, 100)
	img2 := bytes.Repeat([]byte{0xAA}, 200)

	cfg := MOBIWriterConfig{
		Title:        "Test Book",
		HTML:         html,
		UniqueID:     &uid,
		CreationTime: creation,
		ImageRecords: [][]byte{img1, img2},
	}
	w, err := NewMOBIWriter(cfg)
	if err != nil {
		t.Fatalf("NewMOBIWriter failed: %v", err)
	}
	data := mobiWriteToBuffer(t, w)

	// MOBI7: Record0(0), text(1), img(2), img(3), FLIS(4), FCIS(5), BoundaryEOF(6)
	// KF8:   Record0(7), text(8), FDST(9), FLIS(10), FCIS(11), EOF(12)
	// Total: 13
	numRecords := readUint16BE(data, 76)
	if numRecords != 13 {
		t.Errorf("record count: got %d, want 13", numRecords)
	}

	// Image records are shared at indices 2,3
	imgRec1 := mobiExtractRecord(data, 2)
	if !bytes.Equal(imgRec1, img1) {
		t.Error("image record 1 content mismatch")
	}
	imgRec2 := mobiExtractRecord(data, 3)
	if !bytes.Equal(imgRec2, img2) {
		t.Error("image record 2 content mismatch")
	}

	// Verify MOBI7 FirstImageIndex
	mobi7Rec0 := mobiExtractRecord(data, 0)
	mobi7Start := PalmDOCHeaderSize
	mobi7FirstImage := readUint32BE(mobi7Rec0, mobi7Start+80)
	if mobi7FirstImage != 2 {
		t.Errorf("MOBI7 FirstImageIndex: got %d, want 2", mobi7FirstImage)
	}

	// Verify KF8 FirstImageIndex (same global index)
	kf8Rec0 := mobiExtractRecord(data, 7)
	kf8Start := PalmDOCHeaderSize
	kf8FirstImage := readUint32BE(kf8Rec0, kf8Start+80)
	if kf8FirstImage != 2 {
		t.Errorf("KF8 FirstImageIndex: got %d, want 2", kf8FirstImage)
	}
}

// --- Step 9: NCX record ---

func TestMOBIWriter_WithNCX(t *testing.T) {
	html := generateTestHTML(100)
	uid := uint32(12345)
	creation := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	ncxData := []byte("<html><body><h1>TOC</h1></body></html>")

	cfg := MOBIWriterConfig{
		Title:        "Test Book",
		HTML:         html,
		UniqueID:     &uid,
		CreationTime: creation,
		NCXRecord:    ncxData,
	}
	w, err := NewMOBIWriter(cfg)
	if err != nil {
		t.Fatalf("NewMOBIWriter failed: %v", err)
	}
	data := mobiWriteToBuffer(t, w)

	// MOBI7: Record0(0), text(1), FLIS(2), FCIS(3), BoundaryEOF(4)
	// KF8:   Record0(5), text(6), NCX(7), FDST(8), FLIS(9), FCIS(10), EOF(11)
	// Total: 12
	numRecords := readUint16BE(data, 76)
	if numRecords != 12 {
		t.Errorf("record count: got %d, want 12", numRecords)
	}

	// NCX is in KF8 section at index 7
	ncxRec := mobiExtractRecord(data, 7)
	if !bytes.Equal(ncxRec, ncxData) {
		t.Error("NCX record content mismatch")
	}
}

// --- Step 10: Metadata ---

func TestMOBIWriter_WithMetadata(t *testing.T) {
	html := generateTestHTML(100)
	uid := uint32(12345)
	creation := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	meta := &epub.Metadata{
		Title:    "Test Book",
		Creators: []epub.Creator{{Name: "Author"}},
		Language: "ja",
	}
	cfg := MOBIWriterConfig{
		Title:        "Test Book",
		HTML:         html,
		Metadata:     meta,
		UniqueID:     &uid,
		CreationTime: creation,
	}
	w, err := NewMOBIWriter(cfg)
	if err != nil {
		t.Fatalf("NewMOBIWriter failed: %v", err)
	}
	data := mobiWriteToBuffer(t, w)

	// Check MOBI7 Record 0 has EXTH with author info
	rec0 := mobiExtractRecord(data, 0)
	exthStart := PalmDOCHeaderSize + MOBI7HeaderSize

	if string(rec0[exthStart:exthStart+4]) != "EXTH" {
		t.Fatalf("MOBI7 EXTH magic missing")
	}

	exthRecordCount := readUint32BE(rec0, exthStart+8)
	offset := exthStart + 12
	var foundAuthor bool
	for i := 0; i < int(exthRecordCount); i++ {
		recType := readUint32BE(rec0, offset)
		recLen := readUint32BE(rec0, offset+4)
		if recType == 100 {
			dataBytes := rec0[offset+8 : offset+int(recLen)]
			if string(dataBytes) == "Author" {
				foundAuthor = true
			}
		}
		offset += int(recLen)
	}
	if !foundAuthor {
		t.Error("MOBI7 EXTH type 100 (author) not found")
	}
}

// --- Step 11: Multiple text records ---

func TestMOBIWriter_MultipleTextRecords(t *testing.T) {
	html := generateTestHTML(5000)
	uid := uint32(12345)
	creation := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	cfg := MOBIWriterConfig{
		Title:        "Test Book",
		HTML:         html,
		UniqueID:     &uid,
		CreationTime: creation,
	}
	w, err := NewMOBIWriter(cfg)
	if err != nil {
		t.Fatalf("NewMOBIWriter failed: %v", err)
	}
	data := mobiWriteToBuffer(t, w)

	// 5000B HTML = 2 text records
	// MOBI7: Record0(0), text(1), text(2), FLIS(3), FCIS(4), BoundaryEOF(5)
	// KF8:   Record0(6), text(7), text(8), FDST(9), FLIS(10), FCIS(11), EOF(12)
	// Total: 13
	numRecords := readUint16BE(data, 76)
	if numRecords != 13 {
		t.Errorf("record count: got %d, want 13", numRecords)
	}

	// Verify MOBI7 text records concat = original HTML
	var mobi7Text []byte
	mobi7Text = append(mobi7Text, mobiExtractRecord(data, 1)...)
	mobi7Text = append(mobi7Text, mobiExtractRecord(data, 2)...)
	if !bytes.Equal(mobi7Text, html) {
		t.Errorf("MOBI7 text records don't match HTML: got %d bytes, want %d", len(mobi7Text), len(html))
	}

	// Verify KF8 text records concat = original HTML
	var kf8Text []byte
	kf8Text = append(kf8Text, mobiExtractRecord(data, 7)...)
	kf8Text = append(kf8Text, mobiExtractRecord(data, 8)...)
	if !bytes.Equal(kf8Text, html) {
		t.Errorf("KF8 text records don't match HTML: got %d bytes, want %d", len(kf8Text), len(html))
	}
}

// --- Step 12: PalmDoc compression ---

func TestMOBIWriter_PalmDocCompression(t *testing.T) {
	html := generateTestHTML(100)
	uid := uint32(12345)
	creation := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	cfg := MOBIWriterConfig{
		Title:        "Test Book",
		HTML:         html,
		Compression:  CompressionPalmDoc,
		UniqueID:     &uid,
		CreationTime: creation,
	}
	w, err := NewMOBIWriter(cfg)
	if err != nil {
		t.Fatalf("NewMOBIWriter failed: %v", err)
	}
	var buf bytes.Buffer
	if _, err := w.WriteTo(&buf); err != nil {
		t.Fatalf("WriteTo failed: %v", err)
	}
	if buf.Len() == 0 {
		t.Fatal("WriteTo produced empty output")
	}
}

// --- Step 13: Record offsets ---

func TestMOBIWriter_RecordOffsets(t *testing.T) {
	w := newMinimalMOBIWriter(t)
	data := mobiWriteToBuffer(t, w)

	numRecords := int(readUint16BE(data, 76))
	recordListStart := 78

	// Offsets should be monotonically increasing
	prevOffset := readUint32BE(data, recordListStart)
	for i := 1; i < numRecords; i++ {
		off := readUint32BE(data, recordListStart+i*8)
		if off <= prevOffset {
			t.Errorf("record %d offset %d not greater than record %d offset %d", i, off, i-1, prevOffset)
		}
		prevOffset = off
	}
}

// --- Step 14: UniqueID shared between MOBI7 and KF8 ---

func TestMOBIWriter_UniqueIDSharedWhenAutoGenerated(t *testing.T) {
	html := generateTestHTML(100)
	creation := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	// UniqueID is nil → auto-generated, must be shared
	cfg := MOBIWriterConfig{
		Title:        "Test Book",
		HTML:         html,
		CreationTime: creation,
	}
	w, err := NewMOBIWriter(cfg)
	if err != nil {
		t.Fatalf("NewMOBIWriter failed: %v", err)
	}
	data := mobiWriteToBuffer(t, w)

	// MOBI7 Record 0 UniqueID
	rec0 := mobiExtractRecord(data, 0)
	mobi7UID := readUint32BE(rec0, PalmDOCHeaderSize+16)

	// Find KF8 Record 0 (after boundary)
	numRecords := int(readUint16BE(data, 76))
	kf8Rec0Idx := -1
	for i := 1; i < numRecords; i++ {
		rec := mobiExtractRecord(data, i)
		if len(rec) == 4 && readUint32BE(rec, 0) == 0xE98E0D0A {
			kf8Rec0Idx = i + 1
			break
		}
	}
	if kf8Rec0Idx < 0 || kf8Rec0Idx >= numRecords {
		t.Fatal("could not find KF8 Record 0")
	}

	kf8Rec0 := mobiExtractRecord(data, kf8Rec0Idx)
	kf8UID := readUint32BE(kf8Rec0, PalmDOCHeaderSize+16)

	if mobi7UID != kf8UID {
		t.Errorf("UniqueID mismatch: MOBI7=0x%08X, KF8=0x%08X", mobi7UID, kf8UID)
	}
}

// --- Step 15: Large HTML generates correct structure ---

func TestMOBIWriter_LargeHTML(t *testing.T) {
	// HTML spanning many text records
	html := []byte("<html><body>" + strings.Repeat("A", 20000) + "</body></html>")
	uid := uint32(12345)
	creation := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	cfg := MOBIWriterConfig{
		Title:        "Test Book",
		HTML:         html,
		UniqueID:     &uid,
		CreationTime: creation,
	}
	w, err := NewMOBIWriter(cfg)
	if err != nil {
		t.Fatalf("NewMOBIWriter failed: %v", err)
	}
	data := mobiWriteToBuffer(t, w)

	// Verify output is non-empty and well-formed
	numRecords := readUint16BE(data, 76)
	if numRecords < 10 {
		t.Errorf("expected at least 10 records for large HTML, got %d", numRecords)
	}

	// Verify MOBI7 Record 0 exists
	rec0 := mobiExtractRecord(data, 0)
	mobiStart := PalmDOCHeaderSize
	mobiType := readUint32BE(rec0, mobiStart+8)
	if mobiType != MOBITypeMOBI7 {
		t.Errorf("Record 0 MOBI type = %d, want %d", mobiType, MOBITypeMOBI7)
	}
}
