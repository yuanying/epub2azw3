package mobi

import (
	"bytes"
	"encoding/binary"
	"testing"
	"time"
)

func TestPalmEpochSeconds(t *testing.T) {
	unixZero := time.Unix(0, 0).UTC()
	if PalmEpochSeconds(unixZero) != PalmEpochOffset {
		t.Fatalf("PalmEpochSeconds(Unix epoch) = %d, want %d", PalmEpochSeconds(unixZero), PalmEpochOffset)
	}

	sampleTime := time.Date(2024, 1, 2, 3, 4, 5, 0, time.UTC)
	expected := uint32(sampleTime.Unix()) + PalmEpochOffset
	if PalmEpochSeconds(sampleTime) != expected {
		t.Fatalf("PalmEpochSeconds(%v) = %d, want %d", sampleTime, PalmEpochSeconds(sampleTime), expected)
	}
}

func TestPDBHeaderBytes(t *testing.T) {
	creation := time.Date(2024, 2, 3, 4, 5, 6, 0, time.UTC)
	modification := creation.Add(2 * time.Hour)
	title := "A very long book title that exceeds thirty-one bytes"

	pdb, err := NewPDB(title, []int{100, 200}, creation, modification)
	if err != nil {
		t.Fatalf("NewPDB returned error: %v", err)
	}

	headerBytes, err := pdb.HeaderBytes()
	if err != nil {
		t.Fatalf("HeaderBytes returned error: %v", err)
	}

	if len(headerBytes) != 78 {
		t.Fatalf("header length = %d, want 78", len(headerBytes))
	}

	// Name should be truncated to 31 bytes and NULL padded
	nameField := headerBytes[:32]
	expectedName := []byte(title)
	if len(expectedName) > 31 {
		expectedName = expectedName[:31]
	}

	for i, b := range expectedName {
		if nameField[i] != b {
			t.Fatalf("name byte %d = %d, want %d", i, nameField[i], b)
		}
	}
	for i := len(expectedName); i < 32; i++ {
		if nameField[i] != 0x00 {
			t.Fatalf("name padding byte %d = %d, want 0", i, nameField[i])
		}
	}

	if string(headerBytes[60:64]) != "BOOK" {
		t.Fatalf("Type field = %s, want BOOK", string(headerBytes[60:64]))
	}
	if string(headerBytes[64:68]) != "MOBI" {
		t.Fatalf("Creator field = %s, want MOBI", string(headerBytes[64:68]))
	}

	if got := binary.BigEndian.Uint16(headerBytes[76:78]); got != 2 {
		t.Fatalf("NumRecords = %d, want 2", got)
	}

	if got := binary.BigEndian.Uint32(headerBytes[36:40]); got != PalmEpochSeconds(creation) {
		t.Fatalf("CreationDate = %d, want %d", got, PalmEpochSeconds(creation))
	}

	if got := binary.BigEndian.Uint32(headerBytes[40:44]); got != PalmEpochSeconds(modification) {
		t.Fatalf("ModificationDate = %d, want %d", got, PalmEpochSeconds(modification))
	}
}

func TestPDBHeaderBytes_MultibyteAndEmptyTitle(t *testing.T) {
	tests := []struct {
		name         string
		title        string
		wantTrimmed  string
		wantLenBytes int
	}{
		{
			name:         "empty title",
			title:        "",
			wantTrimmed:  "",
			wantLenBytes: 0,
		},
		{
			name:         "multibyte title truncated without cutting rune",
			title:        "あいうえおかきくけこさ", // 11 chars, 33 bytes
			wantTrimmed:  "あいうえおかきくけこ",  // 10 chars, 30 bytes
			wantLenBytes: 30,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pdb, err := NewPDB(tt.title, nil, time.Now(), time.Now())
			if err != nil {
				t.Fatalf("NewPDB returned error: %v", err)
			}

			headerBytes, err := pdb.HeaderBytes()
			if err != nil {
				t.Fatalf("HeaderBytes returned error: %v", err)
			}

			if len(headerBytes) != 78 {
				t.Fatalf("header length = %d, want 78", len(headerBytes))
			}

			trimmed := bytes.TrimRight(headerBytes[:32], "\x00")
			if string(trimmed) != tt.wantTrimmed {
				t.Fatalf("trimmed title = %q, want %q", string(trimmed), tt.wantTrimmed)
			}
			if len(trimmed) != tt.wantLenBytes {
				t.Fatalf("trimmed title length = %d, want %d", len(trimmed), tt.wantLenBytes)
			}
		})
	}
}

