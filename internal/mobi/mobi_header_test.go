package mobi

import (
	"encoding/binary"
	"testing"
)

func TestLanguageCode(t *testing.T) {
	tests := []struct {
		name string
		lang string
		want uint32
	}{
		{name: "Japanese", lang: "ja", want: 0x0411},
		{name: "English", lang: "en", want: 0x0409},
		{name: "German", lang: "de", want: 0x0407},
		{name: "French", lang: "fr", want: 0x040C},
		{name: "Unknown language defaults to English", lang: "zz", want: 0x0409},
		{name: "Empty string defaults to English", lang: "", want: 0x0409},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := LanguageCode(tt.lang)
			if got != tt.want {
				t.Errorf("LanguageCode(%q) = 0x%04X, want 0x%04X", tt.lang, got, tt.want)
			}
		})
	}
}

func TestPalmDOCHeaderBytes(t *testing.T) {
	cfg := MOBIHeaderConfig{
		Compression:        CompressionPalmDoc,
		TextLength:         12345,
		TextRecordCount:    4,
		Language:           "en",
		FirstImageIndex:    5,
		FirstContentRecord: 1,
		LastContentRecord:  4,
		FCISRecordNumber:   10,
		FLISRecordNumber:   11,
	}

	h, err := NewMOBIHeader(cfg)
	if err != nil {
		t.Fatalf("NewMOBIHeader() error = %v", err)
	}

	data, err := h.PalmDOCHeaderBytes()
	if err != nil {
		t.Fatalf("PalmDOCHeaderBytes() error = %v", err)
	}

	if len(data) != PalmDOCHeaderSize {
		t.Fatalf("PalmDOCHeaderBytes() length = %d, want %d", len(data), PalmDOCHeaderSize)
	}

	// Verify each field
	compression := binary.BigEndian.Uint16(data[0:2])
	if compression != CompressionPalmDoc {
		t.Errorf("compression = %d, want %d", compression, CompressionPalmDoc)
	}

	unused1 := binary.BigEndian.Uint16(data[2:4])
	if unused1 != 0 {
		t.Errorf("unused[2:4] = %d, want 0", unused1)
	}

	textLength := binary.BigEndian.Uint32(data[4:8])
	if textLength != 12345 {
		t.Errorf("textLength = %d, want 12345", textLength)
	}

	textRecordCount := binary.BigEndian.Uint16(data[8:10])
	if textRecordCount != 4 {
		t.Errorf("textRecordCount = %d, want 4", textRecordCount)
	}

	maxRecordSize := binary.BigEndian.Uint16(data[10:12])
	if maxRecordSize != MaxRecordSize {
		t.Errorf("maxRecordSize = %d, want %d", maxRecordSize, MaxRecordSize)
	}

	encryptionType := binary.BigEndian.Uint16(data[12:14])
	if encryptionType != 0 {
		t.Errorf("encryptionType = %d, want 0", encryptionType)
	}

	unused2 := binary.BigEndian.Uint16(data[14:16])
	if unused2 != 0 {
		t.Errorf("unused[14:16] = %d, want 0", unused2)
	}
}

func TestPalmDOCHeaderBytes_NoCompression(t *testing.T) {
	cfg := MOBIHeaderConfig{
		Compression:        CompressionNone,
		TextLength:         8192,
		TextRecordCount:    2,
		Language:           "ja",
		FirstImageIndex:    3,
		FirstContentRecord: 1,
		LastContentRecord:  2,
		FCISRecordNumber:   5,
		FLISRecordNumber:   6,
	}

	h, err := NewMOBIHeader(cfg)
	if err != nil {
		t.Fatalf("NewMOBIHeader() error = %v", err)
	}

	data, err := h.PalmDOCHeaderBytes()
	if err != nil {
		t.Fatalf("PalmDOCHeaderBytes() error = %v", err)
	}

	compression := binary.BigEndian.Uint16(data[0:2])
	if compression != CompressionNone {
		t.Errorf("compression = %d, want %d", compression, CompressionNone)
	}
}

