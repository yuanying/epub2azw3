package converter

import (
	"strings"
	"testing"

	"github.com/yuanying/epub2azw3/internal/epub"
)

// --- GenerateInlineTOC tests ---

func TestGenerateInlineTOC_Basic(t *testing.T) {
	ncx := &epub.NCX{
		DocTitle: "My Book",
		NavPoints: []epub.NavPoint{
			{Label: "Chapter 1", ContentPath: "text/ch01.xhtml"},
			{Label: "Chapter 2", ContentPath: "text/ch02.xhtml"},
		},
	}
	chapterIDs := map[string]string{
		"text/ch01.xhtml": "ch01",
		"text/ch02.xhtml": "ch02",
	}
	gen := NewTOCGenerator(ncx, chapterIDs)
	html := gen.GenerateInlineTOC()

	// Should have toc div
	if !strings.Contains(html, `id="toc"`) {
		t.Error("expected toc div with id")
	}

	// Should have title
	if !strings.Contains(html, "<h1>My Book</h1>") {
		t.Errorf("expected title, got: %s", html)
	}

	// Should have chapter links pointing to chapter IDs
	if !strings.Contains(html, `href="#ch01"`) {
		t.Error("expected link to #ch01")
	}
	if !strings.Contains(html, `href="#ch02"`) {
		t.Error("expected link to #ch02")
	}

	// Should have labels
	if !strings.Contains(html, "Chapter 1") {
		t.Error("expected Chapter 1 label")
	}
	if !strings.Contains(html, "Chapter 2") {
		t.Error("expected Chapter 2 label")
	}
}

func TestGenerateInlineTOC_WithFragment(t *testing.T) {
	ncx := &epub.NCX{
		DocTitle: "Book",
		NavPoints: []epub.NavPoint{
			{Label: "Section 1", ContentPath: "text/ch01.xhtml", Fragment: "sec1"},
		},
	}
	chapterIDs := map[string]string{
		"text/ch01.xhtml": "ch01",
	}
	gen := NewTOCGenerator(ncx, chapterIDs)
	html := gen.GenerateInlineTOC()

	// Fragment should be sanitized and namespaced
	if !strings.Contains(html, `href="#ch01-sec1"`) {
		t.Errorf("expected link to #ch01-sec1, got: %s", html)
	}
}

func TestGenerateInlineTOC_Nested(t *testing.T) {
	ncx := &epub.NCX{
		DocTitle: "Book",
		NavPoints: []epub.NavPoint{
			{
				Label:       "Part 1",
				ContentPath: "text/ch01.xhtml",
				Children: []epub.NavPoint{
					{Label: "Chapter 1.1", ContentPath: "text/ch01.xhtml", Fragment: "s1"},
				},
			},
		},
	}
	chapterIDs := map[string]string{
		"text/ch01.xhtml": "ch01",
	}
	gen := NewTOCGenerator(ncx, chapterIDs)
	html := gen.GenerateInlineTOC()

	// Should have nested structure
	ulCount := strings.Count(html, "<ul>")
	if ulCount < 2 {
		t.Errorf("expected at least 2 <ul> tags for nested structure, got %d", ulCount)
	}
	if !strings.Contains(html, "Part 1") {
		t.Error("expected Part 1")
	}
	if !strings.Contains(html, "Chapter 1.1") {
		t.Error("expected Chapter 1.1")
	}
}

func TestGenerateInlineTOC_Empty(t *testing.T) {
	ncx := &epub.NCX{
		DocTitle:  "Book",
		NavPoints: nil,
	}
	gen := NewTOCGenerator(ncx, nil)
	html := gen.GenerateInlineTOC()

	if html != "" {
		t.Errorf("expected empty string for empty NavPoints, got: %s", html)
	}
}

func TestGenerateInlineTOC_EmptyDocTitle(t *testing.T) {
	ncx := &epub.NCX{
		DocTitle: "",
		NavPoints: []epub.NavPoint{
			{Label: "Chapter 1", ContentPath: "text/ch01.xhtml"},
		},
	}
	chapterIDs := map[string]string{
		"text/ch01.xhtml": "ch01",
	}
	gen := NewTOCGenerator(ncx, chapterIDs)
	html := gen.GenerateInlineTOC()

	// Should use "Table of Contents" as default title
	if !strings.Contains(html, "<h1>Table of Contents</h1>") {
		t.Errorf("expected default title, got: %s", html)
	}
}

// --- InsertInlineTOC tests ---

