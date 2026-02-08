package mobi

import (
	"bytes"
	"testing"
)

func TestNoCompression(t *testing.T) {
	nc := &NoCompression{}

	if nc.Type() != 1 {
		t.Fatalf("NoCompression.Type() = %d, want 1", nc.Type())
	}

	data := []byte("hello world")
	compressed, err := nc.Compress(data)
	if err != nil {
		t.Fatalf("NoCompression.Compress returned error: %v", err)
	}

	if !bytes.Equal(compressed, data) {
		t.Fatalf("NoCompression.Compress returned %q, want %q", compressed, data)
	}
}

func TestSplitTextRecords(t *testing.T) {
	tests := []struct {
		name      string
		dataSize  int
		wantCount int
	}{
		{
			name:      "empty data returns 0 records",
			dataSize:  0,
			wantCount: 0,
		},
		{
			name:      "100 bytes returns 1 record",
			dataSize:  100,
			wantCount: 1,
		},
		{
			name:      "exactly 4096 bytes returns 1 record",
			dataSize:  4096,
			wantCount: 1,
		},
		{
			name:      "4097 bytes returns 2 records",
			dataSize:  4097,
			wantCount: 2,
		},
		{
			name:      "12288 bytes (3x4096) returns 3 records",
			dataSize:  12288,
			wantCount: 3,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			data := make([]byte, tt.dataSize)
			for i := range data {
				data[i] = byte(i % 256)
			}

			records, err := SplitTextRecords(data, &NoCompression{})
			if err != nil {
				t.Fatalf("SplitTextRecords returned error: %v", err)
			}

			if len(records) != tt.wantCount {
				t.Fatalf("SplitTextRecords returned %d records, want %d", len(records), tt.wantCount)
			}
		})
	}
}

func TestSplitTextRecords_DataIntegrity(t *testing.T) {
	data := make([]byte, 10000)
	for i := range data {
		data[i] = byte(i % 256)
	}

	records, err := SplitTextRecords(data, &NoCompression{})
	if err != nil {
		t.Fatalf("SplitTextRecords returned error: %v", err)
	}

	var reconstructed []byte
	for _, rec := range records {
		reconstructed = append(reconstructed, rec...)
	}

	if !bytes.Equal(reconstructed, data) {
		t.Fatalf("reconstructed data does not match original: got %d bytes, want %d bytes", len(reconstructed), len(data))
	}
}

func TestSplitTextRecords_NilCompressor(t *testing.T) {
	data := []byte("hello world")

	records, err := SplitTextRecords(data, nil)
	if err != nil {
		t.Fatalf("SplitTextRecords with nil compressor returned error: %v", err)
	}

	if len(records) != 1 {
		t.Fatalf("SplitTextRecords returned %d records, want 1", len(records))
	}

	if !bytes.Equal(records[0], data) {
		t.Fatalf("record data = %q, want %q", records[0], data)
	}
}

func TestSplitTextRecords_ChunkSizes(t *testing.T) {
	data := make([]byte, RecordSize+1)
	for i := range data {
		data[i] = byte(i % 256)
	}

	records, err := SplitTextRecords(data, &NoCompression{})
	if err != nil {
		t.Fatalf("SplitTextRecords returned error: %v", err)
	}

	if len(records) != 2 {
		t.Fatalf("got %d records, want 2", len(records))
	}

	if len(records[0]) != RecordSize {
		t.Fatalf("first record size = %d, want %d", len(records[0]), RecordSize)
	}

	if len(records[1]) != 1 {
		t.Fatalf("second record size = %d, want 1", len(records[1]))
	}
}

type countingCompressor struct{ calls int }

func (c *countingCompressor) Compress(b []byte) ([]byte, error) {
	c.calls++
	out := make([]byte, len(b))
	copy(out, b)
	return out, nil
}

func (c *countingCompressor) Type() uint16 { return 1 }

func TestSplitTextRecords_CompressorCalledPerChunk(t *testing.T) {
	data := make([]byte, RecordSize*2+1)
	cc := &countingCompressor{}

	_, err := SplitTextRecords(data, cc)
	if err != nil {
		t.Fatalf("SplitTextRecords returned error: %v", err)
	}

	if cc.calls != 3 {
		t.Fatalf("compressor calls = %d, want 3", cc.calls)
	}
}

func TestTextLength(t *testing.T) {
	tests := []struct {
		name string
		data []byte
		want uint32
	}{
		{"nil data", nil, 0},
		{"empty data", []byte{}, 0},
		{"non-empty data", []byte("hello"), 5},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := TextLength(tt.data); got != tt.want {
				t.Fatalf("TextLength() = %d, want %d", got, tt.want)
			}
		})
	}
}

func TestTextRecordCount(t *testing.T) {
	tests := []struct {
		name string
		size int
		want int
	}{
		{"empty", 0, 0},
		{"100 bytes", 100, 1},
		{"4096 bytes", 4096, 1},
		{"4097 bytes", 4097, 2},
		{"12288 bytes", 12288, 3},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			data := make([]byte, tt.size)
			if got := TextRecordCount(data); got != tt.want {
				t.Fatalf("TextRecordCount() = %d, want %d", got, tt.want)
			}
		})
	}
}
