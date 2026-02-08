package mobi

import (
	"bytes"
	"strings"
	"testing"
)

func TestPalmDocCompressor_Type(t *testing.T) {
	c := &PalmDocCompressor{}
	if got := c.Type(); got != CompressionPalmDoc {
		t.Fatalf("PalmDocCompressor.Type() = %d, want %d", got, CompressionPalmDoc)
	}
}

func TestPalmDocRoundTrip_Empty(t *testing.T) {
	c := &PalmDocCompressor{}
	compressed, err := c.Compress(nil)
	if err != nil {
		t.Fatalf("Compress(nil) error: %v", err)
	}
	decompressed, err := PalmDocDecompress(compressed)
	if err != nil {
		t.Fatalf("Decompress error: %v", err)
	}
	if len(decompressed) != 0 {
		t.Fatalf("Decompress(Compress(nil)) = %d bytes, want 0", len(decompressed))
	}
}

func TestPalmDocRoundTrip_LiteralOnly(t *testing.T) {
	c := &PalmDocCompressor{}
	// All bytes in the literal range (0x09-0x7F) should survive round-trip
	data := []byte("Hello, World! This is a test.")
	compressed, err := c.Compress(data)
	if err != nil {
		t.Fatalf("Compress error: %v", err)
	}
	decompressed, err := PalmDocDecompress(compressed)
	if err != nil {
		t.Fatalf("Decompress error: %v", err)
	}
	if !bytes.Equal(decompressed, data) {
		t.Fatalf("round-trip mismatch:\n  got:  %q\n  want: %q", decompressed, data)
	}
}

func TestPalmDocRoundTrip_BackReference(t *testing.T) {
	c := &PalmDocCompressor{}
	// Repeated text should trigger back references
	data := []byte("abcdefghij abcdefghij abcdefghij")
	compressed, err := c.Compress(data)
	if err != nil {
		t.Fatalf("Compress error: %v", err)
	}
	// Compressed should be smaller than original due to back references
	if len(compressed) >= len(data) {
		t.Logf("warning: compressed size %d >= original %d (may not have found back refs)", len(compressed), len(data))
	}
	decompressed, err := PalmDocDecompress(compressed)
	if err != nil {
		t.Fatalf("Decompress error: %v", err)
	}
	if !bytes.Equal(decompressed, data) {
		t.Fatalf("round-trip mismatch:\n  got:  %q\n  want: %q", decompressed, data)
	}
}

func TestPalmDocRoundTrip_SpacePlusChar(t *testing.T) {
	c := &PalmDocCompressor{}
	// Spaces followed by printable ASCII should use space+char encoding
	data := []byte("word word word word")
	compressed, err := c.Compress(data)
	if err != nil {
		t.Fatalf("Compress error: %v", err)
	}
	decompressed, err := PalmDocDecompress(compressed)
	if err != nil {
		t.Fatalf("Decompress error: %v", err)
	}
	if !bytes.Equal(decompressed, data) {
		t.Fatalf("round-trip mismatch:\n  got:  %q\n  want: %q", decompressed, data)
	}
}

func TestPalmDocRoundTrip_BoundarySize(t *testing.T) {
	c := &PalmDocCompressor{}
	// Exactly 4096 bytes - the max record size
	data := make([]byte, RecordSize)
	for i := range data {
		data[i] = byte('A' + (i % 26))
	}
	compressed, err := c.Compress(data)
	if err != nil {
		t.Fatalf("Compress error: %v", err)
	}
	decompressed, err := PalmDocDecompress(compressed)
	if err != nil {
		t.Fatalf("Decompress error: %v", err)
	}
	if !bytes.Equal(decompressed, data) {
		t.Fatalf("round-trip mismatch: got %d bytes, want %d bytes", len(decompressed), len(data))
	}
}

