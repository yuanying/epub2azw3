package mobi

import (
	"bytes"
	"crypto/rand"
	"encoding/binary"
	"fmt"
)

// languageCodeMap maps BCP 47 language tags to MOBI language codes.
var languageCodeMap = map[string]uint32{
	"en": 0x0409, // English (US)
	"ja": 0x0411, // Japanese
	"de": 0x0407, // German
	"fr": 0x040C, // French
	"es": 0x040A, // Spanish
	"it": 0x0410, // Italian
	"pt": 0x0416, // Portuguese (Brazil)
	"zh": 0x0804, // Chinese (Simplified)
	"ko": 0x0412, // Korean
	"nl": 0x0413, // Dutch
	"ru": 0x0419, // Russian
}

const (
	// defaultLanguageCode is used when the language is not found in the map.
	defaultLanguageCode = 0x0409

	// CompressionNone indicates no compression.
	CompressionNone uint16 = 1
	// CompressionPalmDoc indicates PalmDoc compression.
	CompressionPalmDoc uint16 = 2

	// MaxRecordSize is the maximum size of a single text record (4096 bytes).
	MaxRecordSize uint16 = 4096

	// PalmDOCHeaderSize is the size of the PalmDOC header in bytes.
	PalmDOCHeaderSize = 16

	// MOBIHeaderSize is the size of the MOBI header in bytes (KF8).
	MOBIHeaderSize = 248

	// EncodingUTF8 is the MOBI encoding code for UTF-8.
	EncodingUTF8 uint32 = 65001

	// MOBITypeKF8 is the MOBI type for KF8 format.
	MOBITypeKF8 uint32 = 248

	// FileVersionKF8 is the file version for KF8 format.
	FileVersionKF8 uint32 = 8

	// EXTHFlagPresent indicates that EXTH records are present.
	EXTHFlagPresent uint32 = 0x40
)

// LanguageCode converts a BCP 47 language tag to a MOBI language code.
// Returns defaultLanguageCode (English US) for unknown or empty strings.
func LanguageCode(lang string) uint32 {
	if code, ok := languageCodeMap[lang]; ok {
		return code
	}
	return defaultLanguageCode
}

// MOBIHeaderConfig holds the configurable parameters for creating a MOBIHeader.
type MOBIHeaderConfig struct {
	Compression          uint16
	TextLength           uint32
	TextRecordCount      uint16
	Language             string
	FirstImageIndex      uint32
	FirstContentRecord   uint16
	LastContentRecord    uint16
	FCISRecordNumber     uint32
	FLISRecordNumber     uint32
	ExtraRecordDataFlags uint32
	FDSTFlowCount        uint32
	FDSTOffset           uint32
}

// MOBIHeader represents the internal state of a MOBI header for Record 0.
type MOBIHeader struct {
	Compression          uint16
	TextLength           uint32
	TextRecordCount      uint16
	LanguageCode         uint32
	UniqueID             uint32
	FirstImageIndex      uint32
	FirstContentRecord   uint16
	LastContentRecord    uint16
	FCISRecordNumber     uint32
	FLISRecordNumber     uint32
	ExtraRecordDataFlags uint32
	FDSTFlowCount        uint32
	FDSTOffset           uint32
}

// NewMOBIHeader creates a MOBIHeader from the given configuration.
// It generates a random UniqueID using crypto/rand.
func NewMOBIHeader(cfg MOBIHeaderConfig) (*MOBIHeader, error) {
	uid, err := generateUniqueID()
	if err != nil {
		return nil, fmt.Errorf("failed to generate unique ID: %w", err)
	}

	return &MOBIHeader{
		Compression:          cfg.Compression,
		TextLength:           cfg.TextLength,
		TextRecordCount:      cfg.TextRecordCount,
		LanguageCode:         LanguageCode(cfg.Language),
		UniqueID:             uid,
		FirstImageIndex:      cfg.FirstImageIndex,
		FirstContentRecord:   cfg.FirstContentRecord,
		LastContentRecord:    cfg.LastContentRecord,
		FCISRecordNumber:     cfg.FCISRecordNumber,
		FLISRecordNumber:     cfg.FLISRecordNumber,
		ExtraRecordDataFlags: cfg.ExtraRecordDataFlags,
		FDSTFlowCount:        cfg.FDSTFlowCount,
		FDSTOffset:           cfg.FDSTOffset,
	}, nil
}

// PalmDOCHeaderBytes serializes the 16-byte PalmDOC header.
func (h *MOBIHeader) PalmDOCHeaderBytes() ([]byte, error) {
	buf := &bytes.Buffer{}
	fields := []interface{}{
		h.Compression,     // 0: compression type
		uint16(0),         // 2: unused
		h.TextLength,      // 4: text length
		h.TextRecordCount, // 8: text record count
		MaxRecordSize,     // 10: max record size (4096)
		uint16(0),         // 12: encryption type (none)
		uint16(0),         // 14: unused
	}

	for _, field := range fields {
		if err := binary.Write(buf, binary.BigEndian, field); err != nil {
			return nil, fmt.Errorf("failed to encode PalmDOC header: %w", err)
		}
	}

	return buf.Bytes(), nil
}

// generateUniqueID generates a random uint32 using crypto/rand.
func generateUniqueID() (uint32, error) {
	var b [4]byte
	if _, err := rand.Read(b[:]); err != nil {
		return 0, fmt.Errorf("failed to read random bytes: %w", err)
	}
	return binary.BigEndian.Uint32(b[:]), nil
}
