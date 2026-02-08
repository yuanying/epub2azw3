package mobi

import "fmt"

// PalmDocCompressor implements the Compressor interface using PalmDoc (LZ77-based) compression.
type PalmDocCompressor struct{}

// Compress applies PalmDoc compression to the input data.
func (p *PalmDocCompressor) Compress(data []byte) ([]byte, error) {
	if len(data) == 0 {
		return nil, nil
	}

	out := make([]byte, 0, len(data))
	i := 0

	for i < len(data) {
		// Try back reference (need at least 3 bytes match, look back up to 2047 bytes)
		if bestLen, bestDist := findMatch(data, i); bestLen >= 3 {
			// Encode as 2-byte back reference: 0x80-0xBF range
			// High byte: 0x80 | (distance >> 5) (top bits of distance)
			// Low byte: ((distance & 0x1F) << 3) | (length - 3)
			high := byte(0x80 | (bestDist >> 5))
			low := byte(((bestDist & 0x1F) << 3) | (bestLen - 3))
			out = append(out, high, low)
			i += bestLen
			continue
		}

		// Try space + printable char encoding
		if data[i] == 0x20 && i+1 < len(data) && data[i+1] >= 0x40 && data[i+1] <= 0x7F {
			out = append(out, data[i+1]^0x80)
			i += 2
			continue
		}

		// Literal byte handling
		b := data[i]
		if b == 0x00 || (b >= 0x09 && b <= 0x7F) {
			// These bytes can be output as-is
			out = append(out, b)
			i++
		} else {
			// Bytes 0x01-0x08, 0x80-0xFF need to be wrapped in an uncompressed block
			// Collect consecutive bytes that need wrapping
			start := i
			for i < len(data) && (i-start) < 8 {
				b := data[i]
				if b == 0x00 || (b >= 0x09 && b <= 0x7F) {
					break
				}
				// Check for space+char opportunity
				if b == 0x20 && i+1 < len(data) && data[i+1] >= 0x40 && data[i+1] <= 0x7F {
					break
				}
				// Check for back reference opportunity
				if matchLen, _ := findMatch(data, i); matchLen >= 3 {
					break
				}
				i++
			}
			count := i - start
			out = append(out, byte(count))
			out = append(out, data[start:start+count]...)
		}
	}

	return out, nil
}

// Type returns the MOBI compression type identifier for PalmDoc compression.
func (p *PalmDocCompressor) Type() uint16 {
	return CompressionPalmDoc
}

// findMatch searches for the longest match in the sliding window.
// Returns (length, distance) where length >= 3 and distance <= 2047, or (0, 0) if no match.
func findMatch(data []byte, pos int) (int, int) {
	if pos+3 > len(data) {
		return 0, 0
	}

	maxDist := 2047
	if pos < maxDist {
		maxDist = pos
	}
	if maxDist == 0 {
		return 0, 0
	}

	bestLen := 0
	bestDist := 0
	maxLen := min(10, len(data)-pos) // PalmDoc max match length

	for dist := 1; dist <= maxDist; dist++ {
		start := pos - dist
		matchLen := 0
		for matchLen < maxLen && data[start+matchLen] == data[pos+matchLen] {
			matchLen++
		}
		if matchLen >= 3 && matchLen > bestLen {
			bestLen = matchLen
			bestDist = dist
			if bestLen == maxLen {
				break
			}
		}
	}

	return bestLen, bestDist
}

// PalmDocDecompress decompresses PalmDoc-compressed data.
func PalmDocDecompress(data []byte) ([]byte, error) {
	if len(data) == 0 {
		return nil, nil
	}

	out := make([]byte, 0, len(data)*2)
	i := 0

	for i < len(data) {
		b := data[i]
		i++

		switch {
		case b == 0x00:
			// Literal NULL byte
			out = append(out, 0x00)

		case b >= 0x01 && b <= 0x08:
			// Uncompressed block: next N bytes are literal
			count := int(b)
			if i+count > len(data) {
				return nil, fmt.Errorf("palmDoc decompress: uncompressed block overflows at offset %d", i-1)
			}
			out = append(out, data[i:i+count]...)
			i += count

		case b >= 0x09 && b <= 0x7F:
			// Literal byte
			out = append(out, b)

		case b >= 0x80 && b <= 0xBF:
			// Back reference (2 bytes)
			if i >= len(data) {
				return nil, fmt.Errorf("palmDoc decompress: back reference missing second byte at offset %d", i-1)
			}
			low := data[i]
			i++

			distance := (int(b&0x3F) << 5) | int(low>>3)
			length := int(low&0x07) + 3

			if distance == 0 || distance > len(out) {
				return nil, fmt.Errorf("palmDoc decompress: invalid back reference distance %d at output offset %d", distance, len(out))
			}

			start := len(out) - distance
			for j := range length {
				out = append(out, out[start+j])
			}

		case b >= 0xC0:
			// Space + literal char
			out = append(out, 0x20, b^0x80)
		}
	}

	return out, nil
}
