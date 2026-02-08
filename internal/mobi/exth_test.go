package mobi

import (
	"encoding/binary"
	"testing"

	"github.com/yuanying/epub2azw3/internal/epub"
)

func TestEXTHHeader_Identifier(t *testing.T) {
	h := NewEXTHHeader(0, 0)
	data, err := h.Bytes()
	if err != nil {
		t.Fatalf("Bytes() returned error: %v", err)
	}

	if string(data[:4]) != "EXTH" {
		t.Fatalf("identifier = %q, want %q", string(data[:4]), "EXTH")
	}
}

func TestEXTHHeader_HeaderLength(t *testing.T) {
	h := NewEXTHHeader(0, 0)
	h.AddStringRecord(100, "Author")

	data, err := h.Bytes()
	if err != nil {
		t.Fatalf("Bytes() returned error: %v", err)
	}

	headerLen := binary.BigEndian.Uint32(data[4:8])
	if headerLen != uint32(len(data)) {
		t.Fatalf("header length field = %d, want %d", headerLen, len(data))
	}
}

func TestEXTHHeader_RecordEncode(t *testing.T) {
	h := NewEXTHHeader(0, 0)
	h.AddStringRecord(100, "Author")

	data, err := h.Bytes()
	if err != nil {
		t.Fatalf("Bytes() returned error: %v", err)
	}

	recordCount := binary.BigEndian.Uint32(data[8:12])
	// 2 KF8 mandatory records + 1 added record = 3
	if recordCount != 3 {
		t.Fatalf("record count = %d, want 3", recordCount)
	}

	// First record should be type 121 (KF8 boundary offset)
	offset := 12
	recType := binary.BigEndian.Uint32(data[offset : offset+4])
	if recType != 121 {
		t.Fatalf("first record type = %d, want 121", recType)
	}
	recLen := binary.BigEndian.Uint32(data[offset+4 : offset+8])
	if recLen != 12 { // 8 + 4 bytes data
		t.Fatalf("first record length = %d, want 12", recLen)
	}

	// Second record should be type 125 (KF8 record count)
	offset += int(recLen)
	recType = binary.BigEndian.Uint32(data[offset : offset+4])
	if recType != 125 {
		t.Fatalf("second record type = %d, want 125", recType)
	}

	// Third record should be type 100 (Author)
	offset += int(binary.BigEndian.Uint32(data[offset+4 : offset+8]))
	recType = binary.BigEndian.Uint32(data[offset : offset+4])
	if recType != 100 {
		t.Fatalf("third record type = %d, want 100", recType)
	}
	recLen = binary.BigEndian.Uint32(data[offset+4 : offset+8])
	if recLen != 14 { // 8 + len("Author") = 14
		t.Fatalf("third record length = %d, want 14", recLen)
	}

	recData := string(data[offset+8 : offset+int(recLen)])
	if recData != "Author" {
		t.Fatalf("third record data = %q, want %q", recData, "Author")
	}
}

func TestEXTHHeader_Uint32Record(t *testing.T) {
	h := NewEXTHHeader(0, 0)
	h.AddUint32Record(201, 42)

	data, err := h.Bytes()
	if err != nil {
		t.Fatalf("Bytes() returned error: %v", err)
	}

	// Skip identifier(4) + headerLen(4) + recordCount(4) = 12
	// Skip 2 KF8 mandatory records: type121(12 bytes) + type125(12 bytes) = 24
	offset := 12 + 12 + 12
	recType := binary.BigEndian.Uint32(data[offset : offset+4])
	if recType != 201 {
		t.Fatalf("record type = %d, want 201", recType)
	}

	recLen := binary.BigEndian.Uint32(data[offset+4 : offset+8])
	if recLen != 12 { // 8 + 4
		t.Fatalf("record length = %d, want 12", recLen)
	}

	value := binary.BigEndian.Uint32(data[offset+8 : offset+12])
	if value != 42 {
		t.Fatalf("record value = %d, want 42", value)
	}
}