func TestPalmDocRoundTrip_JapaneseUTF8(t *testing.T) {
	c := &PalmDocCompressor{}
	data := []byte("これはテストです。日本語のテキストを圧縮して、正しく復元できるか確認します。")
	compressed, err := c.Compress(data)
	if err != nil {
		t.Fatalf("Compress error: %v", err)
	}
	decompressed, err := PalmDocDecompress(compressed)
	if err != nil {
		t.Fatalf("Decompress error: %v", err)
	}
	if !bytes.Equal(decompressed, data) {
		t.Fatalf("round-trip mismatch:\n  got:  %q\n  want: %q", decompressed, data)
	}
}

func TestPalmDocRoundTrip_SpecialBytes(t *testing.T) {
	c := &PalmDocCompressor{}
	// Test bytes that need special handling: 0x00, 0x01-0x08
	data := []byte{0x00, 0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08}
	compressed, err := c.Compress(data)
	if err != nil {
		t.Fatalf("Compress error: %v", err)
	}
	decompressed, err := PalmDocDecompress(compressed)
	if err != nil {
		t.Fatalf("Decompress error: %v", err)
	}
	if !bytes.Equal(decompressed, data) {
		t.Fatalf("round-trip mismatch:\n  got:  %x\n  want: %x", decompressed, data)
	}
}

func TestPalmDocRoundTrip_HTMLContent(t *testing.T) {
	c := &PalmDocCompressor{}
	data := []byte(`<html><head><style>body { margin: 0; }</style></head><body>
<div id="ch01" class="vrtl"><mbp:pagebreak/>
<h1>第一章</h1>
<p>これは本文です。同じ文字列が繰り返されます。同じ文字列が繰り返されます。</p>
</div>
</body></html>`)
	compressed, err := c.Compress(data)
	if err != nil {
		t.Fatalf("Compress error: %v", err)
	}
	decompressed, err := PalmDocDecompress(compressed)
	if err != nil {
		t.Fatalf("Decompress error: %v", err)
	}
	if !bytes.Equal(decompressed, data) {
		t.Fatalf("round-trip mismatch:\n  got:  %q\n  want: %q", decompressed, data)
	}
}

func TestPalmDocRoundTrip_LargeRepeating(t *testing.T) {
	c := &PalmDocCompressor{}
	// Large input with lots of repetition - should compress well
	data := []byte(strings.Repeat("The quick brown fox jumps over the lazy dog. ", 80))
	compressed, err := c.Compress(data)
	if err != nil {
		t.Fatalf("Compress error: %v", err)
	}
	if len(compressed) >= len(data) {
		t.Errorf("compression did not reduce size: compressed=%d original=%d", len(compressed), len(data))
	}
	decompressed, err := PalmDocDecompress(compressed)
	if err != nil {
		t.Fatalf("Decompress error: %v", err)
	}
	if !bytes.Equal(decompressed, data) {
		t.Fatalf("round-trip mismatch: got %d bytes, want %d bytes", len(decompressed), len(data))
	}
}

func TestPalmDocCompressor_ImplementsCompressor(t *testing.T) {
	// Verify PalmDocCompressor implements the Compressor interface
	var _ Compressor = &PalmDocCompressor{}
}

func TestSplitTextRecords_WithPalmDoc(t *testing.T) {
	// Test SplitTextRecords with PalmDoc compression
	data := []byte(strings.Repeat("Hello World! This is a test of PalmDoc compression. ", 200))
	compressor := &PalmDocCompressor{}

	records, err := SplitTextRecords(data, compressor)
	if err != nil {
		t.Fatalf("SplitTextRecords error: %v", err)
	}

	expectedCount := TextRecordCount(data)
	if len(records) != expectedCount {
		t.Fatalf("got %d records, want %d", len(records), expectedCount)
	}

	// Verify each record decompresses correctly
	for i, rec := range records {
		offset := i * RecordSize
		end := min(offset+RecordSize, len(data))
		expected := data[offset:end]

		decompressed, err := PalmDocDecompress(rec)
		if err != nil {
			t.Fatalf("record %d: Decompress error: %v", i, err)
		}
		if !bytes.Equal(decompressed, expected) {
			t.Fatalf("record %d: decompressed mismatch: got %d bytes, want %d bytes", i, len(decompressed), len(expected))
		}
	}
}
