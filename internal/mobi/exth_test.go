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
		100: {"Author One & Author Two"},
		101: {"Test Publisher"},
		103: {"A test book description"},
		104: {"9784123456780"},
		105: {"Fiction; Science"},
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

func TestJoinAuthors(t *testing.T) {
	tests := []struct {
		name     string
		creators []epub.Creator
		want     string
	}{
		{
			name:     "single author with role aut",
			creators: []epub.Creator{{Name: "Author One", Role: "aut"}},
			want:     "Author One",
		},
		{
			name:     "single author with empty role",
			creators: []epub.Creator{{Name: "Author One", Role: ""}},
			want:     "Author One",
		},
		{
			name: "multiple authors joined with ampersand",
			creators: []epub.Creator{
				{Name: "Author One", Role: "aut"},
				{Name: "Author Two", Role: ""},
			},
			want: "Author One & Author Two",
		},
		{
			name: "editor excluded",
			creators: []epub.Creator{
				{Name: "Author One", Role: "aut"},
				{Name: "Editor", Role: "edt"},
			},
			want: "Author One",
		},
		{
			name:     "uppercase AUT role",
			creators: []epub.Creator{{Name: "Author One", Role: "AUT"}},
			want:     "Author One",
		},
		{
			name: "role with surrounding whitespace excluded",
			creators: []epub.Creator{
				{Name: "Author One", Role: "aut"},
				{Name: "Editor", Role: " edt "},
			},
			want: "Author One",
		},
		{
			name: "all editors returns empty",
			creators: []epub.Creator{
				{Name: "Editor One", Role: "edt"},
				{Name: "Editor Two", Role: "edt"},
			},
			want: "",
		},
		{
			name:     "empty creators",
			creators: nil,
			want:     "",
		},
		{
			name: "empty name skipped",
			creators: []epub.Creator{
				{Name: "", Role: "aut"},
				{Name: "Author", Role: "aut"},
			},
			want: "Author",
		},
		{
			name: "whitespace-only name skipped",
			creators: []epub.Creator{
				{Name: "  ", Role: "aut"},
				{Name: "Author", Role: "aut"},
			},
			want: "Author",
		},
		{
			name: "Japanese author names",
			creators: []epub.Creator{
				{Name: "太宰 治", Role: "aut"},
				{Name: "芥川 龍之介", Role: "aut"},
			},
			want: "太宰 治 & 芥川 龍之介",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := joinAuthors(tt.creators)
			if got != tt.want {
				t.Errorf("joinAuthors() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestNormalizeDate(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{"date only", "2024-01-15", "2024-01-15"},
		{"RFC3339 with Z", "2023-01-15T00:00:00Z", "2023-01-15"},
		{"RFC3339 with timezone offset", "2023-06-20T14:30:00+09:00", "2023-06-20"},
		{"datetime without timezone", "2023-03-10T12:00:00", "2023-03-10"},
		{"datetime with space separator", "2024-01-01 13:45:30", "2024-01-01"},
		{"year-month only (unparseable)", "2023-01", "2023-01"},
		{"year only (unparseable)", "2023", "2023"},
		{"empty string", "", ""},
		{"not a date", "not-a-date", "not-a-date"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := normalizeDate(tt.input)
			if got != tt.want {
				t.Errorf("normalizeDate(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestExtractISBN(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
		ok    bool
	}{
		{"ISBN-13 bare", "9784123456780", "9784123456780", true},
		{"ISBN-10 bare", "4123456780", "4123456780", true},
		{"ISBN-13 with hyphens", "978-4-12345678-0", "9784123456780", true},
		{"urn:isbn prefix", "urn:isbn:9784123456780", "9784123456780", true},
		{"urn:isbn with hyphens", "urn:isbn:978-4-12345678-0", "9784123456780", true},
		{"ISBN embedded in text", "abc 9784123456780 def", "9784123456780", true},
		{"UUID should not match", "urn:uuid:12345678-1234-1234-1234-123456789012", "", false},
		{"empty string", "", "", false},
		{"short number", "12345", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, ok := extractISBN(tt.input)
			if ok != tt.ok {
				t.Errorf("extractISBN(%q) ok = %v, want %v", tt.input, ok, tt.ok)
			}
			if got != tt.want {
				t.Errorf("extractISBN(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestJoinSubjects(t *testing.T) {
	tests := []struct {
		name     string
		subjects []string
		want     string
	}{
		{"multiple subjects joined", []string{"Fiction", "Science"}, "Fiction; Science"},
		{"single subject", []string{"Fiction"}, "Fiction"},
		{"nil subjects", nil, ""},
		{"empty string skipped", []string{"Fiction", "", "Science"}, "Fiction; Science"},
		{"whitespace-only skipped", []string{"Fiction", "  ", "Science"}, "Fiction; Science"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := joinSubjects(tt.subjects)
			if got != tt.want {
				t.Errorf("joinSubjects() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestEXTHFromMetadata_SubjectJoining(t *testing.T) {
	tests := []struct {
		name     string
		subjects []string
		want     []string // expected type 105 values; nil means no record
	}{
		{"multiple joined", []string{"Fiction", "Science"}, []string{"Fiction; Science"}},
		{"empty skipped in join", []string{"", "Fiction", "  ", "Science"}, []string{"Fiction; Science"}},
		{"all empty produces no record", []string{"", "  "}, nil},
		{"nil produces no record", nil, nil},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			meta := epub.Metadata{Subjects: tt.subjects}
			h := EXTHFromMetadata(meta, 0, 0)
			data, err := h.Bytes()
			if err != nil {
				t.Fatalf("Bytes() error: %v", err)
			}
			records := parseEXTHRecords(t, data)
			if tt.want == nil {
				if len(records[105]) != 0 {
					t.Errorf("type 105 records = %v, want none", records[105])
				}
			} else {
				if len(records[105]) != len(tt.want) {
					t.Fatalf("type 105 count = %d, want %d", len(records[105]), len(tt.want))
				}
				for i, v := range tt.want {
					if records[105][i] != v {
						t.Errorf("type 105[%d] = %q, want %q", i, records[105][i], v)
					}
				}
			}
		})
	}
}

func TestEXTHFromMetadata_ISBNExtraction(t *testing.T) {
	tests := []struct {
		name       string
		identifier string
		want       []string // expected type 104 values; nil means no record
	}{
		{"ISBN-13 extracted", "978-4-12345678-0", []string{"9784123456780"}},
		{"urn:isbn extracted", "urn:isbn:9784123456780", []string{"9784123456780"}},
		{"UUID produces no record", "urn:uuid:12345678-1234-1234-1234-123456789012", nil},
		{"empty produces no record", "", nil},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			meta := epub.Metadata{Identifier: tt.identifier}
			h := EXTHFromMetadata(meta, 0, 0)
			data, err := h.Bytes()
			if err != nil {
				t.Fatalf("Bytes() error: %v", err)
			}
			records := parseEXTHRecords(t, data)
			if tt.want == nil {
				if len(records[104]) != 0 {
					t.Errorf("type 104 records = %v, want none", records[104])
				}
			} else {
				if len(records[104]) != len(tt.want) {
					t.Fatalf("type 104 count = %d, want %d", len(records[104]), len(tt.want))
				}
				for i, v := range tt.want {
					if records[104][i] != v {
						t.Errorf("type 104[%d] = %q, want %q", i, records[104][i], v)
					}
				}
			}
		})
	}
}

func TestEXTHFromMetadata_DateNormalization(t *testing.T) {
	tests := []struct {
		name string
		date string
		want string
	}{
		{"ISO 8601 full", "2023-01-15T00:00:00Z", "2023-01-15"},
		{"date only", "2024-01-15", "2024-01-15"},
		{"unparseable passes through", "2023", "2023"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			meta := epub.Metadata{Date: tt.date}
			h := EXTHFromMetadata(meta, 0, 0)
			data, err := h.Bytes()
			if err != nil {
				t.Fatalf("Bytes() error: %v", err)
			}
			records := parseEXTHRecords(t, data)
			if len(records[106]) != 1 {
				t.Fatalf("type 106 count = %d, want 1", len(records[106]))
			}
			if records[106][0] != tt.want {
				t.Errorf("type 106 = %q, want %q", records[106][0], tt.want)
			}
		})
	}
}

func TestEXTHFromMetadata_AuthorJoining(t *testing.T) {
	tests := []struct {
		name     string
		creators []epub.Creator
		want     []string // expected type 100 record values
	}{
		{
			name: "multiple authors joined",
			creators: []epub.Creator{
				{Name: "Author One", Role: "aut"},
				{Name: "Author Two", Role: ""},
			},
			want: []string{"Author One & Author Two"},
		},
		{
			name: "editor excluded from joining",
			creators: []epub.Creator{
				{Name: "Author", Role: "aut"},
				{Name: "Editor", Role: "edt"},
			},
			want: []string{"Author"},
		},
		{
			name:     "no authors produces no record",
			creators: []epub.Creator{{Name: "Editor", Role: "edt"}},
			want:     nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			meta := epub.Metadata{Creators: tt.creators}
			h := EXTHFromMetadata(meta, 0, 0)
			data, err := h.Bytes()
			if err != nil {
				t.Fatalf("Bytes() error: %v", err)
			}
			records := parseEXTHRecords(t, data)
			if tt.want == nil {
				if len(records[100]) != 0 {
					t.Errorf("type 100 records = %v, want none", records[100])
				}
			} else {
				if len(records[100]) != len(tt.want) {
					t.Fatalf("type 100 count = %d, want %d", len(records[100]), len(tt.want))
				}
				for i, v := range tt.want {
					if records[100][i] != v {
						t.Errorf("type 100[%d] = %q, want %q", i, records[100][i], v)
					}
				}
			}
		})
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