func TestMOBIHeaderBytes(t *testing.T) {
	cfg := MOBIHeaderConfig{
		Compression:          CompressionPalmDoc,
		TextLength:           50000,
		TextRecordCount:      13,
		Language:             "ja",
		FirstImageIndex:      14,
		FirstContentRecord:   1,
		LastContentRecord:    13,
		FCISRecordNumber:     20,
		FLISRecordNumber:     21,
		ExtraRecordDataFlags: 0x01,
		FDSTFlowCount:        1,
		FDSTOffset:           18,
	}

	h, err := NewMOBIHeader(cfg)
	if err != nil {
		t.Fatalf("NewMOBIHeader() error = %v", err)
	}

	data, err := h.MOBIHeaderBytes(0, 0, 0)
	if err != nil {
		t.Fatalf("MOBIHeaderBytes() error = %v", err)
	}

	// Verify total size is 248 bytes
	if len(data) != MOBIHeaderSize {
		t.Fatalf("MOBIHeaderBytes() length = %d, want %d", len(data), MOBIHeaderSize)
	}

	// Verify "MOBI" identifier at offset 0
	if string(data[0:4]) != "MOBI" {
		t.Errorf("identifier = %q, want %q", string(data[0:4]), "MOBI")
	}

	// Verify header length at offset 4
	headerLen := binary.BigEndian.Uint32(data[4:8])
	if headerLen != MOBIHeaderSize {
		t.Errorf("header length = %d, want %d", headerLen, MOBIHeaderSize)
	}

	// Verify MOBI type at offset 8
	mobiType := binary.BigEndian.Uint32(data[8:12])
	if mobiType != MOBITypeKF8 {
		t.Errorf("MOBI type = %d, want %d", mobiType, MOBITypeKF8)
	}

	// Verify text encoding at offset 12
	encoding := binary.BigEndian.Uint32(data[12:16])
	if encoding != EncodingUTF8 {
		t.Errorf("encoding = %d, want %d", encoding, EncodingUTF8)
	}

	// Verify UniqueID at offset 16 is non-zero
	uniqueID := binary.BigEndian.Uint32(data[16:20])
	if uniqueID == 0 {
		t.Error("uniqueID should not be zero")
	}

	// Verify file version at offset 20
	fileVersion := binary.BigEndian.Uint32(data[20:24])
	if fileVersion != FileVersionKF8 {
		t.Errorf("file version = %d, want %d", fileVersion, FileVersionKF8)
	}

	// Verify unused index fields at offsets 24-63 are 0xFFFFFFFF
	for offset := 24; offset <= 64; offset += 4 {
		val := binary.BigEndian.Uint32(data[offset : offset+4])
		if val != 0xFFFFFFFF {
			t.Errorf("offset %d = 0x%08X, want 0xFFFFFFFF", offset, val)
		}
	}

	// Verify language code at offset 76
	langCode := binary.BigEndian.Uint32(data[76:80])
	if langCode != 0x0411 {
		t.Errorf("language code = 0x%04X, want 0x0411", langCode)
	}

	// Verify first image index at offset 80
	firstImage := binary.BigEndian.Uint32(data[80:84])
	if firstImage != 14 {
		t.Errorf("first image index = %d, want 14", firstImage)
	}

	// Verify HUFF fields at offsets 84-96
	huffFirst := binary.BigEndian.Uint32(data[84:88])
	if huffFirst != 0xFFFFFFFF {
		t.Errorf("HUFF first index = 0x%08X, want 0xFFFFFFFF", huffFirst)
	}
	huffCount := binary.BigEndian.Uint32(data[88:92])
	if huffCount != 0 {
		t.Errorf("HUFF record count = %d, want 0", huffCount)
	}
	huffTable := binary.BigEndian.Uint32(data[92:96])
	if huffTable != 0xFFFFFFFF {
		t.Errorf("HUFF table index = 0x%08X, want 0xFFFFFFFF", huffTable)
	}
	huffTableCount := binary.BigEndian.Uint32(data[96:100])
	if huffTableCount != 0 {
		t.Errorf("HUFF table count = %d, want 0", huffTableCount)
	}

	// Verify first/last content record at offsets 160-163
	firstContent := binary.BigEndian.Uint16(data[160:162])
	if firstContent != 1 {
		t.Errorf("first content record = %d, want 1", firstContent)
	}
	lastContent := binary.BigEndian.Uint16(data[162:164])
	if lastContent != 13 {
		t.Errorf("last content record = %d, want 13", lastContent)
	}

	// Verify unused at offset 164
	unused164 := binary.BigEndian.Uint32(data[164:168])
	if unused164 != 1 {
		t.Errorf("offset 164 = %d, want 1", unused164)
	}

	// Verify FCIS at offset 168
	fcis := binary.BigEndian.Uint32(data[168:172])
	if fcis != 20 {
		t.Errorf("FCIS record number = %d, want 20", fcis)
	}
	fcisCount := binary.BigEndian.Uint32(data[172:176])
	if fcisCount != 1 {
		t.Errorf("FCIS record count = %d, want 1", fcisCount)
	}

	// Verify FLIS at offset 176
	flis := binary.BigEndian.Uint32(data[176:180])
	if flis != 21 {
		t.Errorf("FLIS record number = %d, want 21", flis)
	}
	flisCount := binary.BigEndian.Uint32(data[180:184])
	if flisCount != 1 {
		t.Errorf("FLIS record count = %d, want 1", flisCount)
	}

	// Verify extra record data flags at offset 208
	extraFlags := binary.BigEndian.Uint32(data[208:212])
	if extraFlags != 0x01 {
		t.Errorf("extra record data flags = 0x%08X, want 0x01", extraFlags)
	}

	// Verify INDX record offset at 212
	indx := binary.BigEndian.Uint32(data[212:216])
	if indx != 0xFFFFFFFF {
		t.Errorf("INDX record offset = 0x%08X, want 0xFFFFFFFF", indx)
	}

	// Verify KF8 unused fields at offsets 216-232
	for offset := 216; offset <= 232; offset += 4 {
		val := binary.BigEndian.Uint32(data[offset : offset+4])
		if val != 0xFFFFFFFF {
			t.Errorf("KF8 offset %d = 0x%08X, want 0xFFFFFFFF", offset, val)
		}
	}

	// Verify FDST flow count at offset 236
	fdstFlow := binary.BigEndian.Uint32(data[236:240])
	if fdstFlow != 1 {
		t.Errorf("FDST flow count = %d, want 1", fdstFlow)
	}

	// Verify FDST offset at offset 240
	fdstOff := binary.BigEndian.Uint32(data[240:244])
	if fdstOff != 18 {
		t.Errorf("FDST offset = %d, want 18", fdstOff)
	}

	// Verify final unused at offset 244
	unused244 := binary.BigEndian.Uint32(data[244:248])
	if unused244 != 0 {
		t.Errorf("offset 244 = %d, want 0", unused244)
	}
}

