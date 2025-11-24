# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

EPUB to AZW3 converter - a standalone Go implementation for converting EPUB ebooks to Amazon Kindle compatible AZW3 (KF8) format without external dependencies like Calibre.

## Build Commands

```bash
# Build the CLI
go build -o epub2azw3 ./cmd/epub2azw3

# Run tests
go test ./...

# Run a specific test
go test -v -run TestFunctionName ./internal/epub

# Run tests with coverage
go test -coverprofile=coverage.out ./...
go tool cover -html=coverage.out
```

## Architecture

The project follows a pipeline architecture for converting EPUB to AZW3:

```
EPUB Input → Parse → Validate → Transform HTML/CSS → Optimize Images → Generate TOC → Build AZW3 → Output
```

### Package Structure

- **cmd/epub2azw3**: CLI entry point using Cobra
- **internal/epub**: EPUB parsing (ZIP extraction, OPF/NCX/container.xml parsing, content loading)
- **internal/converter**: Transformation logic (HTML→Kindle-compatible, CSS optimization, image processing, TOC generation)
- **internal/mobi**: AZW3/MOBI file generation (PDB structure, MOBI header, EXTH records, PalmDoc compression, record assembly)
- **internal/util**: Utilities (path resolution, encoding, Palm epoch time conversion)
- **pkg/epub2azw3**: Public API for library usage

### Key Technical Details

- All binary data uses **big-endian** byte order
- Text encoding is **UTF-8** (MOBI encoding type 65001)
- PalmDoc compression is the recommended compression method
- Image references use `kindle:embed:XXXX` format (4-digit hex record number)
- TOC entries use `filepos` attributes pointing to byte offsets in uncompressed HTML

### Implementation Reference

See `spec.md` for complete technical specifications including:
- EPUB format parsing details
- AZW3/MOBI binary format structure
- PalmDoc compression algorithm
- EXTH metadata mapping
- Implementation phases and priorities