func TestInsertInlineTOC_Basic(t *testing.T) {
	ncx := &epub.NCX{
		DocTitle: "Book",
		NavPoints: []epub.NavPoint{
			{Label: "Chapter 1", ContentPath: "text/ch01.xhtml"},
		},
	}
	chapterIDs := map[string]string{
		"text/ch01.xhtml": "ch01",
	}
	gen := NewTOCGenerator(ncx, chapterIDs)

	html := `<html><head></head><body><div id="ch01">content</div></body></html>`
	result := gen.InsertInlineTOC(html)

	// TOC should be inserted right after <body>
	bodyIdx := strings.Index(result, "<body>")
	tocIdx := strings.Index(result, `<div id="toc">`)
	if tocIdx < 0 {
		t.Fatal("expected toc div to be inserted")
	}
	if tocIdx != bodyIdx+len("<body>") {
		t.Errorf("expected toc right after <body>, bodyEnd=%d, tocStart=%d", bodyIdx+len("<body>"), tocIdx)
	}

	// Original content should still be present
	if !strings.Contains(result, `<div id="ch01">content</div>`) {
		t.Error("expected original content to remain")
	}
}

func TestInsertInlineTOC_BodyWithAttributes(t *testing.T) {
	ncx := &epub.NCX{
		DocTitle: "Book",
		NavPoints: []epub.NavPoint{
			{Label: "Chapter 1", ContentPath: "text/ch01.xhtml"},
		},
	}
	chapterIDs := map[string]string{
		"text/ch01.xhtml": "ch01",
	}
	gen := NewTOCGenerator(ncx, chapterIDs)

	html := `<html><head></head><body class="vrtl"><div id="ch01">content</div></body></html>`
	result := gen.InsertInlineTOC(html)

	// TOC should be inserted right after <body class="vrtl">
	tocIdx := strings.Index(result, `<div id="toc">`)
	if tocIdx < 0 {
		t.Fatal("expected toc div to be inserted")
	}
	bodyCloseIdx := strings.Index(result, `<body class="vrtl">`) + len(`<body class="vrtl">`)
	if tocIdx != bodyCloseIdx {
		t.Errorf("expected toc right after <body class=\"vrtl\">, bodyEnd=%d, tocStart=%d", bodyCloseIdx, tocIdx)
	}
}

func TestInsertInlineTOC_EmptyNavPoints(t *testing.T) {
	ncx := &epub.NCX{
		NavPoints: nil,
	}
	gen := NewTOCGenerator(ncx, nil)

	html := `<html><head></head><body><div>content</div></body></html>`
	result := gen.InsertInlineTOC(html)

	// Should return original HTML unchanged
	if result != html {
		t.Errorf("expected unchanged HTML for empty NavPoints, got: %s", result)
	}
}

// --- BuildTOCEntries tests ---

func TestBuildTOCEntries_Basic(t *testing.T) {
	ncx := &epub.NCX{
		NavPoints: []epub.NavPoint{
			{Label: "Chapter 1", ContentPath: "text/ch01.xhtml"},
			{Label: "Chapter 2", ContentPath: "text/ch02.xhtml"},
		},
	}
	chapterIDs := map[string]string{
		"text/ch01.xhtml": "ch01",
		"text/ch02.xhtml": "ch02",
	}
	gen := NewTOCGenerator(ncx, chapterIDs)

	// Build HTML with chapter divs
	html := []byte(`<html><body><div id="ch01"><p>Chapter 1 content</p></div><div id="ch02"><p>Chapter 2 content</p></div></body></html>`)
	entries, err := gen.BuildTOCEntries(html)
	if err != nil {
		t.Fatalf("BuildTOCEntries failed: %v", err)
	}

	if len(entries) != 2 {
		t.Fatalf("expected 2 entries, got %d", len(entries))
	}

	if entries[0].Label != "Chapter 1" {
		t.Errorf("expected label 'Chapter 1', got %q", entries[0].Label)
	}
	if entries[1].Label != "Chapter 2" {
		t.Errorf("expected label 'Chapter 2', got %q", entries[1].Label)
	}

	// FilePos should point to the byte offset of "id=\"ch01\"" tag
	if entries[0].FilePos == 0 && entries[1].FilePos == 0 {
		t.Error("expected non-zero filepos for at least one entry")
	}

	// ch02 should have a higher filepos than ch01
	if entries[1].FilePos <= entries[0].FilePos {
		t.Errorf("expected ch02 filepos > ch01 filepos, got ch01=%d, ch02=%d",
			entries[0].FilePos, entries[1].FilePos)
	}
}

