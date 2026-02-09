package converter

import (
	"testing"

	"github.com/yuanying/epub2azw3/internal/epub"
	"github.com/yuanying/epub2azw3/internal/mobi"
)

type mockFileReader struct {
	files map[string][]byte
}

func (r mockFileReader) ReadFile(path string) ([]byte, error) {
	data, ok := r.files[path]
	if !ok {
		return nil, epub.ErrFileNotFound
	}
	return data, nil
}

func TestDetectCoverInfo_ByManifestProperty(t *testing.T) {
	opf := &epub.OPF{
		Manifest: map[string]epub.ManifestItem{
			"cover": {
				ID:         "cover",
				Href:       "OEBPS/images/cover.jpg",
				MediaType:  "image/jpeg",
				Properties: []string{"cover-image"},
			},
		},
		ManifestOrder: []string{"cover"},
	}

	info := DetectCoverInfo(opf, nil)
	if info == nil {
		t.Fatal("DetectCoverInfo() returned nil")
	}
	if info.ManifestID != "cover" {
		t.Errorf("ManifestID = %q, want %q", info.ManifestID, "cover")
	}
	if info.DetectionMethod != "manifest-property" {
		t.Errorf("DetectionMethod = %q, want %q", info.DetectionMethod, "manifest-property")
	}
}

func TestDetectCoverInfo_ByMetadataCoverID(t *testing.T) {
	opf := &epub.OPF{
		Metadata: epub.Metadata{CoverID: "c1"},
		Manifest: map[string]epub.ManifestItem{
			"c1": {ID: "c1", Href: "OEBPS/images/front.png", MediaType: "image/png"},
		},
		ManifestOrder: []string{"c1"},
	}

	info := DetectCoverInfo(opf, nil)
	if info == nil {
		t.Fatal("DetectCoverInfo() returned nil")
	}
	if info.Href != "OEBPS/images/front.png" {
		t.Errorf("Href = %q, want %q", info.Href, "OEBPS/images/front.png")
	}
	if info.DetectionMethod != "metadata-cover" {
		t.Errorf("DetectionMethod = %q, want %q", info.DetectionMethod, "metadata-cover")
	}
}

func TestDetectCoverInfo_ByGuideXHTMLFirstImage(t *testing.T) {
	opf := &epub.OPF{
		Manifest: map[string]epub.ManifestItem{
			"cover-page": {ID: "cover-page", Href: "OEBPS/text/cover.xhtml", MediaType: "application/xhtml+xml"},
			"cover-img":  {ID: "cover-img", Href: "OEBPS/images/cover.jpg", MediaType: "image/jpeg"},
		},
		ManifestOrder: []string{"cover-page", "cover-img"},
		Guide: []epub.GuideReference{
			{Type: "cover", Href: "OEBPS/text/cover.xhtml"},
		},
	}

	reader := mockFileReader{
		files: map[string][]byte{
			"OEBPS/text/cover.xhtml": []byte(`<?xml version="1.0" encoding="UTF-8"?>
<html xmlns="http://www.w3.org/1999/xhtml"><body><img src="../images/cover.jpg" /></body></html>`),
		},
	}

	info := DetectCoverInfo(opf, reader)
	if info == nil {
		t.Fatal("DetectCoverInfo() returned nil")
	}
	if info.ManifestID != "cover-img" {
		t.Errorf("ManifestID = %q, want %q", info.ManifestID, "cover-img")
	}
	if info.DetectionMethod != "guide-xhtml-first-img" {
		t.Errorf("DetectionMethod = %q, want %q", info.DetectionMethod, "guide-xhtml-first-img")
	}
}

func TestDetectCoverInfo_ByFilenamePattern(t *testing.T) {
	opf := &epub.OPF{
		Manifest: map[string]epub.ManifestItem{
			"img1": {ID: "img1", Href: "OEBPS/images/Cover.jpeg", MediaType: "image/jpeg"},
		},
		ManifestOrder: []string{"img1"},
	}

	info := DetectCoverInfo(opf, nil)
	if info == nil {
		t.Fatal("DetectCoverInfo() returned nil")
	}
	if info.DetectionMethod != "filename-pattern" {
		t.Errorf("DetectionMethod = %q, want %q", info.DetectionMethod, "filename-pattern")
	}
}

func TestDetectCoverInfo_Priority(t *testing.T) {
	opf := &epub.OPF{
		Metadata: epub.Metadata{CoverID: "meta-cover"},
		Manifest: map[string]epub.ManifestItem{
			"manifest-cover": {
				ID:         "manifest-cover",
				Href:       "OEBPS/images/manifest.jpg",
				MediaType:  "image/jpeg",
				Properties: []string{"cover-image"},
			},
			"meta-cover": {
				ID:        "meta-cover",
				Href:      "OEBPS/images/meta.jpg",
				MediaType: "image/jpeg",
			},
			"filename-cover": {
				ID:        "filename-cover",
				Href:      "OEBPS/images/cover.png",
				MediaType: "image/png",
			},
		},
		ManifestOrder: []string{"manifest-cover", "meta-cover", "filename-cover"},
	}

	info := DetectCoverInfo(opf, nil)
	if info == nil {
		t.Fatal("DetectCoverInfo() returned nil")
	}
	if info.ManifestID != "manifest-cover" {
		t.Errorf("ManifestID = %q, want %q", info.ManifestID, "manifest-cover")
	}
}

func TestDetectCoverInfo_NoCover(t *testing.T) {
	opf := &epub.OPF{
		Manifest: map[string]epub.ManifestItem{
			"ch1": {ID: "ch1", Href: "OEBPS/text/ch1.xhtml", MediaType: "application/xhtml+xml"},
			"img": {ID: "img", Href: "OEBPS/images/photo.jpg", MediaType: "image/jpeg"},
		},
		ManifestOrder: []string{"ch1", "img"},
	}

	info := DetectCoverInfo(opf, nil)
	if info != nil {
		t.Fatalf("DetectCoverInfo() = %+v, want nil", info)
	}
}

func TestComputeCoverOffset(t *testing.T) {
	mapper := mobi.NewImageMapper()
	mapper.AddImage("OEBPS/images/cover.jpg", []byte("cover"), "image/jpeg")
	mapper.AddImage("OEBPS/images/p01.jpg", []byte("page"), "image/jpeg")

	offset, ok := ComputeCoverOffset(&CoverInfo{Href: "OEBPS/images/cover.jpg"}, mapper)
	if !ok {
		t.Fatal("ComputeCoverOffset() ok = false, want true")
	}
	if offset != 0 {
		t.Errorf("offset = %d, want 0", offset)
	}
}

func TestComputeCoverOffset_NotFound(t *testing.T) {
	mapper := mobi.NewImageMapper()
	mapper.AddImage("OEBPS/images/p01.jpg", []byte("page"), "image/jpeg")

	_, ok := ComputeCoverOffset(&CoverInfo{Href: "OEBPS/images/cover.jpg"}, mapper)
	if ok {
		t.Fatal("ComputeCoverOffset() ok = true, want false")
	}
}
