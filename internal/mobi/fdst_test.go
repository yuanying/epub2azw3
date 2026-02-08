package mobi

import (
	"encoding/binary"
	"testing"
)

func TestFDSTBytes_Identifier(t *testing.T) {
	fdst := NewFDSTSingleFlow(1000)
	data, err := fdst.Bytes()
	if err != nil {
		t.Fatalf("Bytes() returned error: %v", err)
	}

	if len(data) < 4 {
		t.Fatalf("data length = %d, want at least 4", len(data))
	}

	if string(data[:4]) != "FDST" {
		t.Fatalf("identifier = %q, want %q", string(data[:4]), "FDST")
	}
}

func TestFDSTSingleFlow(t *testing.T) {
	tests := []struct {
		name       string
		textLength uint32
	}{
		{"zero length", 0},
		{"small length", 12345},
		{"large length", 1000000},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fdst := NewFDSTSingleFlow(tt.textLength)

			data, err := fdst.Bytes()
			if err != nil {
				t.Fatalf("Bytes() returned error: %v", err)
			}

			// Check identifier
			if string(data[:4]) != "FDST" {
				t.Fatalf("identifier = %q, want %q", string(data[:4]), "FDST")
			}

			// Check entry count
			entryCount := binary.BigEndian.Uint32(data[4:8])
			if entryCount != 1 {
				t.Fatalf("entry count = %d, want 1", entryCount)
			}

			// Check offset table start position
			offsetTableStart := binary.BigEndian.Uint32(data[8:12])
			if offsetTableStart != 12 {
				t.Fatalf("offset table start = %d, want 12", offsetTableStart)
			}

			// Check entry offsets
			startOffset := binary.BigEndian.Uint32(data[12:16])
			if startOffset != 0 {
				t.Fatalf("entry start offset = %d, want 0", startOffset)
			}

			endOffset := binary.BigEndian.Uint32(data[16:20])
			if endOffset != tt.textLength {
				t.Fatalf("entry end offset = %d, want %d", endOffset, tt.textLength)
			}

			// Total length should be 20 bytes (4 + 4 + 4 + 8)
			if len(data) != 20 {
				t.Fatalf("data length = %d, want 20", len(data))
			}
		})
	}
}

func TestFDSTSize(t *testing.T) {
	fdst := NewFDSTSingleFlow(5000)

	data, err := fdst.Bytes()
	if err != nil {
		t.Fatalf("Bytes() returned error: %v", err)
	}

	if fdst.Size() != len(data) {
		t.Fatalf("Size() = %d, len(Bytes()) = %d, want equal", fdst.Size(), len(data))
	}
}

func TestFDSTFlowCount(t *testing.T) {
	fdst := NewFDSTSingleFlow(1000)

	if fdst.FlowCount() != 1 {
		t.Fatalf("FlowCount() = %d, want 1", fdst.FlowCount())
	}
}

func TestFDSTFlowCount_MultipleEntries(t *testing.T) {
	fdst := &FDSTRecord{
		Entries: [][2]uint32{
			{0, 1000},
			{1000, 2000},
			{2000, 3000},
		},
	}

	if fdst.FlowCount() != 3 {
		t.Fatalf("FlowCount() = %d, want 3", fdst.FlowCount())
	}

	if fdst.Size() != 12+3*8 {
		t.Fatalf("Size() = %d, want %d", fdst.Size(), 12+3*8)
	}
}

func TestFDSTBytes_MultipleFlows(t *testing.T) {
	fdst := &FDSTRecord{
		Entries: [][2]uint32{
			{0, 10},
			{10, 25},
			{25, 40},
		},
	}
	data, err := fdst.Bytes()
	if err != nil {
		t.Fatalf("Bytes() returned error: %v", err)
	}

	// Identifier
	if string(data[:4]) != "FDST" {
		t.Fatalf("identifier = %q, want %q", string(data[:4]), "FDST")
	}

	// Entry count
	if got := binary.BigEndian.Uint32(data[4:8]); got != 3 {
		t.Fatalf("entry count = %d, want 3", got)
	}

	// Offset table start
	if got := binary.BigEndian.Uint32(data[8:12]); got != 12 {
		t.Fatalf("offset table start = %d, want 12", got)
	}

	// Verify each entry's start and end offsets
	expected := [][2]uint32{{0, 10}, {10, 25}, {25, 40}}
	for i, want := range expected {
		off := 12 + i*8
		gotStart := binary.BigEndian.Uint32(data[off : off+4])
		gotEnd := binary.BigEndian.Uint32(data[off+4 : off+8])
		if gotStart != want[0] {
			t.Fatalf("entry %d start = %d, want %d", i, gotStart, want[0])
		}
		if gotEnd != want[1] {
			t.Fatalf("entry %d end = %d, want %d", i, gotEnd, want[1])
		}
	}

	// Total length
	if len(data) != 12+3*8 {
		t.Fatalf("data length = %d, want %d", len(data), 12+3*8)
	}
}