func TestBuildTOCEntries_WithFragment(t *testing.T) {
	ncx := &epub.NCX{
		NavPoints: []epub.NavPoint{
			{Label: "Section 1", ContentPath: "text/ch01.xhtml", Fragment: "sec1"},
		},
	}
	chapterIDs := map[string]string{
		"text/ch01.xhtml": "ch01",
	}
	gen := NewTOCGenerator(ncx, chapterIDs)

	html := []byte(`<html><body><div id="ch01"><h2 id="ch01-sec1">Section 1</h2><p>content</p></div></body></html>`)
	entries, err := gen.BuildTOCEntries(html)
	if err != nil {
		t.Fatalf("BuildTOCEntries failed: %v", err)
	}

	if len(entries) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(entries))
	}

	// Should find the position of id="ch01-sec1"
	if entries[0].FilePos == 0 {
		t.Error("expected non-zero filepos for fragment entry")
	}
}

func TestBuildTOCEntries_UTF8Multibyte(t *testing.T) {
	ncx := &epub.NCX{
		NavPoints: []epub.NavPoint{
			{Label: "第1章", ContentPath: "text/ch01.xhtml"},
			{Label: "第2章", ContentPath: "text/ch02.xhtml"},
		},
	}
	chapterIDs := map[string]string{
		"text/ch01.xhtml": "ch01",
		"text/ch02.xhtml": "ch02",
	}
	gen := NewTOCGenerator(ncx, chapterIDs)

	// HTML with multibyte characters before ch02
	html := []byte(`<html><body><div id="ch01"><p>日本語テキスト</p></div><div id="ch02"><p>more content</p></div></body></html>`)
	entries, err := gen.BuildTOCEntries(html)
	if err != nil {
		t.Fatalf("BuildTOCEntries failed: %v", err)
	}

	if len(entries) != 2 {
		t.Fatalf("expected 2 entries, got %d", len(entries))
	}

	// Verify filepos accounts for multibyte UTF-8 bytes
	// "日本語テキスト" = 7 chars × 3 bytes = 21 bytes
	// ch02 filepos should be larger due to multibyte content
	if entries[1].FilePos <= entries[0].FilePos {
		t.Errorf("expected ch02 filepos > ch01 filepos with multibyte content")
	}
}

func TestBuildTOCEntries_Nested(t *testing.T) {
	ncx := &epub.NCX{
		NavPoints: []epub.NavPoint{
			{
				Label:       "Part 1",
				ContentPath: "text/ch01.xhtml",
				Children: []epub.NavPoint{
					{Label: "Chapter 1.1", ContentPath: "text/ch01.xhtml", Fragment: "s1"},
				},
			},
		},
	}
	chapterIDs := map[string]string{
		"text/ch01.xhtml": "ch01",
	}
	gen := NewTOCGenerator(ncx, chapterIDs)

	html := []byte(`<html><body><div id="ch01"><h1>Part 1</h1><h2 id="ch01-s1">Chapter 1.1</h2></div></body></html>`)
	entries, err := gen.BuildTOCEntries(html)
	if err != nil {
		t.Fatalf("BuildTOCEntries failed: %v", err)
	}

	if len(entries) != 1 {
		t.Fatalf("expected 1 top-level entry, got %d", len(entries))
	}
	if len(entries[0].Children) != 1 {
		t.Fatalf("expected 1 child entry, got %d", len(entries[0].Children))
	}
	if entries[0].Children[0].Label != "Chapter 1.1" {
		t.Errorf("expected child label 'Chapter 1.1', got %q", entries[0].Children[0].Label)
	}
}

func TestBuildTOCEntries_FragmentNotFound_FallbackToChapter(t *testing.T) {
	ncx := &epub.NCX{
		NavPoints: []epub.NavPoint{
			{Label: "Missing Section", ContentPath: "text/ch01.xhtml", Fragment: "nonexistent"},
		},
	}
	chapterIDs := map[string]string{
		"text/ch01.xhtml": "ch01",
	}
	gen := NewTOCGenerator(ncx, chapterIDs)

	html := []byte(`<html><body><div id="ch01"><p>content</p></div></body></html>`)
	entries, err := gen.BuildTOCEntries(html)
	if err != nil {
		t.Fatalf("BuildTOCEntries should not error on missing fragment: %v", err)
	}

	// Should fall back to chapter start when fragment not found
	if len(entries) != 1 {
		t.Fatalf("expected 1 entry (fallback to chapter), got %d", len(entries))
	}

	// FilePos should point to the chapter div start, not 0
	expectedPos := strings.Index(string(html), `<div id="ch01">`)
	if entries[0].FilePos != uint32(expectedPos) {
		t.Errorf("expected filepos %d (chapter start), got %d", expectedPos, entries[0].FilePos)
	}
}