func TestEXTHHeader_Alignment(t *testing.T) {
	tests := []struct {
		name    string
		dataLen int
	}{
		{"data length 1", 1},
		{"data length 2", 2},
		{"data length 3", 3},
		{"data length 4", 4},
		{"data length 5", 5},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := NewEXTHHeader(0, 0)
			h.AddStringRecord(100, string(make([]byte, tt.dataLen)))

			data, err := h.Bytes()
			if err != nil {
				t.Fatalf("Bytes() returned error: %v", err)
			}

			if len(data)%4 != 0 {
				t.Fatalf("total size %d is not 4-byte aligned", len(data))
			}

			headerLen := binary.BigEndian.Uint32(data[4:8])
			if headerLen != uint32(len(data)) {
				t.Fatalf("header length = %d, want %d", headerLen, len(data))
			}
		})
	}
}

func TestEXTHHeader_KF8MandatoryRecords(t *testing.T) {
	h := NewEXTHHeader(100, 50)

	data, err := h.Bytes()
	if err != nil {
		t.Fatalf("Bytes() returned error: %v", err)
	}

	// Record count should be 2 (only the KF8 mandatory records)
	recordCount := binary.BigEndian.Uint32(data[8:12])
	if recordCount != 2 {
		t.Fatalf("record count = %d, want 2", recordCount)
	}

	// First record: type 121 (boundary offset = 100)
	offset := 12
	recType := binary.BigEndian.Uint32(data[offset : offset+4])
	if recType != 121 {
		t.Fatalf("first record type = %d, want 121", recType)
	}
	value := binary.BigEndian.Uint32(data[offset+8 : offset+12])
	if value != 100 {
		t.Fatalf("boundary offset value = %d, want 100", value)
	}

	// Second record: type 125 (record count = 50)
	offset += 12
	recType = binary.BigEndian.Uint32(data[offset : offset+4])
	if recType != 125 {
		t.Fatalf("second record type = %d, want 125", recType)
	}
	value = binary.BigEndian.Uint32(data[offset+8 : offset+12])
	if value != 50 {
		t.Fatalf("record count value = %d, want 50", value)
	}
}

func TestEXTHHeader_SetBoundaryOffset(t *testing.T) {
	h := NewEXTHHeader(0, 0)
	h.SetBoundaryOffset(999)

	data, err := h.Bytes()
	if err != nil {
		t.Fatalf("Bytes() returned error: %v", err)
	}

	// First record data: boundary offset
	offset := 12
	value := binary.BigEndian.Uint32(data[offset+8 : offset+12])
	if value != 999 {
		t.Fatalf("boundary offset = %d, want 999", value)
	}
}

func TestEXTHHeader_SetRecordCount(t *testing.T) {
	h := NewEXTHHeader(0, 0)
	h.SetRecordCount(777)

	data, err := h.Bytes()
	if err != nil {
		t.Fatalf("Bytes() returned error: %v", err)
	}

	// Second record data: record count
	offset := 12 + 12
	value := binary.BigEndian.Uint32(data[offset+8 : offset+12])
	if value != 777 {
		t.Fatalf("record count = %d, want 777", value)
	}
}

func TestEXTHFromMetadata_AllFields(t *testing.T) {
	meta := epub.Metadata{
		Title:       "Test Book",
		Creators:    []epub.Creator{{Name: "Author One"}, {Name: "Author Two"}},
		Language:    "ja",
		Identifier:  "978-4-12345678-0",
		Publisher:   "Test Publisher",
		Date:        "2024-01-01",
		Description: "A test book description",
		Subjects:    []string{"Fiction", "Science"},
		Rights:      "Copyright 2024",
	}

	h := EXTHFromMetadata(meta, 10, 20)
	data, err := h.Bytes()
	if err != nil {
		t.Fatalf("Bytes() returned error: %v", err)
	}

	if string(data[:4]) != "EXTH" {
		t.Fatalf("identifier = %q, want %q", string(data[:4]), "EXTH")
	}

	// Verify all expected records are present
	records := parseEXTHRecords(t, data)

	expectedTypes := map[uint32][]string{
		100: {"Author One", "Author Two"},
		101: {"Test Publisher"},
		103: {"A test book description"},
		104: {"978-4-12345678-0"},
		105: {"Fiction", "Science"},
		106: {"2024-01-01"},
		109: {"Copyright 2024"},
		503: {"Test Book"},
		524: {"ja"},
	}

	for recType, expectedValues := range expectedTypes {
		found := records[recType]
		if len(found) != len(expectedValues) {
			t.Fatalf("type %d: got %d records, want %d", recType, len(found), len(expectedValues))
		}
		for i, v := range expectedValues {
			if found[i] != v {
				t.Fatalf("type %d record %d = %q, want %q", recType, i, found[i], v)
			}
		}
	}

	// KF8 mandatory records
	if records[121] == nil {
		t.Fatal("type 121 (boundary offset) record missing")
	}
	if records[125] == nil {
		t.Fatal("type 125 (record count) record missing")
	}
}

