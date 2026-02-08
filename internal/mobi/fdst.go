package mobi

import (
	"bytes"
	"encoding/binary"
	"fmt"
)

// fdstIdentifier is the 4-byte magic identifier for FDST records.
var fdstIdentifier = [4]byte{'F', 'D', 'S', 'T'}

// fdstOffsetTableStart is the fixed byte offset where the entry table begins.
const fdstOffsetTableStart uint32 = 12

// FDSTRecord represents a Flow Descriptor Table (FDST) record in AZW3/KF8 format.
// Each entry defines a flow section by its start and end byte offsets.
type FDSTRecord struct {
	Entries [][2]uint32
}

// NewFDSTSingleFlow creates an FDSTRecord with a single flow entry
// spanning from offset 0 to the given textLength.
func NewFDSTSingleFlow(textLength uint32) *FDSTRecord {
	return &FDSTRecord{
		Entries: [][2]uint32{{0, textLength}},
	}
}

// Bytes serializes the FDSTRecord into its binary representation.
// Layout:
//  1. "FDST" identifier (4 bytes)
//  2. Entry count (uint32)
//  3. Offset table start position = 12 (uint32)
//  4. Entry pairs: each is [start offset (uint32), end offset (uint32)]
func (f *FDSTRecord) Bytes() ([]byte, error) {
	buf := bytes.NewBuffer(make([]byte, 0, f.Size()))

	if err := binary.Write(buf, binary.BigEndian, fdstIdentifier); err != nil {
		return nil, fmt.Errorf("failed to write FDST identifier: %w", err)
	}

	if err := binary.Write(buf, binary.BigEndian, uint32(len(f.Entries))); err != nil {
		return nil, fmt.Errorf("failed to write FDST entry count: %w", err)
	}

	if err := binary.Write(buf, binary.BigEndian, fdstOffsetTableStart); err != nil {
		return nil, fmt.Errorf("failed to write FDST offset table start: %w", err)
	}

	for i, entry := range f.Entries {
		if err := binary.Write(buf, binary.BigEndian, entry[0]); err != nil {
			return nil, fmt.Errorf("failed to write FDST entry %d start: %w", i, err)
		}
		if err := binary.Write(buf, binary.BigEndian, entry[1]); err != nil {
			return nil, fmt.Errorf("failed to write FDST entry %d end: %w", i, err)
		}
	}

	return buf.Bytes(), nil
}

// Size returns the byte size of the serialized FDST record.
func (f *FDSTRecord) Size() int {
	return 12 + len(f.Entries)*8
}

// FlowCount returns the number of flow entries.
func (f *FDSTRecord) FlowCount() uint32 {
	return uint32(len(f.Entries))
}
