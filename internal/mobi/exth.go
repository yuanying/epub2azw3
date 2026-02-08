package mobi

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"math"

	"github.com/yuanying/epub2azw3/internal/epub"
)

// EXTHRecord represents a single EXTH metadata record.
type EXTHRecord struct {
	Type uint32
	Data []byte
}

// EXTHHeader represents the EXTH header containing metadata records.
// Records[0] is reserved for KF8 boundary offset (type 121).
// Records[1] is reserved for KF8 record count (type 125).
type EXTHHeader struct {
	Records []EXTHRecord
}

// NewEXTHHeader creates an EXTHHeader with KF8 mandatory records (type 121, 125)
// pre-populated at indices 0 and 1.
func NewEXTHHeader(boundaryOffset, recordCount uint32) *EXTHHeader {
	h := &EXTHHeader{
		Records: make([]EXTHRecord, 2),
	}
	h.Records[0] = makeUint32Record(121, boundaryOffset)
	h.Records[1] = makeUint32Record(125, recordCount)
	return h
}

// AddStringRecord appends a UTF-8 string metadata record.
func (h *EXTHHeader) AddStringRecord(recordType uint32, value string) {
	h.Records = append(h.Records, EXTHRecord{
		Type: recordType,
		Data: []byte(value),
	})
}

// AddUint32Record appends a 4-byte unsigned integer metadata record.
func (h *EXTHHeader) AddUint32Record(recordType uint32, value uint32) {
	h.Records = append(h.Records, makeUint32Record(recordType, value))
}

// SetBoundaryOffset updates the KF8 boundary offset record (Records[0]).
// This value represents the PDB record index of the KF8 boundary, not the EXTH record count.
// It must be set by the caller after all PDB records are assembled (typically in Task 06 integration).
func (h *EXTHHeader) SetBoundaryOffset(offset uint32) {
	h.Records[0] = makeUint32Record(121, offset)
}

// SetRecordCount updates the KF8 total record count record (Records[1]).
// This value represents the total number of PDB records, not the EXTH record count.
// It must be set by the caller after all PDB records are assembled (typically in Task 06 integration).
func (h *EXTHHeader) SetRecordCount(count uint32) {
	h.Records[1] = makeUint32Record(125, count)
}

// Bytes serializes the EXTH header to its binary representation.
// Format: "EXTH"(4) + headerLength(4) + recordCount(4) + records + padding
func (h *EXTHHeader) Bytes() ([]byte, error) {
	if len(h.Records) < 2 {
		return nil, fmt.Errorf("EXTH requires at least 2 KF8 mandatory records (type 121, 125)")
	}

	totalSize := h.Size()
	if totalSize > math.MaxUint32 {
		return nil, fmt.Errorf("EXTH header too large: %d bytes", totalSize)
	}

	buf := bytes.NewBuffer(make([]byte, 0, totalSize))

	// Write identifier
	if _, err := buf.WriteString("EXTH"); err != nil {
		return nil, fmt.Errorf("failed to write EXTH identifier: %w", err)
	}

	// Write header length (includes padding)
	if err := binary.Write(buf, binary.BigEndian, uint32(totalSize)); err != nil {
		return nil, fmt.Errorf("failed to write EXTH header length: %w", err)
	}

	// Write record count
	if err := binary.Write(buf, binary.BigEndian, uint32(len(h.Records))); err != nil {
		return nil, fmt.Errorf("failed to write EXTH record count: %w", err)
	}

	// Write records
	for _, rec := range h.Records {
		if err := binary.Write(buf, binary.BigEndian, rec.Type); err != nil {
			return nil, fmt.Errorf("failed to write EXTH record type: %w", err)
		}
		recLen := uint32(8 + len(rec.Data))
		if err := binary.Write(buf, binary.BigEndian, recLen); err != nil {
			return nil, fmt.Errorf("failed to write EXTH record length: %w", err)
		}
		if _, err := buf.Write(rec.Data); err != nil {
			return nil, fmt.Errorf("failed to write EXTH record data: %w", err)
		}
	}

	// Write padding
	padding := totalSize - (12 + h.recordsDataSize())
	for i := 0; i < padding; i++ {
		if err := buf.WriteByte(0x00); err != nil {
			return nil, fmt.Errorf("failed to write EXTH padding: %w", err)
		}
	}

	return buf.Bytes(), nil
}

// Size returns the total serialized size in bytes, including padding.
func (h *EXTHHeader) Size() int {
	unpaddedSize := 12 + h.recordsDataSize()
	padding := (4 - (unpaddedSize % 4)) % 4
	return unpaddedSize + padding
}

// recordsDataSize returns the total byte size of all records (Type + Length + Data).
func (h *EXTHHeader) recordsDataSize() int {
	size := 0
	for _, rec := range h.Records {
		size += 8 + len(rec.Data)
	}
	return size
}

// EXTHFromMetadata creates an EXTHHeader populated from EPUB metadata.
// Empty fields are skipped.
func EXTHFromMetadata(meta epub.Metadata, boundaryOffset, recordCount uint32) *EXTHHeader {
	h := NewEXTHHeader(boundaryOffset, recordCount)

	// Creators → type 100 (each Creator's Name)
	for _, c := range meta.Creators {
		if c.Name != "" {
			h.AddStringRecord(100, c.Name)
		}
	}

	// Publisher → type 101
	if meta.Publisher != "" {
		h.AddStringRecord(101, meta.Publisher)
	}

	// Description → type 103
	if meta.Description != "" {
		h.AddStringRecord(103, meta.Description)
	}

	// Identifier → type 104
	if meta.Identifier != "" {
		h.AddStringRecord(104, meta.Identifier)
	}

	// Subjects → type 105 (each Subject)
	for _, s := range meta.Subjects {
		if s != "" {
			h.AddStringRecord(105, s)
		}
	}

	// Date → type 106
	if meta.Date != "" {
		h.AddStringRecord(106, meta.Date)
	}

	// Rights → type 109
	if meta.Rights != "" {
		h.AddStringRecord(109, meta.Rights)
	}

	// Title → type 503
	if meta.Title != "" {
		h.AddStringRecord(503, meta.Title)
	}

	// Language → type 524
	if meta.Language != "" {
		h.AddStringRecord(524, meta.Language)
	}

	return h
}

// makeUint32Record creates an EXTHRecord with a 4-byte big-endian uint32 value.
func makeUint32Record(recordType, value uint32) EXTHRecord {
	data := make([]byte, 4)
	binary.BigEndian.PutUint32(data, value)
	return EXTHRecord{
		Type: recordType,
		Data: data,
	}
}
