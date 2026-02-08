package mobi

// RecordSize is the maximum size in bytes of a single text record.
const RecordSize = 4096

// Compressor defines the interface for text record compression.
type Compressor interface {
	Compress(data []byte) ([]byte, error)
	Type() uint16
}

// NoCompression implements Compressor with no compression (type 1).
type NoCompression struct{}

// Compress returns a copy of the input data without modification.
func (n *NoCompression) Compress(data []byte) ([]byte, error) {
	out := make([]byte, len(data))
	copy(out, data)
	return out, nil
}

// Type returns the MOBI compression type identifier for no compression.
func (n *NoCompression) Type() uint16 {
	return 1
}

// SplitTextRecords splits the HTML content into RecordSize-byte chunks and applies
// the given compressor to each chunk. If compressor is nil, NoCompression is used.
// Note: compressed output may exceed RecordSize depending on the compressor implementation.
func SplitTextRecords(html []byte, compressor Compressor) ([][]byte, error) {
	if len(html) == 0 {
		return nil, nil
	}

	if compressor == nil {
		compressor = &NoCompression{}
	}

	count := TextRecordCount(html)
	records := make([][]byte, 0, count)

	for offset := 0; offset < len(html); offset += RecordSize {
		end := min(offset+RecordSize, len(html))
		chunk := html[offset:end]
		compressed, err := compressor.Compress(chunk)
		if err != nil {
			return nil, err
		}
		records = append(records, compressed)
	}

	return records, nil
}

// TextLength returns the total byte length of the HTML content as uint32.
func TextLength(html []byte) uint32 {
	return uint32(len(html))
}

// TextRecordCount returns the number of records needed to store the HTML content.
func TextRecordCount(html []byte) int {
	n := len(html)
	if n == 0 {
		return 0
	}
	return (n + RecordSize - 1) / RecordSize
}