func TestBuildTOCEntries_ContentPathNotMapped(t *testing.T) {
	ncx := &epub.NCX{
		NavPoints: []epub.NavPoint{
			{Label: "Unknown", ContentPath: "text/unknown.xhtml"},
		},
	}
	chapterIDs := map[string]string{
		"text/ch01.xhtml": "ch01",
	}
	gen := NewTOCGenerator(ncx, chapterIDs)

	html := []byte(`<html><body><div id="ch01"><p>content</p></div></body></html>`)
	entries, err := gen.BuildTOCEntries(html)
	if err != nil {
		t.Fatalf("BuildTOCEntries should not error on unmapped path: %v", err)
	}

	// Unresolved entries should be skipped
	if len(entries) != 0 {
		t.Fatalf("expected 0 entries (unresolved skipped), got %d", len(entries))
	}
}

// --- resolveTargetID tests ---

func TestResolveTargetID_ChapterOnly(t *testing.T) {
	chapterIDs := map[string]string{
		"text/ch01.xhtml": "ch01",
	}
	gen := NewTOCGenerator(nil, chapterIDs)

	id := gen.resolveTargetID("text/ch01.xhtml", "")
	if id != "ch01" {
		t.Errorf("expected 'ch01', got %q", id)
	}
}

func TestResolveTargetID_WithFragment(t *testing.T) {
	chapterIDs := map[string]string{
		"text/ch01.xhtml": "ch01",
	}
	gen := NewTOCGenerator(nil, chapterIDs)

	id := gen.resolveTargetID("text/ch01.xhtml", "section1")
	if id != "ch01-section1" {
		t.Errorf("expected 'ch01-section1', got %q", id)
	}
}

func TestResolveTargetID_FragmentSanitized(t *testing.T) {
	chapterIDs := map[string]string{
		"text/ch01.xhtml": "ch01",
	}
	gen := NewTOCGenerator(nil, chapterIDs)

	// Special characters should be URL-encoded (matching sanitizeFragmentForHTMLID behavior)
	id := gen.resolveTargetID("text/ch01.xhtml", "a&b")
	if id != "ch01-a%26b" {
		t.Errorf("expected 'ch01-a%%26b', got %q", id)
	}
}

func TestResolveTargetID_UnmappedPath(t *testing.T) {
	chapterIDs := map[string]string{
		"text/ch01.xhtml": "ch01",
	}
	gen := NewTOCGenerator(nil, chapterIDs)

	id := gen.resolveTargetID("text/unknown.xhtml", "")
	if id != "" {
		t.Errorf("expected empty string for unmapped path, got %q", id)
	}
}

// --- calculateFilePos tests ---

func TestCalculateFilePos_ChapterDiv(t *testing.T) {
	chapterIDs := map[string]string{
		"text/ch01.xhtml": "ch01",
	}
	gen := NewTOCGenerator(nil, chapterIDs)

	html := []byte(`<html><body><div id="ch01"><p>content</p></div></body></html>`)
	pos, err := gen.calculateFilePos(html, "text/ch01.xhtml", "")
	if err != nil {
		t.Fatalf("calculateFilePos failed: %v", err)
	}

	// Should point to the byte position of '<' before id="ch01"
	expected := strings.Index(string(html), `<div id="ch01">`)
	if expected < 0 {
		t.Fatal("test HTML does not contain expected pattern")
	}
	if pos != uint32(expected) {
		t.Errorf("expected filepos %d, got %d", expected, pos)
	}
}

func TestCalculateFilePos_FragmentID(t *testing.T) {
	chapterIDs := map[string]string{
		"text/ch01.xhtml": "ch01",
	}
	gen := NewTOCGenerator(nil, chapterIDs)

	html := []byte(`<html><body><div id="ch01"><h2 id="ch01-sec1">Section 1</h2></div></body></html>`)
	pos, err := gen.calculateFilePos(html, "text/ch01.xhtml", "sec1")
	if err != nil {
		t.Fatalf("calculateFilePos failed: %v", err)
	}

	expected := strings.Index(string(html), `<h2 id="ch01-sec1">`)
	if expected < 0 {
		t.Fatal("test HTML does not contain expected pattern")
	}
	if pos != uint32(expected) {
		t.Errorf("expected filepos %d, got %d", expected, pos)
	}
}