func TestMOBIHeaderBytes_UniqueIDIsDifferent(t *testing.T) {
	cfg := MOBIHeaderConfig{
		Compression:        CompressionPalmDoc,
		TextLength:         1000,
		TextRecordCount:    1,
		Language:           "en",
		FirstImageIndex:    2,
		FirstContentRecord: 1,
		LastContentRecord:  1,
		FCISRecordNumber:   3,
		FLISRecordNumber:   4,
	}

	h1, err := NewMOBIHeader(cfg)
	if err != nil {
		t.Fatalf("NewMOBIHeader() error = %v", err)
	}

	h2, err := NewMOBIHeader(cfg)
	if err != nil {
		t.Fatalf("NewMOBIHeader() error = %v", err)
	}

	if h1.UniqueID == h2.UniqueID {
		t.Error("two MOBIHeaders should have different UniqueIDs")
	}
}

func TestMOBIHeaderBytes_EXTHFlags(t *testing.T) {
	cfg := MOBIHeaderConfig{
		Compression:        CompressionPalmDoc,
		TextLength:         1000,
		TextRecordCount:    1,
		Language:           "en",
		FirstImageIndex:    2,
		FirstContentRecord: 1,
		LastContentRecord:  1,
		FCISRecordNumber:   3,
		FLISRecordNumber:   4,
	}

	h, err := NewMOBIHeader(cfg)
	if err != nil {
		t.Fatalf("NewMOBIHeader() error = %v", err)
	}

	// With EXTH flags set
	data, err := h.MOBIHeaderBytes(0, 0, EXTHFlagPresent)
	if err != nil {
		t.Fatalf("MOBIHeaderBytes() error = %v", err)
	}

	exthFlags := binary.BigEndian.Uint32(data[100:104])
	if exthFlags != EXTHFlagPresent {
		t.Errorf("EXTH flags = 0x%08X, want 0x%08X", exthFlags, EXTHFlagPresent)
	}

	// With EXTH flags unset
	data, err = h.MOBIHeaderBytes(0, 0, 0)
	if err != nil {
		t.Fatalf("MOBIHeaderBytes() error = %v", err)
	}

	exthFlags = binary.BigEndian.Uint32(data[100:104])
	if exthFlags != 0 {
		t.Errorf("EXTH flags = 0x%08X, want 0x00000000", exthFlags)
	}
}

