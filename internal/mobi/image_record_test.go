package mobi

import (
	"strings"
	"testing"
)

func TestNewImageMapper(t *testing.T) {
	m := NewImageMapper()
	if m == nil {
		t.Fatal("NewImageMapper returned nil")
	}
	if len(m.Images) != 0 {
		t.Fatalf("expected 0 images, got %d", len(m.Images))
	}
}

func TestImageMapper_AddImage(t *testing.T) {
	m := NewImageMapper()
	m.AddImage("images/cover.jpg", []byte("fake-jpg-data"), "image/jpeg")
	m.AddImage("images/ch01.png", []byte("fake-png-data"), "image/png")

	if len(m.Images) != 2 {
		t.Fatalf("expected 2 images, got %d", len(m.Images))
	}
	if m.Images[0].OriginalPath != "images/cover.jpg" {
		t.Fatalf("expected path 'images/cover.jpg', got %q", m.Images[0].OriginalPath)
	}
	if m.Images[1].MediaType != "image/png" {
		t.Fatalf("expected media type 'image/png', got %q", m.Images[1].MediaType)
	}
}

func TestImageMapper_PathToIndex(t *testing.T) {
	m := NewImageMapper()
	m.AddImage("images/cover.jpg", []byte("data"), "image/jpeg")
	m.AddImage("images/ch01.png", []byte("data"), "image/png")

	idx, ok := m.PathToIndex["images/cover.jpg"]
	if !ok {
		t.Fatal("path 'images/cover.jpg' not found")
	}
	if idx != 0 {
		t.Fatalf("expected index 0, got %d", idx)
	}

	idx, ok = m.PathToIndex["images/ch01.png"]
	if !ok {
		t.Fatal("path 'images/ch01.png' not found")
	}
	if idx != 1 {
		t.Fatalf("expected index 1, got %d", idx)
	}
}

func TestImageMapper_KindleEmbedRef(t *testing.T) {
	m := NewImageMapper()
	m.AddImage("images/cover.jpg", []byte("data"), "image/jpeg")
	m.AddImage("images/ch01.png", []byte("data"), "image/png")

	ref, ok := m.KindleEmbedRef("images/cover.jpg")
	if !ok {
		t.Fatal("expected to find images/cover.jpg")
	}
	if ref != "kindle:embed:0001" {
		t.Fatalf("expected kindle:embed:0001, got %q", ref)
	}

	ref, ok = m.KindleEmbedRef("images/ch01.png")
	if !ok {
		t.Fatal("expected to find images/ch01.png")
	}
	if ref != "kindle:embed:0002" {
		t.Fatalf("expected kindle:embed:0002, got %q", ref)
	}

	_, ok = m.KindleEmbedRef("images/nonexistent.jpg")
	if ok {
		t.Fatal("expected nonexistent image to not be found")
	}
}

func TestImageMapper_KindleEmbedRef_ManyImages(t *testing.T) {
	m := NewImageMapper()
	// Add 256 images
	for i := range 256 {
		path := "images/" + string(rune('a'+i%26)) + string(rune('0'+i/26)) + ".jpg"
		m.AddImage(path, []byte("data"), "image/jpeg")
	}

	// Check the 256th image (index 255, record number 256 = 0x0100)
	lastPath := m.Images[255].OriginalPath
	ref, ok := m.KindleEmbedRef(lastPath)
	if !ok {
		t.Fatal("expected to find last image")
	}
	if ref != "kindle:embed:0100" {
		t.Fatalf("expected kindle:embed:0100, got %q", ref)
	}
}

func TestImageMapper_ImageRecordData(t *testing.T) {
	m := NewImageMapper()
	data1 := []byte("jpeg-data-here")
	data2 := []byte("png-data-here")
	m.AddImage("a.jpg", data1, "image/jpeg")
	m.AddImage("b.png", data2, "image/png")

	records := m.ImageRecordData()
	if len(records) != 2 {
		t.Fatalf("expected 2 records, got %d", len(records))
	}
	if string(records[0]) != "jpeg-data-here" {
		t.Fatalf("record 0 mismatch")
	}
	if string(records[1]) != "png-data-here" {
		t.Fatalf("record 1 mismatch")
	}
}

func TestImageMapper_ImageRecordData_Empty(t *testing.T) {
	m := NewImageMapper()
	records := m.ImageRecordData()
	if records != nil {
		t.Fatalf("expected nil, got %v", records)
	}
}

func TestTransformImageReferences_SingleImage(t *testing.T) {
	m := NewImageMapper()
	m.AddImage("images/cover.jpg", []byte("data"), "image/jpeg")

	html := `<html><body><img src="images/cover.jpg"/></body></html>`
	result := TransformImageReferences(html, m)

	if !strings.Contains(result, `src="kindle:embed:0001"`) {
		t.Fatalf("expected kindle:embed:0001 in result:\n%s", result)
	}
}

func TestTransformImageReferences_MultipleImages(t *testing.T) {
	m := NewImageMapper()
	m.AddImage("images/cover.jpg", []byte("data"), "image/jpeg")
	m.AddImage("images/ch01.png", []byte("data"), "image/png")

	html := `<img src="images/cover.jpg"/><img src="images/ch01.png"/>`
	result := TransformImageReferences(html, m)

	if !strings.Contains(result, `src="kindle:embed:0001"`) {
		t.Fatalf("expected kindle:embed:0001 in result:\n%s", result)
	}
	if !strings.Contains(result, `src="kindle:embed:0002"`) {
		t.Fatalf("expected kindle:embed:0002 in result:\n%s", result)
	}
}

func TestTransformImageReferences_NoImages(t *testing.T) {
	m := NewImageMapper()
	html := `<html><body><p>No images here</p></body></html>`
	result := TransformImageReferences(html, m)
	if result != html {
		t.Fatalf("expected unchanged HTML, got:\n%s", result)
	}
}

func TestTransformImageReferences_UnknownImage(t *testing.T) {
	m := NewImageMapper()
	m.AddImage("images/known.jpg", []byte("data"), "image/jpeg")

	html := `<img src="images/unknown.jpg"/>`
	result := TransformImageReferences(html, m)

	// Unknown image should remain unchanged
	if !strings.Contains(result, `src="images/unknown.jpg"`) {
		t.Fatalf("unknown image should be unchanged, got:\n%s", result)
	}
}

func TestTransformImageReferences_NilMapper(t *testing.T) {
	html := `<img src="images/test.jpg"/>`
	result := TransformImageReferences(html, nil)
	if result != html {
		t.Fatalf("nil mapper should return unchanged HTML")
	}
}

func TestTransformImageReferences_ResolvedPath(t *testing.T) {
	// Simulates the case where img src has been resolved to absolute EPUB path
	// (matching the manifest Href format)
	m := NewImageMapper()
	m.AddImage("OEBPS/images/cover.jpg", []byte("data"), "image/jpeg")

	html := `<img src="OEBPS/images/cover.jpg"/>`
	result := TransformImageReferences(html, m)

	if !strings.Contains(result, `src="kindle:embed:0001"`) {
		t.Fatalf("expected kindle:embed:0001 in result:\n%s", result)
	}
}

func TestImageMapper_DuplicatePath(t *testing.T) {
	m := NewImageMapper()
	m.AddImage("images/cover.jpg", []byte("data1"), "image/jpeg")
	m.AddImage("images/cover.jpg", []byte("data2"), "image/jpeg")

	// Duplicate should not add a second image
	if len(m.Images) != 1 {
		t.Fatalf("expected 1 image (duplicate skipped), got %d", len(m.Images))
	}
}