func TestEXTHFromMetadata_EmptyMetadata(t *testing.T) {
	meta := epub.Metadata{}

	h := EXTHFromMetadata(meta, 0, 0)
	data, err := h.Bytes()
	if err != nil {
		t.Fatalf("Bytes() returned error: %v", err)
	}

	// Should only contain 2 KF8 mandatory records
	recordCount := binary.BigEndian.Uint32(data[8:12])
	if recordCount != 2 {
		t.Fatalf("record count = %d, want 2", recordCount)
	}
}

func TestEXTHHeader_SizeMatchesBytes(t *testing.T) {
	h := NewEXTHHeader(100, 50)
	h.AddStringRecord(100, "Author")
	h.AddStringRecord(101, "Publisher")
	h.AddUint32Record(201, 42)

	data, err := h.Bytes()
	if err != nil {
		t.Fatalf("Bytes() returned error: %v", err)
	}

	if h.Size() != len(data) {
		t.Fatalf("Size() = %d, len(Bytes()) = %d", h.Size(), len(data))
	}
}

func TestEXTHHeader_PaddingIsZero(t *testing.T) {
	h := NewEXTHHeader(0, 0)
	h.AddStringRecord(100, "A") // data length 1, should produce 3 bytes of padding

	data, err := h.Bytes()
	if err != nil {
		t.Fatalf("Bytes() returned error: %v", err)
	}

	// Calculate expected unpadded size:
	// identifier(4) + headerLen(4) + recordCount(4) + type121(12) + type125(12) + type100(8+1=9) = 45
	// padding = (4 - (45 % 4)) % 4 = 3
	expectedPadding := 3
	for i := len(data) - expectedPadding; i < len(data); i++ {
		if data[i] != 0x00 {
			t.Fatalf("padding byte[%d] = 0x%02x, want 0x00", i, data[i])
		}
	}
}

func TestEXTHHeader_BytesRequiresMandatoryRecords(t *testing.T) {
	h := &EXTHHeader{}
	_, err := h.Bytes()
	if err == nil {
		t.Fatal("Bytes() should return error for empty records")
	}
}

func TestEXTHFromMetadata_SkipEmptyFields(t *testing.T) {
	meta := epub.Metadata{
		Title:    "T",
		Creators: []epub.Creator{{Name: ""}, {Name: "A"}},
		Subjects: []string{"", "S"},
	}
	h := EXTHFromMetadata(meta, 0, 0)
	data, err := h.Bytes()
	if err != nil {
		t.Fatalf("Bytes() returned error: %v", err)
	}

	records := parseEXTHRecords(t, data)
	if len(records[100]) != 1 || records[100][0] != "A" {
		t.Fatalf("creators = %v, want [A]", records[100])
	}
	if len(records[105]) != 1 || records[105][0] != "S" {
		t.Fatalf("subjects = %v, want [S]", records[105])
	}
}

// parseEXTHRecords is a test helper that parses the EXTH binary data into a map of type -> []string values.
func parseEXTHRecords(t *testing.T, data []byte) map[uint32][]string {
	t.Helper()
	result := make(map[uint32][]string)

	recordCount := binary.BigEndian.Uint32(data[8:12])
	offset := 12

	for i := uint32(0); i < recordCount; i++ {
		if offset+8 > len(data) {
			t.Fatalf("record %d: insufficient data at offset %d", i, offset)
		}
		recType := binary.BigEndian.Uint32(data[offset : offset+4])
		recLen := binary.BigEndian.Uint32(data[offset+4 : offset+8])

		if recLen < 8 {
			t.Fatalf("record %d: invalid length %d (minimum 8)", i, recLen)
		}
		if offset+int(recLen) > len(data) {
			t.Fatalf("record %d: data exceeds buffer at offset %d, length %d", i, offset, recLen)
		}

		recData := string(data[offset+8 : offset+int(recLen)])
		result[recType] = append(result[recType], recData)
		offset += int(recLen)
	}

	return result
}
