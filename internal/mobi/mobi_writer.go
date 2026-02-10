package mobi

import (
	"bytes"
	"fmt"
	"io"
	"time"

	"github.com/yuanying/epub2azw3/internal/epub"
)

// MOBIWriterConfig holds configuration for creating a MOBIWriter (dual format MOBI7+KF8).
type MOBIWriterConfig struct {
	Title        string
	HTML         []byte
	Metadata     *epub.Metadata
	ImageRecords [][]byte
	CoverOffset  *uint32
	NCXRecord    []byte
	Compression  uint16
	CreationTime time.Time
	UniqueID     *uint32
}

// MOBIWriter assembles and writes a complete dual-format MOBI file (MOBI7+KF8).
type MOBIWriter struct {
	cfg MOBIWriterConfig
}

// NewMOBIWriter creates a new MOBIWriter from the given configuration.
func NewMOBIWriter(cfg MOBIWriterConfig) (*MOBIWriter, error) {
	if len(cfg.HTML) == 0 {
		return nil, fmt.Errorf("HTML content is required")
	}

	if cfg.Compression == 0 {
		cfg.Compression = CompressionNone
	}
	if cfg.Compression != CompressionNone && cfg.Compression != CompressionPalmDoc {
		return nil, fmt.Errorf("unsupported compression type: %d", cfg.Compression)
	}

	return &MOBIWriter{cfg: cfg}, nil
}

