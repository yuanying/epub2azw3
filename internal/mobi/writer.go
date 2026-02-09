package mobi

import (
	"bytes"
	"fmt"
	"io"
	"time"

	"github.com/yuanying/epub2azw3/internal/epub"
)

// AZW3WriterConfig holds configuration for creating an AZW3Writer.
type AZW3WriterConfig struct {
	Title           string
	HTML            []byte
	Metadata        *epub.Metadata
	ImageRecords    [][]byte
	Compression     uint16
	CreationTime    time.Time
	UniqueID        *uint32
	CoverImageIndex *int // 0-based index into ImageRecords for cover; nil means no cover
}

// AZW3Writer assembles and writes a complete AZW3 file.
type AZW3Writer struct {
	cfg AZW3WriterConfig
}

// NewAZW3Writer creates a new AZW3Writer from the given configuration.
func NewAZW3Writer(cfg AZW3WriterConfig) (*AZW3Writer, error) {
	if len(cfg.HTML) == 0 {
		return nil, fmt.Errorf("HTML content is required")
	}

	// Default compression to None
	if cfg.Compression == 0 {
		cfg.Compression = CompressionNone
	}
	if cfg.Compression != CompressionNone && cfg.Compression != CompressionPalmDoc {
		return nil, fmt.Errorf("unsupported compression type: %d", cfg.Compression)
	}

	return &AZW3Writer{cfg: cfg}, nil
}

// WriteTo writes the complete AZW3 file to the given writer.
func (w *AZW3Writer) WriteTo(out io.Writer) (int64, error) {
	cfg := w.cfg

	// --- Pass 1: Determine record numbers ---

	// Select compressor based on compression type
	var compressor Compressor
	if cfg.Compression == CompressionPalmDoc {
		compressor = &PalmDocCompressor{}
	} else {
		compressor = &NoCompression{}
	}

	// Split text into records
	textRecords, err := SplitTextRecords(cfg.HTML, compressor)
	if err != nil {
		return 0, fmt.Errorf("failed to split text records: %w", err)
	}

	textLen := TextLength(cfg.HTML)
	textRecCount := len(textRecords)

	// Record index calculation
	firstContentRecord := uint16(1)
	lastContentRecord := uint16(textRecCount)
	nextIndex := 1 + textRecCount // after Record 0 and text records

	var firstImageIndex uint32 = 0xFFFFFFFF
	if len(cfg.ImageRecords) > 0 {
		firstImageIndex = uint32(nextIndex)
	}
	nextIndex += len(cfg.ImageRecords)

	fdstIndex := nextIndex
	_ = fdstIndex // used for record placement
	nextIndex++

	flisIndex := uint32(nextIndex)
	nextIndex++

	fcisIndex := uint32(nextIndex)
	nextIndex++

	// EOF
	nextIndex++

	totalRecordCount := uint32(nextIndex)

	// --- Build EXTH ---
	var exth *EXTHHeader
	if cfg.Metadata != nil {
		exth = EXTHFromMetadata(*cfg.Metadata, 0, totalRecordCount)
	} else {
		exth = NewEXTHHeader(0, totalRecordCount)
	}

	if cfg.CoverImageIndex != nil {
		exth.AddUint32Record(131, uint32(*cfg.CoverImageIndex))
	}

	exthData, err := exth.Bytes()
	if err != nil {
		return 0, fmt.Errorf("failed to serialize EXTH: %w", err)
	}

	// --- Build fixed records ---
	fdst := NewFDSTSingleFlow(textLen)
	fdstData, err := fdst.Bytes()
	if err != nil {
		return 0, fmt.Errorf("failed to serialize FDST: %w", err)
	}

	flisData := FLISRecord()

	fcisData, err := FCISRecord(textLen)
	if err != nil {
		return 0, fmt.Errorf("failed to serialize FCIS: %w", err)
	}

	eofData := EOFRecord()

	// --- Pass 2: Build Record 0 ---
	language := ""
	if cfg.Metadata != nil {
		language = cfg.Metadata.Language
	}

	mobiCfg := MOBIHeaderConfig{
		Compression:          cfg.Compression,
		TextLength:           textLen,
		TextRecordCount:      uint16(textRecCount),
		Language:             language,
		UniqueID:             cfg.UniqueID,
		FirstImageIndex:      firstImageIndex,
		FirstContentRecord:   firstContentRecord,
		LastContentRecord:    lastContentRecord,
		FCISRecordNumber:     fcisIndex,
		FLISRecordNumber:     flisIndex,
		ExtraRecordDataFlags: 0,
		FDSTFlowCount:        fdst.FlowCount(),
		FDSTOffset:           0xFFFFFFFF, // FDST is a standalone record
	}

	mobiHeader, err := NewMOBIHeader(mobiCfg)
	if err != nil {
		return 0, fmt.Errorf("failed to create MOBI header: %w", err)
	}

	record0, err := mobiHeader.Record0Bytes(exthData, cfg.Title)
	if err != nil {
		return 0, fmt.Errorf("failed to build Record 0: %w", err)
	}

	// --- Build record sizes slice ---
	recordSizes := make([]int, 0, int(totalRecordCount))
	recordSizes = append(recordSizes, len(record0))
	for _, tr := range textRecords {
		recordSizes = append(recordSizes, len(tr))
	}
	for _, ir := range cfg.ImageRecords {
		recordSizes = append(recordSizes, len(ir))
	}
	recordSizes = append(recordSizes, len(fdstData))
	recordSizes = append(recordSizes, len(flisData))
	recordSizes = append(recordSizes, len(fcisData))
	recordSizes = append(recordSizes, len(eofData))

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

	if err := writeAll(record0, "Record 0"); err != nil {
		return written, err
	}

	for i, tr := range textRecords {
		if err := writeAll(tr, fmt.Sprintf("text record %d", i)); err != nil {
			return written, err
		}
	}

	for i, ir := range cfg.ImageRecords {
		if err := writeAll(ir, fmt.Sprintf("image record %d", i)); err != nil {
			return written, err
		}
	}

	if err := writeAll(fdstData, "FDST"); err != nil {
		return written, err
	}
	if err := writeAll(flisData, "FLIS"); err != nil {
		return written, err
	}
	if err := writeAll(fcisData, "FCIS"); err != nil {
		return written, err
	}
	if err := writeAll(eofData, "EOF"); err != nil {
		return written, err
	}

	return written, nil
}
