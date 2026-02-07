package mobi

import (
	"bytes"
	"crypto/rand"
	"encoding/binary"
	"fmt"
	"math"
	"strings"
)

// languageCodeMap maps BCP 47 language tags to MOBI language codes (Windows LCID).
// Full regional tags (e.g. "en-gb") are checked first, then primary subtag (e.g. "en").
var languageCodeMap = map[string]uint32{
	// Primary language tags
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
	// Regional variants
	"en-gb": 0x0809, // English (UK)
	"en-au": 0x0C09, // English (Australia)
	"pt-pt": 0x0816, // Portuguese (Portugal)
	"pt-br": 0x0416, // Portuguese (Brazil)
	"zh-tw": 0x0404, // Chinese (Traditional)
	"zh-cn": 0x0804, // Chinese (Simplified)
	"es-mx": 0x080A, // Spanish (Mexico)
	"fr-ca": 0x0C0C, // French (Canada)
	"de-at": 0x0C07, // German (Austria)
	"de-ch": 0x0807, // German (Switzerland)
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
// Tries full regional tag first (e.g. "en-gb"), then falls back to primary subtag (e.g. "en").
// Returns defaultLanguageCode (English US) for unknown or empty strings.
func LanguageCode(lang string) uint32 {
	lang = strings.TrimSpace(lang)
	if lang == "" {
		return defaultLanguageCode
	}
	// Normalize: lowercase and replace _ with -
	lang = strings.ToLower(lang)
	lang = strings.ReplaceAll(lang, "_", "-")
	// Try full tag first (e.g. "en-gb")
	if code, ok := languageCodeMap[lang]; ok {
		return code
	}
	// Fallback to primary subtag (e.g. "en")
	if i := strings.IndexByte(lang, '-'); i >= 0 {
		if code, ok := languageCodeMap[lang[:i]]; ok {
			return code
		}
	}
	return defaultLanguageCode
}

// MOBIHeaderConfig holds the configurable parameters for creating a MOBIHeader.
type MOBIHeaderConfig struct {
	Compression          uint16
	TextLength           uint32
	TextRecordCount      uint16
	Language             string
	UniqueID             *uint32 // nil means random generation
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
// If cfg.UniqueID is nil, a random UniqueID is generated using crypto/rand.
func NewMOBIHeader(cfg MOBIHeaderConfig) (*MOBIHeader, error) {
	if cfg.Compression != CompressionNone && cfg.Compression != CompressionPalmDoc {
		return nil, fmt.Errorf("unsupported compression type: %d", cfg.Compression)
	}
	if cfg.LastContentRecord < cfg.FirstContentRecord {
		return nil, fmt.Errorf("invalid content record range: first=%d > last=%d", cfg.FirstContentRecord, cfg.LastContentRecord)
	}

	var uid uint32
	if cfg.UniqueID != nil {
		uid = *cfg.UniqueID
	} else {
		var err error
		uid, err = generateUniqueID()
		if err != nil {
			return nil, fmt.Errorf("failed to generate unique ID: %w", err)
		}
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

// MOBIHeaderBytes serializes the 248-byte MOBI header.
// fullNameOffset and fullNameLength specify the Full Name position relative to the MOBI header start.
// exthFlags specifies the EXTH flags (use EXTHFlagPresent when EXTH data is present).
func (h *MOBIHeader) MOBIHeaderBytes(fullNameOffset, fullNameLength, exthFlags uint32) ([]byte, error) {
	buf := &bytes.Buffer{}

	writeU32 := func(v uint32) error {
		return binary.Write(buf, binary.BigEndian, v)
	}
	writeU16 := func(v uint16) error {
		return binary.Write(buf, binary.BigEndian, v)
	}

	// Offset 0: "MOBI" identifier
	if _, err := buf.Write([]byte("MOBI")); err != nil {
		return nil, fmt.Errorf("failed to write MOBI identifier: %w", err)
	}

	// Offset 4: header length
	if err := writeU32(MOBIHeaderSize); err != nil {
		return nil, fmt.Errorf("failed to write header length: %w", err)
	}

	// Offset 8: MOBI type
	if err := writeU32(MOBITypeKF8); err != nil {
		return nil, fmt.Errorf("failed to write MOBI type: %w", err)
	}

	// Offset 12: text encoding
	if err := writeU32(EncodingUTF8); err != nil {
		return nil, fmt.Errorf("failed to write text encoding: %w", err)
	}

	// Offset 16: unique ID
	if err := writeU32(h.UniqueID); err != nil {
		return nil, fmt.Errorf("failed to write unique ID: %w", err)
	}

	// Offset 20: file version
	if err := writeU32(FileVersionKF8); err != nil {
		return nil, fmt.Errorf("failed to write file version: %w", err)
	}

	// Offsets 24-63: unused index fields (all 0xFFFFFFFF), 10 fields
	for i := 0; i < 10; i++ {
		if err := writeU32(0xFFFFFFFF); err != nil {
			return nil, fmt.Errorf("failed to write unused index field: %w", err)
		}
	}

	// Offset 64: first non-book index (0xFFFFFFFF)
	if err := writeU32(0xFFFFFFFF); err != nil {
		return nil, fmt.Errorf("failed to write first non-book index: %w", err)
	}

	// Offset 68: Full Name Offset
	if err := writeU32(fullNameOffset); err != nil {
		return nil, fmt.Errorf("failed to write full name offset: %w", err)
	}

	// Offset 72: Full Name Length
	if err := writeU32(fullNameLength); err != nil {
		return nil, fmt.Errorf("failed to write full name length: %w", err)
	}

	// Offset 76: language code
	if err := writeU32(h.LanguageCode); err != nil {
		return nil, fmt.Errorf("failed to write language code: %w", err)
	}

	// Offset 80: first image index
	if err := writeU32(h.FirstImageIndex); err != nil {
		return nil, fmt.Errorf("failed to write first image index: %w", err)
	}

	// Offset 84: first HUFF index (0xFFFFFFFF)
	if err := writeU32(0xFFFFFFFF); err != nil {
		return nil, fmt.Errorf("failed to write HUFF first index: %w", err)
	}

	// Offset 88: HUFF record count (0)
	if err := writeU32(0); err != nil {
		return nil, fmt.Errorf("failed to write HUFF record count: %w", err)
	}

	// Offset 92: HUFF table index (0xFFFFFFFF)
	if err := writeU32(0xFFFFFFFF); err != nil {
		return nil, fmt.Errorf("failed to write HUFF table index: %w", err)
	}

	// Offset 96: HUFF table record count (0)
	if err := writeU32(0); err != nil {
		return nil, fmt.Errorf("failed to write HUFF table count: %w", err)
	}

	// Offset 100: EXTH flags
	if err := writeU32(exthFlags); err != nil {
		return nil, fmt.Errorf("failed to write EXTH flags: %w", err)
	}

	// Offsets 104-135: unused (32 bytes of 0x00)
	if _, err := buf.Write(make([]byte, 32)); err != nil {
		return nil, fmt.Errorf("failed to write unused block: %w", err)
	}

	// Offset 136: DRM offset (0xFFFFFFFF)
	if err := writeU32(0xFFFFFFFF); err != nil {
		return nil, fmt.Errorf("failed to write DRM offset: %w", err)
	}

	// Offset 140: DRM count (0)
	if err := writeU32(0); err != nil {
		return nil, fmt.Errorf("failed to write DRM count: %w", err)
	}

	// Offset 144: DRM size (0)
	if err := writeU32(0); err != nil {
		return nil, fmt.Errorf("failed to write DRM size: %w", err)
	}

	// Offset 148: DRM flags (0)
	if err := writeU32(0); err != nil {
		return nil, fmt.Errorf("failed to write DRM flags: %w", err)
	}

	// Offsets 152-159: unused (8 bytes of 0x00)
	if _, err := buf.Write(make([]byte, 8)); err != nil {
		return nil, fmt.Errorf("failed to write unused block: %w", err)
	}

	// Offset 160: first content record
	if err := writeU16(h.FirstContentRecord); err != nil {
		return nil, fmt.Errorf("failed to write first content record: %w", err)
	}

	// Offset 162: last content record
	if err := writeU16(h.LastContentRecord); err != nil {
		return nil, fmt.Errorf("failed to write last content record: %w", err)
	}

	// Offset 164: unused (0x00000001)
	if err := writeU32(1); err != nil {
		return nil, fmt.Errorf("failed to write unused field: %w", err)
	}

	// Offset 168: FCIS record number
	if err := writeU32(h.FCISRecordNumber); err != nil {
		return nil, fmt.Errorf("failed to write FCIS record number: %w", err)
	}

	// Offset 172: FCIS record count (1)
	if err := writeU32(1); err != nil {
		return nil, fmt.Errorf("failed to write FCIS record count: %w", err)
	}

	// Offset 176: FLIS record number
	if err := writeU32(h.FLISRecordNumber); err != nil {
		return nil, fmt.Errorf("failed to write FLIS record number: %w", err)
	}

	// Offset 180: FLIS record count (1)
	if err := writeU32(1); err != nil {
		return nil, fmt.Errorf("failed to write FLIS record count: %w", err)
	}

	// Offsets 184-191: unused (8 bytes of 0x00)
	if _, err := buf.Write(make([]byte, 8)); err != nil {
		return nil, fmt.Errorf("failed to write unused block: %w", err)
	}

	// Offset 192: unused (0xFFFFFFFF)
	if err := writeU32(0xFFFFFFFF); err != nil {
		return nil, fmt.Errorf("failed to write unused field: %w", err)
	}

	// Offset 196: unused (0x00000000)
	if err := writeU32(0); err != nil {
		return nil, fmt.Errorf("failed to write unused field: %w", err)
	}

	// Offset 200: unused (0xFFFFFFFF)
	if err := writeU32(0xFFFFFFFF); err != nil {
		return nil, fmt.Errorf("failed to write unused field: %w", err)
	}

	// Offset 204: unused (0xFFFFFFFF)
	if err := writeU32(0xFFFFFFFF); err != nil {
		return nil, fmt.Errorf("failed to write unused field: %w", err)
	}

	// Offset 208: extra record data flags
	if err := writeU32(h.ExtraRecordDataFlags); err != nil {
		return nil, fmt.Errorf("failed to write extra record data flags: %w", err)
	}

	// Offset 212: INDX record offset (0xFFFFFFFF)
	if err := writeU32(0xFFFFFFFF); err != nil {
		return nil, fmt.Errorf("failed to write INDX record offset: %w", err)
	}

	// KF8 additional fields (offsets 216-247)

	// Offsets 216-232: unused (5 * 0xFFFFFFFF)
	for i := 0; i < 5; i++ {
		if err := writeU32(0xFFFFFFFF); err != nil {
			return nil, fmt.Errorf("failed to write KF8 unused field: %w", err)
		}
	}

	// Offset 236: FDST flow count
	if err := writeU32(h.FDSTFlowCount); err != nil {
		return nil, fmt.Errorf("failed to write FDST flow count: %w", err)
	}

	// Offset 240: FDST offset
	if err := writeU32(h.FDSTOffset); err != nil {
		return nil, fmt.Errorf("failed to write FDST offset: %w", err)
	}

	// Offset 244: unused (0)
	if err := writeU32(0); err != nil {
		return nil, fmt.Errorf("failed to write unused field: %w", err)
	}

	return buf.Bytes(), nil
}

// Record0Bytes assembles the complete Record 0: PalmDOC header + MOBI header + EXTH data + Full Name + padding.
// exthData may be nil if no EXTH records are present.
// When exthData is provided, it must have a valid EXTH header (magic "EXTH", correct length) and be 4-byte aligned.
// fullName is the book title encoded as UTF-8.
func (h *MOBIHeader) Record0Bytes(exthData []byte, fullName string) ([]byte, error) {
	if err := validateEXTH(exthData); err != nil {
		return nil, err
	}

	palmDoc, err := h.PalmDOCHeaderBytes()
	if err != nil {
		return nil, fmt.Errorf("failed to build PalmDOC header: %w", err)
	}

	// Validate sizes before uint32 conversion
	fullNameBytes := []byte(fullName)
	if len(exthData) > math.MaxUint32-MOBIHeaderSize {
		return nil, fmt.Errorf("EXTH data too large: %d bytes", len(exthData))
	}
	if len(fullNameBytes) > math.MaxUint32 {
		return nil, fmt.Errorf("full name too large: %d bytes", len(fullNameBytes))
	}

	// Calculate Full Name position relative to MOBI header start
	fullNameOffset := uint32(MOBIHeaderSize + len(exthData))
	fullNameLength := uint32(len(fullNameBytes))

	// Determine EXTH flags
	var exthFlags uint32
	if len(exthData) > 0 {
		exthFlags = EXTHFlagPresent
	}

	mobiHeader, err := h.MOBIHeaderBytes(fullNameOffset, fullNameLength, exthFlags)
	if err != nil {
		return nil, fmt.Errorf("failed to build MOBI header: %w", err)
	}

	buf := &bytes.Buffer{}

	// PalmDOC header (16 bytes)
	if _, err := buf.Write(palmDoc); err != nil {
		return nil, fmt.Errorf("failed to write PalmDOC header: %w", err)
	}

	// MOBI header (248 bytes)
	if _, err := buf.Write(mobiHeader); err != nil {
		return nil, fmt.Errorf("failed to write MOBI header: %w", err)
	}

	// EXTH data (variable, may be empty)
	if len(exthData) > 0 {
		if _, err := buf.Write(exthData); err != nil {
			return nil, fmt.Errorf("failed to write EXTH data: %w", err)
		}
	}

	// Full Name (UTF-8)
	if _, err := buf.Write(fullNameBytes); err != nil {
		return nil, fmt.Errorf("failed to write full name: %w", err)
	}

	// Pad to 4-byte boundary
	padLen := (4 - (buf.Len() % 4)) % 4
	if padLen > 0 {
		if _, err := buf.Write(make([]byte, padLen)); err != nil {
			return nil, fmt.Errorf("failed to write padding: %w", err)
		}
	}

	return buf.Bytes(), nil
}

// validateEXTH validates EXTH data integrity.
// Returns nil if exthData is nil or empty (no EXTH present).
func validateEXTH(exthData []byte) error {
	if len(exthData) == 0 {
		return nil
	}
	if len(exthData) < 12 {
		return fmt.Errorf("EXTH data too short: got %d bytes, need at least 12", len(exthData))
	}
	if string(exthData[0:4]) != "EXTH" {
		return fmt.Errorf("invalid EXTH magic: got %q, want %q", string(exthData[0:4]), "EXTH")
	}
	exthLen := binary.BigEndian.Uint32(exthData[4:8])
	if int(exthLen) != len(exthData) {
		return fmt.Errorf("EXTH length mismatch: header says %d, actual %d", exthLen, len(exthData))
	}
	if len(exthData)%4 != 0 {
		return fmt.Errorf("EXTH data must be 4-byte aligned: got %d bytes", len(exthData))
	}
	return nil
}

// generateUniqueID generates a random uint32 using crypto/rand.
func generateUniqueID() (uint32, error) {
	var b [4]byte
	if _, err := rand.Read(b[:]); err != nil {
		return 0, fmt.Errorf("failed to read random bytes: %w", err)
	}
	return binary.BigEndian.Uint32(b[:]), nil
}