func TestMOBIHeaderBytes_FullNameOffsetAndLength(t *testing.T) {
	cfg := MOBIHeaderConfig{
		Compression:        CompressionPalmDoc,
		TextLength:         1000,
		TextRecordCount:    1,
		Language:           "en",
		FirstImageIndex:    2,
		FirstContentRecord: 1,
		LastContentRecord:  1,
		FCISRecordNumber:   3,
		FLISRecordNumber:   4,
	}

	h, err := NewMOBIHeader(cfg)
	if err != nil {
		t.Fatalf("NewMOBIHeader() error = %v", err)
	}

	fullNameOffset := uint32(300)
	fullNameLength := uint32(15)
	data, err := h.MOBIHeaderBytes(fullNameOffset, fullNameLength, 0)
	if err != nil {
		t.Fatalf("MOBIHeaderBytes() error = %v", err)
	}

	gotOffset := binary.BigEndian.Uint32(data[68:72])
	if gotOffset != fullNameOffset {
		t.Errorf("Full Name Offset = %d, want %d", gotOffset, fullNameOffset)
	}

	gotLength := binary.BigEndian.Uint32(data[72:76])
	if gotLength != fullNameLength {
		t.Errorf("Full Name Length = %d, want %d", gotLength, fullNameLength)
	}
}

func TestRecord0Bytes_WithoutEXTH(t *testing.T) {
	cfg := MOBIHeaderConfig{
		Compression:        CompressionPalmDoc,
		TextLength:         5000,
		TextRecordCount:    2,
		Language:           "en",
		FirstImageIndex:    3,
		FirstContentRecord: 1,
		LastContentRecord:  2,
		FCISRecordNumber:   5,
		FLISRecordNumber:   6,
	}

	h, err := NewMOBIHeader(cfg)
	if err != nil {
		t.Fatalf("NewMOBIHeader() error = %v", err)
	}

	fullName := "Test Book Title"
	data, err := h.Record0Bytes(nil, fullName)
	if err != nil {
		t.Fatalf("Record0Bytes() error = %v", err)
	}

	// Record 0 = PalmDOC(16) + MOBI(248) + Full Name + padding
	fullNameBytes := []byte(fullName)
	expectedOffset := uint32(MOBIHeaderSize) // 248 (no EXTH)
	expectedLength := uint32(len(fullNameBytes))

	// Check Full Name Offset in MOBI header (offset 68 from MOBI start = 16+68 from record start)
	gotOffset := binary.BigEndian.Uint32(data[PalmDOCHeaderSize+68 : PalmDOCHeaderSize+72])
	if gotOffset != expectedOffset {
		t.Errorf("Full Name Offset = %d, want %d", gotOffset, expectedOffset)
	}

	// Check Full Name Length in MOBI header
	gotLength := binary.BigEndian.Uint32(data[PalmDOCHeaderSize+72 : PalmDOCHeaderSize+76])
	if gotLength != expectedLength {
		t.Errorf("Full Name Length = %d, want %d", gotLength, expectedLength)
	}

	// Verify Full Name content at expected position
	nameStart := PalmDOCHeaderSize + int(expectedOffset)
	nameEnd := nameStart + int(expectedLength)
	gotName := string(data[nameStart:nameEnd])
	if gotName != fullName {
		t.Errorf("Full Name = %q, want %q", gotName, fullName)
	}

	// Check EXTH flags = 0 (no EXTH)
	exthFlags := binary.BigEndian.Uint32(data[PalmDOCHeaderSize+100 : PalmDOCHeaderSize+104])
	if exthFlags != 0 {
		t.Errorf("EXTH flags = 0x%08X, want 0x00000000", exthFlags)
	}

	// Verify 4-byte alignment
	if len(data)%4 != 0 {
		t.Errorf("Record0 length %d is not 4-byte aligned", len(data))
	}
}

