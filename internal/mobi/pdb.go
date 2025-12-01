package mobi

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"math"
	"time"
	"unicode/utf8"
)

// PalmEpochOffset represents the difference in seconds between Unix epoch and Palm epoch.
// Palm epoch starts at 1904-01-01 00:00:00 UTC.
const PalmEpochOffset = 2082844800

// PDBHeader represents the fixed 78-byte Palm Database header.
// All fields are encoded in big-endian order.
type PDBHeader struct {
	Name               [32]byte // Database name (31 bytes max, NULL padded)
	Attributes         uint16
	Version            uint16
	CreationDate       uint32
	ModificationDate   uint32
	BackupDate         uint32
	ModificationNumber uint32
	AppInfoOffset      uint32
	SortInfoOffset     uint32
	Type               [4]byte // "BOOK"
	Creator            [4]byte // "MOBI"
	UniqueSeed         uint32
	NextRecordList     uint32
	NumRecords         uint16
}

// RecordEntry represents a single record entry in the Palm Database record list.
type RecordEntry struct {
	Offset     uint32
	Attributes uint8
	UniqueID   [3]byte
}

// PDB contains the Palm Database header and associated record list entries.
type PDB struct {
	Header  PDBHeader
	Records []RecordEntry
}

// NewPDB creates a PDB header and record entries for the provided title and record sizes.
// creation and modification times default to the current UTC time when zero values are provided.
func NewPDB(title string, recordSizes []int, creation, modification time.Time) (*PDB, error) {
	if len(recordSizes) > math.MaxUint16 {
		return nil, fmt.Errorf("record count exceeds PalmDB limit: %d", len(recordSizes))
	}

	for i, size := range recordSizes {
		if size < 0 {
			return nil, fmt.Errorf("record size cannot be negative (index %d)", i)
		}
	}

	if creation.IsZero() {
		creation = time.Now().UTC()
	}
	if modification.IsZero() {
		modification = creation
	}

	records := buildRecordEntries(recordSizes)

	header := PDBHeader{
		Name:               truncateDatabaseName(title),
		Attributes:         0,
		Version:            0,
		CreationDate:       PalmEpochSeconds(creation),
		ModificationDate:   PalmEpochSeconds(modification),
		BackupDate:         0,
		ModificationNumber: 0,
		AppInfoOffset:      0,
		SortInfoOffset:     0,
		Type:               [4]byte{'B', 'O', 'O', 'K'},
		Creator:            [4]byte{'M', 'O', 'B', 'I'},
		UniqueSeed:         0,
		NextRecordList:     0,
		NumRecords:         uint16(len(records)),
	}

	return &PDB{
		Header:  header,
		Records: records,
	}, nil
}

// PalmEpochSeconds converts a time.Time to Palm epoch seconds.
func PalmEpochSeconds(t time.Time) uint32 {
	return uint32(t.Unix()) + PalmEpochOffset
}

// HeaderBytes encodes the PDB header into its 78-byte binary representation.
func (p *PDB) HeaderBytes() ([]byte, error) {
	buf := &bytes.Buffer{}
	fields := []interface{}{
		p.Header.Name,
		p.Header.Attributes,
		p.Header.Version,
		p.Header.CreationDate,
		p.Header.ModificationDate,
		p.Header.BackupDate,
		p.Header.ModificationNumber,
		p.Header.AppInfoOffset,
		p.Header.SortInfoOffset,
		p.Header.Type,
		p.Header.Creator,
		p.Header.UniqueSeed,
		p.Header.NextRecordList,
		p.Header.NumRecords,
	}

	for _, field := range fields {
		if err := binary.Write(buf, binary.BigEndian, field); err != nil {
			return nil, fmt.Errorf("failed to encode PDB header: %w", err)
		}
	}

	return buf.Bytes(), nil
}

// RecordListBytes encodes the record list entries followed by the 2-byte padding.
func (p *PDB) RecordListBytes() ([]byte, error) {
	buf := &bytes.Buffer{}

	for _, rec := range p.Records {
		if err := binary.Write(buf, binary.BigEndian, rec.Offset); err != nil {
			return nil, fmt.Errorf("failed to write record offset: %w", err)
		}
		if err := buf.WriteByte(rec.Attributes); err != nil {
			return nil, fmt.Errorf("failed to write record attributes: %w", err)
		}
		if _, err := buf.Write(rec.UniqueID[:]); err != nil {
			return nil, fmt.Errorf("failed to write record unique ID: %w", err)
		}
	}

	// Final 2-byte padding
	if err := binary.Write(buf, binary.BigEndian, uint16(0)); err != nil {
		return nil, fmt.Errorf("failed to write record list padding: %w", err)
	}

	return buf.Bytes(), nil
}

// buildRecordEntries generates record list entries and offsets using the specified sizes.
// Offsets follow the PalmDB rule:
//
//	first offset = 78 + (8 * record count) + 2
//	next offset = previous offset + previous record size
func buildRecordEntries(recordSizes []int) []RecordEntry {
	records := make([]RecordEntry, len(recordSizes))

	offset := uint32(78 + len(recordSizes)*8 + 2)
	for i, size := range recordSizes {
		records[i] = RecordEntry{
			Offset:     offset,
			Attributes: 0,
			UniqueID:   encodeUniqueID(uint32(i)),
		}
		offset += uint32(size)
	}

	return records
}

// truncateDatabaseName truncates the database name to 31 bytes and NULL pads to 32 bytes.
func truncateDatabaseName(name string) [32]byte {
	var result [32]byte

	var buf []byte
	for i := 0; i < len(name); {
		r, size := utf8.DecodeRuneInString(name[i:])
		if r == utf8.RuneError && size == 1 {
			// Treat invalid UTF-8 byte as-is to avoid silent drop
			size = 1
		}
		if len(buf)+size > 31 {
			break
		}
		buf = append(buf, name[i:i+size]...)
		i += size
	}

	copy(result[:], buf)
	return result
}

// encodeUniqueID converts a numeric ID into the 3-byte big-endian representation.
func encodeUniqueID(id uint32) [3]byte {
	return [3]byte{
		byte(id >> 16),
		byte(id >> 8),
		byte(id),
	}
}