// WriteTo writes the complete dual-format MOBI file to the given writer.
func (w *MOBIWriter) WriteTo(out io.Writer) (int64, error) {
	cfg := w.cfg

	// --- Ensure shared UniqueID for both MOBI7 and KF8 sections ---
	uid := cfg.UniqueID
	if uid == nil {
		generated, err := generateUniqueID()
		if err != nil {
			return 0, fmt.Errorf("failed to generate unique ID: %w", err)
		}
		uid = &generated
	}

	// --- Select compressor ---
	var compressor Compressor
	if cfg.Compression == CompressionPalmDoc {
		compressor = &PalmDocCompressor{}
	} else {
		compressor = &NoCompression{}
	}

	// --- Split text records (same HTML, reuse for both MOBI7 and KF8) ---
	textRecords, err := SplitTextRecords(cfg.HTML, compressor)
	if err != nil {
		return 0, fmt.Errorf("failed to split text records: %w", err)
	}

	textLen := TextLength(cfg.HTML)
	textRecCount := len(textRecords)

	// --- Record index calculation ---
	// MOBI7 section
	mobi7FirstContent := uint16(1)
	mobi7LastContent := uint16(textRecCount)
	nextIndex := 1 + textRecCount // Record 0 + MOBI7 text records

	// Image records (shared between MOBI7 and KF8)
	var firstImageIndex uint32 = 0xFFFFFFFF
	if len(cfg.ImageRecords) > 0 {
		firstImageIndex = uint32(nextIndex)
	}
	nextIndex += len(cfg.ImageRecords)

	// MOBI7 FLIS, FCIS
	mobi7FLISIndex := uint32(nextIndex)
	nextIndex++
	mobi7FCISIndex := uint32(nextIndex)
	nextIndex++

	// Boundary EOF marker
	boundaryIndex := nextIndex
	nextIndex++

	// KF8 section
	nextIndex++ // KF8 Record 0

	kf8FirstContent := uint16(nextIndex)
	kf8LastContent := uint16(nextIndex + textRecCount - 1)
	nextIndex += textRecCount

	// NCX record (KF8 only, if present)
	if len(cfg.NCXRecord) > 0 {
		nextIndex++
	}

	// KF8 FDST, FLIS, FCIS, EOF
	nextIndex++ // FDST
	kf8FLISIndex := uint32(nextIndex)
	nextIndex++
	kf8FCISIndex := uint32(nextIndex)
	nextIndex++
	nextIndex++ // EOF

	totalRecordCount := uint32(nextIndex)

	// --- Build EXTH headers ---
	// MOBI7 EXTH: boundary offset = boundary marker record number
	var mobi7EXTH *EXTHHeader
	if cfg.Metadata != nil {
		mobi7EXTH = EXTHFromMetadata(*cfg.Metadata, uint32(boundaryIndex), totalRecordCount)
	} else {
		mobi7EXTH = NewEXTHHeader(uint32(boundaryIndex), totalRecordCount)
	}
	if cfg.CoverOffset != nil {
		mobi7EXTH.AddUint32Record(131, *cfg.CoverOffset)
	}

	mobi7EXTHData, err := mobi7EXTH.Bytes()
	if err != nil {
		return 0, fmt.Errorf("failed to serialize MOBI7 EXTH: %w", err)
	}

	// KF8 EXTH: boundary offset = 0
	var kf8EXTH *EXTHHeader
	if cfg.Metadata != nil {
		kf8EXTH = EXTHFromMetadata(*cfg.Metadata, 0, totalRecordCount)
	} else {
		kf8EXTH = NewEXTHHeader(0, totalRecordCount)
	}
	if cfg.CoverOffset != nil {
		kf8EXTH.AddUint32Record(131, *cfg.CoverOffset)
	}

	kf8EXTHData, err := kf8EXTH.Bytes()
	if err != nil {
		return 0, fmt.Errorf("failed to serialize KF8 EXTH: %w", err)
	}

	// --- Build fixed records ---
	fdst := NewFDSTSingleFlow(textLen)
	fdstData, err := fdst.Bytes()
	if err != nil {
		return 0, fmt.Errorf("failed to serialize FDST: %w", err)
	}

	mobi7FLISData := FLISRecord()
	mobi7FCISData, err := FCISRecord(textLen)
	if err != nil {
		return 0, fmt.Errorf("failed to serialize MOBI7 FCIS: %w", err)
	}

	kf8FLISData := FLISRecord()
	kf8FCISData, err := FCISRecord(textLen)
	if err != nil {
		return 0, fmt.Errorf("failed to serialize KF8 FCIS: %w", err)
	}

	boundaryEOFData := EOFRecord()
	kf8EOFData := EOFRecord()

	// --- Build Record 0 headers ---
	language := ""
	if cfg.Metadata != nil {
		language = cfg.Metadata.Language
	}

	// MOBI7 Record 0
	mobi7Cfg := MOBIHeaderConfig{
		Compression:          cfg.Compression,
		TextLength:           textLen,
		TextRecordCount:      uint16(textRecCount),
		Language:             language,
		UniqueID:             uid,
		FirstImageIndex:      firstImageIndex,
		FirstContentRecord:   mobi7FirstContent,
		LastContentRecord:    mobi7LastContent,
		FCISRecordNumber:     mobi7FCISIndex,
		FLISRecordNumber:     mobi7FLISIndex,
		ExtraRecordDataFlags: 0,
		MOBIType:             MOBITypeMOBI7,
		FileVersion:          FileVersionMOBI7,
	}

	mobi7Header, err := NewMOBIHeader(mobi7Cfg)
	if err != nil {
		return 0, fmt.Errorf("failed to create MOBI7 header: %w", err)
	}

	mobi7Record0, err := mobi7Header.Record0Bytes(mobi7EXTHData, cfg.Title)
	if err != nil {
		return 0, fmt.Errorf("failed to build MOBI7 Record 0: %w", err)
	}

	// KF8 Record 0
	kf8Cfg := MOBIHeaderConfig{
		Compression:          cfg.Compression,
		TextLength:           textLen,
		TextRecordCount:      uint16(textRecCount),
		Language:             language,
		UniqueID:             uid,
		FirstImageIndex:      firstImageIndex,
		FirstContentRecord:   kf8FirstContent,
		LastContentRecord:    kf8LastContent,
		FCISRecordNumber:     kf8FCISIndex,
		FLISRecordNumber:     kf8FLISIndex,
		ExtraRecordDataFlags: 0,
		FDSTFlowCount:        fdst.FlowCount(),
		FDSTOffset:           0xFFFFFFFF,
		MOBIType:             MOBITypeKF8,
		FileVersion:          FileVersionKF8,
	}

	kf8Header, err := NewMOBIHeader(kf8Cfg)
	if err != nil {
		return 0, fmt.Errorf("failed to create KF8 header: %w", err)
	}

	kf8Record0, err := kf8Header.Record0Bytes(kf8EXTHData, cfg.Title)
	if err != nil {
		return 0, fmt.Errorf("failed to build KF8 Record 0: %w", err)
	}

	// --- Build record sizes slice ---
	recordSizes := make([]int, 0, int(totalRecordCount))

	// MOBI7 section
	recordSizes = append(recordSizes, len(mobi7Record0))
	for _, tr := range textRecords {
		recordSizes = append(recordSizes, len(tr))
	}
	for _, ir := range cfg.ImageRecords {
		recordSizes = append(recordSizes, len(ir))
	}
	recordSizes = append(recordSizes, len(mobi7FLISData))
	recordSizes = append(recordSizes, len(mobi7FCISData))
	recordSizes = append(recordSizes, len(boundaryEOFData))

	// KF8 section
	recordSizes = append(recordSizes, len(kf8Record0))
	for _, tr := range textRecords {
		recordSizes = append(recordSizes, len(tr))
	}
	if len(cfg.NCXRecord) > 0 {
		recordSizes = append(recordSizes, len(cfg.NCXRecord))
	}
	recordSizes = append(recordSizes, len(fdstData))
	recordSizes = append(recordSizes, len(kf8FLISData))
	recordSizes = append(recordSizes, len(kf8FCISData))
	recordSizes = append(recordSizes, len(kf8EOFData))

	// --- Build PDB ---
	creation := cfg.CreationTime
	if creation.IsZero() {
		creation = time.Now().UTC()
	}

	pdb, err := NewPDB(cfg.Title, recordSizes, creation, creation)
	if err != nil {
		return 0, fmt.Errorf("failed to create PDB: %w", err)
	}

	// --- Write phase ---
	var written int64

	writeAll := func(data []byte, label string) error {
		n, err := io.Copy(out, bytes.NewReader(data))
		written += n
		if err != nil {
			return fmt.Errorf("failed to write %s: %w", label, err)
		}
		return nil
	}

	headerBytes, err := pdb.HeaderBytes()
	if err != nil {
		return written, fmt.Errorf("failed to serialize PDB header: %w", err)
	}
	if err := writeAll(headerBytes, "PDB header"); err != nil {
		return written, err
	}

	recordListBytes, err := pdb.RecordListBytes()
	if err != nil {
		return written, fmt.Errorf("failed to serialize record list: %w", err)
	}
	if err := writeAll(recordListBytes, "record list"); err != nil {
		return written, err
	}

	// MOBI7 section
	if err := writeAll(mobi7Record0, "MOBI7 Record 0"); err != nil {
		return written, err
	}
	for i, tr := range textRecords {
		if err := writeAll(tr, fmt.Sprintf("MOBI7 text record %d", i)); err != nil {
			return written, err
		}
	}
	for i, ir := range cfg.ImageRecords {
		if err := writeAll(ir, fmt.Sprintf("image record %d", i)); err != nil {
			return written, err
		}
	}
	if err := writeAll(mobi7FLISData, "MOBI7 FLIS"); err != nil {
		return written, err
	}
	if err := writeAll(mobi7FCISData, "MOBI7 FCIS"); err != nil {
		return written, err
	}
	if err := writeAll(boundaryEOFData, "boundary EOF"); err != nil {
		return written, err
	}

	// KF8 section
	if err := writeAll(kf8Record0, "KF8 Record 0"); err != nil {
		return written, err
	}
	for i, tr := range textRecords {
		if err := writeAll(tr, fmt.Sprintf("KF8 text record %d", i)); err != nil {
			return written, err
		}
	}
	if len(cfg.NCXRecord) > 0 {
		if err := writeAll(cfg.NCXRecord, "NCX record"); err != nil {
			return written, err
		}
	}
	if err := writeAll(fdstData, "FDST"); err != nil {
		return written, err
	}
	if err := writeAll(kf8FLISData, "KF8 FLIS"); err != nil {
		return written, err
	}
	if err := writeAll(kf8FCISData, "KF8 FCIS"); err != nil {
		return written, err
	}
	if err := writeAll(kf8EOFData, "KF8 EOF"); err != nil {
		return written, err
	}

	return written, nil
}