func TestRecord0Bytes_WithEXTH(t *testing.T) {
	cfg := MOBIHeaderConfig{
		Compression:        CompressionPalmDoc,
		TextLength:         5000,
		TextRecordCount:    2,
		Language:           "en",
		FirstImageIndex:    3,
		FirstContentRecord: 1,
		LastContentRecord:  2,
		FCISRecordNumber:   5,
		FLISRecordNumber:   6,
	}

	h, err := NewMOBIHeader(cfg)
	if err != nil {
		t.Fatalf("NewMOBIHeader() error = %v", err)
	}

	exthData := make([]byte, 100) // dummy EXTH data
	fullName := "Book With EXTH"
	data, err := h.Record0Bytes(exthData, fullName)
	if err != nil {
		t.Fatalf("Record0Bytes() error = %v", err)
	}

	// Full Name Offset should be 248 + 100 = 348
	expectedOffset := uint32(MOBIHeaderSize + len(exthData))
	gotOffset := binary.BigEndian.Uint32(data[PalmDOCHeaderSize+68 : PalmDOCHeaderSize+72])
	if gotOffset != expectedOffset {
		t.Errorf("Full Name Offset = %d, want %d", gotOffset, expectedOffset)
	}

	// Check EXTH flags = 0x40
	exthFlags := binary.BigEndian.Uint32(data[PalmDOCHeaderSize+100 : PalmDOCHeaderSize+104])
	if exthFlags != EXTHFlagPresent {
		t.Errorf("EXTH flags = 0x%08X, want 0x%08X", exthFlags, EXTHFlagPresent)
	}

	// Verify Full Name content
	fullNameBytes := []byte(fullName)
	nameStart := PalmDOCHeaderSize + int(expectedOffset)
	nameEnd := nameStart + len(fullNameBytes)
	gotName := string(data[nameStart:nameEnd])
	if gotName != fullName {
		t.Errorf("Full Name = %q, want %q", gotName, fullName)
	}

	// Verify 4-byte alignment
	if len(data)%4 != 0 {
		t.Errorf("Record0 length %d is not 4-byte aligned", len(data))
	}
}

func TestRecord0Bytes_Padding(t *testing.T) {
	tests := []struct {
		name     string
		fullName string
	}{
		{name: "length 1 (pad 3)", fullName: "A"},
		{name: "length 2 (pad 2)", fullName: "AB"},
		{name: "length 3 (pad 1)", fullName: "ABC"},
		{name: "length 4 (pad 0)", fullName: "ABCD"},
		{name: "length 5 (pad 3)", fullName: "ABCDE"},
		{name: "length 8 (pad 0)", fullName: "ABCDEFGH"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := MOBIHeaderConfig{
				Compression:        CompressionPalmDoc,
				TextLength:         1000,
				TextRecordCount:    1,
				Language:           "en",
				FirstImageIndex:    2,
				FirstContentRecord: 1,
				LastContentRecord:  1,
				FCISRecordNumber:   3,
				FLISRecordNumber:   4,
			}

			h, err := NewMOBIHeader(cfg)
			if err != nil {
				t.Fatalf("NewMOBIHeader() error = %v", err)
			}

			data, err := h.Record0Bytes(nil, tt.fullName)
			if err != nil {
				t.Fatalf("Record0Bytes() error = %v", err)
			}

			if len(data)%4 != 0 {
				t.Errorf("Record0 length %d is not 4-byte aligned for fullName %q", len(data), tt.fullName)
			}
		})
	}
}

func TestRecord0Bytes_MultibyteName(t *testing.T) {
	cfg := MOBIHeaderConfig{
		Compression:        CompressionPalmDoc,
		TextLength:         1000,
		TextRecordCount:    1,
		Language:           "ja",
		FirstImageIndex:    2,
		FirstContentRecord: 1,
		LastContentRecord:  1,
		FCISRecordNumber:   3,
		FLISRecordNumber:   4,
	}

	h, err := NewMOBIHeader(cfg)
	if err != nil {
		t.Fatalf("NewMOBIHeader() error = %v", err)
	}

	fullName := "日本語の本"
	data, err := h.Record0Bytes(nil, fullName)
	if err != nil {
		t.Fatalf("Record0Bytes() error = %v", err)
	}

	// Verify byte length (not rune count)
	fullNameBytes := []byte(fullName)
	gotLength := binary.BigEndian.Uint32(data[PalmDOCHeaderSize+72 : PalmDOCHeaderSize+76])
	if gotLength != uint32(len(fullNameBytes)) {
		t.Errorf("Full Name Length = %d, want %d (byte length)", gotLength, len(fullNameBytes))
	}

	// Verify content
	nameStart := PalmDOCHeaderSize + MOBIHeaderSize
	nameEnd := nameStart + len(fullNameBytes)
	gotName := string(data[nameStart:nameEnd])
	if gotName != fullName {
		t.Errorf("Full Name = %q, want %q", gotName, fullName)
	}

	// Verify alignment
	if len(data)%4 != 0 {
		t.Errorf("Record0 length %d is not 4-byte aligned", len(data))
	}
}
