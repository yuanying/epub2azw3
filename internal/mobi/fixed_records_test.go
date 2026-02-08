package mobi

import (
	"bytes"
	"encoding/binary"
	"testing"
)

func TestFLISRecord_Size(t *testing.T) {
	data := FLISRecord()
	if len(data) != 36 {
		t.Fatalf("FLISRecord size = %d, want 36", len(data))
	}
}

func TestFLISRecord_Identifier(t *testing.T) {
	data := FLISRecord()
	if string(data[:4]) != "FLIS" {
		t.Fatalf("FLISRecord identifier = %q, want %q", string(data[:4]), "FLIS")
	}
}

func TestFLISRecord_FixedFields(t *testing.T) {
	data := FLISRecord()

	tests := []struct {
		name   string
		offset int
		size   int
		want   uint32
	}{
		{"fixed_length", 4, 4, 0x00000008},
		{"unknown1_hi", 8, 2, 0x0041},
		{"unknown1_lo", 10, 2, 0x0000},
		{"unknown2", 12, 4, 0x00000000},
		{"unknown3", 16, 4, 0xFFFFFFFF},
		{"unknown4_hi", 20, 2, 0x0001},
		{"unknown4_lo", 22, 2, 0x0003},
		{"unknown5", 24, 4, 0x00000003},
		{"unknown6", 28, 4, 0x00000001},
		{"unknown7", 32, 4, 0xFFFFFFFF},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var got uint32
			switch tt.size {
			case 2:
				got = uint32(binary.BigEndian.Uint16(data[tt.offset : tt.offset+tt.size]))
			case 4:
				got = binary.BigEndian.Uint32(data[tt.offset : tt.offset+tt.size])
			default:
				t.Fatalf("unsupported size %d", tt.size)
			}
			if got != tt.want {
				t.Fatalf("%s at offset %d = 0x%08X, want 0x%08X", tt.name, tt.offset, got, tt.want)
			}
		})
	}
}

func TestFCISRecord_Size(t *testing.T) {
	data, err := FCISRecord(0)
	if err != nil {
		t.Fatalf("FCISRecord returned error: %v", err)
	}
	if len(data) != 44 {
		t.Fatalf("FCISRecord size = %d, want 44", len(data))
	}
}

func TestFCISRecord_Identifier(t *testing.T) {
	data, err := FCISRecord(0)
	if err != nil {
		t.Fatalf("FCISRecord returned error: %v", err)
	}
	if string(data[:4]) != "FCIS" {
		t.Fatalf("FCISRecord identifier = %q, want %q", string(data[:4]), "FCIS")
	}
}

func TestFCISRecord_FixedFields(t *testing.T) {
	data, err := FCISRecord(99999)
	if err != nil {
		t.Fatalf("FCISRecord returned error: %v", err)
	}

	tests := []struct {
		name   string
		offset int
		size   int
		want   uint32
	}{
		{"fixed_length", 4, 4, 0x00000014},
		{"unknown1", 8, 4, 0x00000010},
		{"unknown2", 12, 4, 0x00000001},
		{"unknown3", 16, 4, 0x00000000},
		// offset 20 is textLength (variable)
		{"unknown4", 24, 4, 0x00000000},
		{"unknown5", 28, 4, 0x00000020},
		{"unknown6", 32, 4, 0x00000008},
		{"unknown7_hi", 36, 2, 0x0001},
		{"unknown7_lo", 38, 2, 0x0001},
		{"unknown8", 40, 4, 0x00000000},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var got uint32
			switch tt.size {
			case 2:
				got = uint32(binary.BigEndian.Uint16(data[tt.offset : tt.offset+tt.size]))
			case 4:
				got = binary.BigEndian.Uint32(data[tt.offset : tt.offset+tt.size])
			default:
				t.Fatalf("unsupported size %d", tt.size)
			}
			if got != tt.want {
				t.Fatalf("%s at offset %d = 0x%08X, want 0x%08X", tt.name, tt.offset, got, tt.want)
			}
		})
	}
}

func TestFCISRecord_TextLength(t *testing.T) {
	tests := []struct {
		name       string
		textLength uint32
	}{
		{"zero", 0},
		{"medium", 12345},
		{"large", 1000000},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			data, err := FCISRecord(tt.textLength)
			if err != nil {
				t.Fatalf("FCISRecord returned error: %v", err)
			}
			got := binary.BigEndian.Uint32(data[20:24])
			if got != tt.textLength {
				t.Fatalf("textLength = %d, want %d", got, tt.textLength)
			}
		})
	}
}

func TestEOFRecord_Size(t *testing.T) {
	data := EOFRecord()
	if len(data) != 4 {
		t.Fatalf("EOFRecord size = %d, want 4", len(data))
	}
}

func TestEOFRecord_Value(t *testing.T) {
	data := EOFRecord()
	got := binary.BigEndian.Uint32(data)
	if got != 0xE98E0D0A {
		t.Fatalf("EOFRecord value = 0x%08X, want 0xE98E0D0A", got)
	}
}

func TestFLISRecord_ExactBytes(t *testing.T) {
	got := FLISRecord()
	want := []byte{
		'F', 'L', 'I', 'S',
		0x00, 0x00, 0x00, 0x08,
		0x00, 0x41, 0x00, 0x00,
		0x00, 0x00, 0x00, 0x00,
		0xFF, 0xFF, 0xFF, 0xFF,
		0x00, 0x01, 0x00, 0x03,
		0x00, 0x00, 0x00, 0x03,
		0x00, 0x00, 0x00, 0x01,
		0xFF, 0xFF, 0xFF, 0xFF,
	}
	if !bytes.Equal(got, want) {
		t.Fatalf("FLISRecord exact bytes mismatch:\n got: %x\nwant: %x", got, want)
	}
}

func TestFCISRecord_ExactBytes(t *testing.T) {
	got, err := FCISRecord(12345)
	if err != nil {
		t.Fatalf("FCISRecord returned error: %v", err)
	}
	want := []byte{
		'F', 'C', 'I', 'S',
		0x00, 0x00, 0x00, 0x14,
		0x00, 0x00, 0x00, 0x10,
		0x00, 0x00, 0x00, 0x01,
		0x00, 0x00, 0x00, 0x00,
		0x00, 0x00, 0x30, 0x39, // 12345
		0x00, 0x00, 0x00, 0x00,
		0x00, 0x00, 0x00, 0x20,
		0x00, 0x00, 0x00, 0x08,
		0x00, 0x01, 0x00, 0x01,
		0x00, 0x00, 0x00, 0x00,
	}
	if !bytes.Equal(got, want) {
		t.Fatalf("FCISRecord exact bytes mismatch:\n got: %x\nwant: %x", got, want)
	}
}

func TestFCISRecord_MaxTextLength(t *testing.T) {
	data, err := FCISRecord(0xFFFFFFFF)
	if err != nil {
		t.Fatalf("FCISRecord returned error: %v", err)
	}
	got := binary.BigEndian.Uint32(data[20:24])
	if got != 0xFFFFFFFF {
		t.Fatalf("textLength = 0x%08X, want 0xFFFFFFFF", got)
	}
}
