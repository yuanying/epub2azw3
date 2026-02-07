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