func TestRecordListBytes(t *testing.T) {
	recordSizes := []int{100, 200, 50}
	creation := time.Date(2024, 5, 6, 7, 8, 9, 0, time.UTC)

	pdb, err := NewPDB("Sample Book", recordSizes, creation, creation)
	if err != nil {
		t.Fatalf("NewPDB returned error: %v", err)
	}

	recordList, err := pdb.RecordListBytes()
	if err != nil {
		t.Fatalf("RecordListBytes returned error: %v", err)
	}

	expectedLen := len(recordSizes)*8 + 2
	if len(recordList) != expectedLen {
		t.Fatalf("record list length = %d, want %d", len(recordList), expectedLen)
	}

	baseOffset := uint32(78 + len(recordSizes)*8 + 2)
	expectedOffsets := []uint32{
		baseOffset,
		baseOffset + uint32(recordSizes[0]),
		baseOffset + uint32(recordSizes[0]+recordSizes[1]),
	}

	for i := range recordSizes {
		start := i * 8
		offset := binary.BigEndian.Uint32(recordList[start : start+4])
		if offset != expectedOffsets[i] {
			t.Fatalf("record %d offset = %d, want %d", i, offset, expectedOffsets[i])
		}

		attrs := recordList[start+4]
		if attrs != 0x00 {
			t.Fatalf("record %d attributes = %d, want 0", i, attrs)
		}

		uid := uint32(recordList[start+5])<<16 | uint32(recordList[start+6])<<8 | uint32(recordList[start+7])
		if uid != uint32(i) {
			t.Fatalf("record %d unique ID = %d, want %d", i, uid, i)
		}
	}

	padding := binary.BigEndian.Uint16(recordList[expectedLen-2:])
	if padding != 0 {
		t.Fatalf("padding = %d, want 0", padding)
	}
}

func TestRecordListBytes_ZeroRecords(t *testing.T) {
	pdb, err := NewPDB("Empty", nil, time.Now(), time.Now())
	if err != nil {
		t.Fatalf("NewPDB returned error: %v", err)
	}

	recordList, err := pdb.RecordListBytes()
	if err != nil {
		t.Fatalf("RecordListBytes returned error: %v", err)
	}

	if len(recordList) != 2 {
		t.Fatalf("record list length = %d, want 2 (only padding)", len(recordList))
	}

	if got := binary.BigEndian.Uint16(recordList); got != 0 {
		t.Fatalf("padding = %d, want 0", got)
	}
}

func TestNewPDB_ZeroTimeDefaults(t *testing.T) {
	start := time.Now().UTC()
	pdb, err := NewPDB("Default Time", []int{1}, time.Time{}, time.Time{})
	if err != nil {
		t.Fatalf("NewPDB returned error: %v", err)
	}
	end := time.Now().UTC()

	creationUnix := int64(pdb.Header.CreationDate) - PalmEpochOffset
	modUnix := int64(pdb.Header.ModificationDate) - PalmEpochOffset

	if modUnix != creationUnix {
		t.Fatalf("modification date (%d) should match creation date (%d) when zero time provided", modUnix, creationUnix)
	}

	if creationUnix < start.Unix() || creationUnix > end.Unix() {
		t.Fatalf("creation date %d not within expected range [%d, %d]", creationUnix, start.Unix(), end.Unix())
	}
}
