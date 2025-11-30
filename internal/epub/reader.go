package epub

import (
	"archive/zip"
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"strings"
)

// EPUBReader provides access to EPUB file contents
type EPUBReader struct {
	zipReader *zip.ReadCloser
	files     map[string]*zip.File
	opfPath   string
}

// container.xml structure
type container struct {
	Rootfiles struct {
		Rootfile []struct {
			FullPath  string `xml:"full-path,attr"`
			MediaType string `xml:"media-type,attr"`
		} `xml:"rootfile"`
	} `xml:"rootfiles"`
}

var (
	ErrInvalidMimetype    = errors.New("invalid mimetype: must be 'application/epub+zip'")
	ErrMimetypeCompressed = errors.New("mimetype must not be compressed")
	ErrMimetypeNotFound   = errors.New("mimetype file not found")
	ErrContainerNotFound  = errors.New("META-INF/container.xml not found")
	ErrOPFPathNotFound    = errors.New("OPF path not found in container.xml")
)

// Open opens an EPUB file and validates its structure
func Open(path string) (*EPUBReader, error) {
	zr, err := zip.OpenReader(path)
	if err != nil {
		return nil, fmt.Errorf("failed to open EPUB: %w", err)
	}

	reader := &EPUBReader{
		zipReader: zr,
		files:     make(map[string]*zip.File),
	}

	// Build file map with normalized paths
	for _, f := range zr.File {
		name := normalizePath(f.Name)
		reader.files[name] = f
	}

	// Validate mimetype
	if err := reader.validateMimetype(); err != nil {
		zr.Close()
		return nil, err
	}

	// Parse container.xml to get OPF path
	if err := reader.parseContainer(); err != nil {
		zr.Close()
		return nil, err
	}

	return reader, nil
}

// Close closes the EPUB reader
func (r *EPUBReader) Close() error {
	return r.zipReader.Close()
}

// OPFPath returns the path to the OPF file
func (r *EPUBReader) OPFPath() string {
	return r.opfPath
}

// Files returns a map of all files in the EPUB
func (r *EPUBReader) Files() map[string]*zip.File {
	return r.files
}

// ReadFile reads the contents of a file from the EPUB
func (r *EPUBReader) ReadFile(path string) ([]byte, error) {
	path = normalizePath(path)
	f, ok := r.files[path]
	if !ok {
		return nil, fmt.Errorf("file not found: %s", path)
	}

	rc, err := f.Open()
	if err != nil {
		return nil, fmt.Errorf("failed to open file %s: %w", path, err)
	}
	defer rc.Close()

	return io.ReadAll(rc)
}

// validateMimetype checks that the mimetype file exists and is valid
func (r *EPUBReader) validateMimetype() error {
	f, ok := r.files["mimetype"]
	if !ok {
		return ErrMimetypeNotFound
	}

	// Check that mimetype is not compressed
	if f.Method != zip.Store {
		return ErrMimetypeCompressed
	}

	// Read and validate content
	content, err := r.ReadFile("mimetype")
	if err != nil {
		return fmt.Errorf("failed to read mimetype: %w", err)
	}

	if string(content) != "application/epub+zip" {
		return ErrInvalidMimetype
	}

	return nil
}

// parseContainer parses container.xml to extract OPF path
func (r *EPUBReader) parseContainer() error {
	content, err := r.ReadFile("META-INF/container.xml")
	if err != nil {
		return ErrContainerNotFound
	}

	var c container
	if err := xml.Unmarshal(content, &c); err != nil {
		return fmt.Errorf("failed to parse container.xml: %w", err)
	}

	// Find the OPF file path
	for _, rf := range c.Rootfiles.Rootfile {
		if rf.MediaType == "application/oebps-package+xml" || rf.MediaType == "" {
			r.opfPath = normalizePath(rf.FullPath)
			return nil
		}
	}

	// If no media-type match, use the first one
	if len(c.Rootfiles.Rootfile) > 0 {
		r.opfPath = normalizePath(c.Rootfiles.Rootfile[0].FullPath)
		return nil
	}

	return ErrOPFPathNotFound
}

// normalizePath normalizes file paths (removes ./ prefix)
func normalizePath(path string) string {
	path = strings.TrimPrefix(path, "./")
	return path
}
