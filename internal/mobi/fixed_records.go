package mobi

import (
	"bytes"
	"encoding/binary"
	"fmt"
)

// FLISRecord generates a 36-byte FLIS (Fixed Layout Indicator Structure) record.
// All fields are fixed values as defined by the AZW3/MOBI format specification.
func FLISRecord() []byte {
	buf := &bytes.Buffer{}
	fields := []any{
		[4]byte{'F', 'L', 'I', 'S'},
		uint32(0x00000008),
		uint16(0x0041),
		uint16(0x0000),
		uint32(0x00000000),
		uint32(0xFFFFFFFF),
		uint16(0x0001),
		uint16(0x0003),
		uint32(0x00000003),
		uint32(0x00000001),
		uint32(0xFFFFFFFF),
	}
	for _, f := range fields {
		_ = binary.Write(buf, binary.BigEndian, f)
	}
	return buf.Bytes()
}

// FCISRecord generates a 44-byte FCIS (Fixed Content Indicator Structure) record.
// The textLength parameter is written at offset 20 to indicate the total uncompressed text length.
func FCISRecord(textLength uint32) ([]byte, error) {
	buf := &bytes.Buffer{}
	fields := []any{
		[4]byte{'F', 'C', 'I', 'S'},
		uint32(0x00000014),
		uint32(0x00000010),
		uint32(0x00000001),
		uint32(0x00000000),
		textLength,
		uint32(0x00000000),
		uint32(0x00000020),
		uint32(0x00000008),
		uint16(0x0001),
		uint16(0x0001),
		uint32(0x00000000),
	}
	for _, f := range fields {
		if err := binary.Write(buf, binary.BigEndian, f); err != nil {
			return nil, fmt.Errorf("failed to write FCIS record: %w", err)
		}
	}
	return buf.Bytes(), nil
}

// EOFRecord generates a 4-byte end-of-file record with the magic value 0xE98E0D0A.
func EOFRecord() []byte {
	buf := make([]byte, 4)
	binary.BigEndian.PutUint32(buf, 0xE98E0D0A)
	return buf
}
