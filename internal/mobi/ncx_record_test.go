package mobi

import (
	"strings"
	"testing"
)

func TestGenerateNCXRecord_Basic(t *testing.T) {
	cfg := NCXRecordConfig{
		Title: "Test Book",
		Entries: []NCXEntry{
			{Label: "Chapter 1", FilePos: 100},
			{Label: "Chapter 2", FilePos: 200},
		},
	}
	result := GenerateNCXRecord(cfg)
	html := string(result)

	// Should contain the title
	if !strings.Contains(html, "<h1>Test Book</h1>") {
		t.Error("expected title in NCX record")
	}

	// Should contain filepos entries
	if !strings.Contains(html, `filepos="00000100"`) {
		t.Errorf("expected filepos 00000100, got: %s", html)
	}
	if !strings.Contains(html, `filepos="00000200"`) {
		t.Errorf("expected filepos 00000200, got: %s", html)
	}

	// Should contain labels
	if !strings.Contains(html, "Chapter 1") {
		t.Error("expected Chapter 1 label")
	}
	if !strings.Contains(html, "Chapter 2") {
		t.Error("expected Chapter 2 label")
	}

	// Should have <ul> and <li> structure
	if !strings.Contains(html, "<ul>") {
		t.Error("expected <ul> tag")
	}
	if !strings.Contains(html, "<li>") {
		t.Error("expected <li> tag")
	}

	// Should be valid HTML structure
	if !strings.HasPrefix(html, "<html>") {
		t.Error("expected HTML to start with <html>")
	}
	if !strings.HasSuffix(html, "</html>") {
		t.Error("expected HTML to end with </html>")
	}
}

func TestGenerateNCXRecord_Nested(t *testing.T) {
	cfg := NCXRecordConfig{
		Title: "Nested Book",
		Entries: []NCXEntry{
			{
				Label:   "Part 1",
				FilePos: 100,
				Children: []NCXEntry{
					{Label: "Chapter 1.1", FilePos: 150},
					{Label: "Chapter 1.2", FilePos: 200},
				},
			},
			{Label: "Part 2", FilePos: 300},
		},
	}
	result := GenerateNCXRecord(cfg)
	html := string(result)

	// Should contain all entries
	if !strings.Contains(html, "Part 1") {
		t.Error("expected Part 1")
	}
	if !strings.Contains(html, "Chapter 1.1") {
		t.Error("expected Chapter 1.1")
	}
	if !strings.Contains(html, "Chapter 1.2") {
		t.Error("expected Chapter 1.2")
	}
	if !strings.Contains(html, "Part 2") {
		t.Error("expected Part 2")
	}

	// Should have nested <ul> for children
	// Count occurrences of <ul>: should be at least 2 (root + nested)
	ulCount := strings.Count(html, "<ul>")
	if ulCount < 2 {
		t.Errorf("expected at least 2 <ul> tags for nested structure, got %d", ulCount)
	}

	// Should have correct filepos values
	if !strings.Contains(html, `filepos="00000100"`) {
		t.Error("expected filepos 00000100 for Part 1")
	}
	if !strings.Contains(html, `filepos="00000150"`) {
		t.Error("expected filepos 00000150 for Chapter 1.1")
	}
}

func TestGenerateNCXRecord_WithGuide(t *testing.T) {
	cfg := NCXRecordConfig{
		Title: "Guided Book",
		Entries: []NCXEntry{
			{Label: "Chapter 1", FilePos: 100},
		},
		Guide: []GuideReference{
			{Type: "toc", Title: "Table of Contents", FilePos: 0},
			{Type: "text", Title: "Start", FilePos: 100},
		},
	}
	result := GenerateNCXRecord(cfg)
	html := string(result)

	// Should contain guide section in head
	if !strings.Contains(html, "<guide>") {
		t.Error("expected <guide> tag")
	}
	if !strings.Contains(html, "</guide>") {
		t.Error("expected </guide> tag")
	}

	// Should contain reference elements with filepos
	if !strings.Contains(html, `type="toc"`) {
		t.Error("expected toc reference type")
	}
	if !strings.Contains(html, `type="text"`) {
		t.Error("expected text reference type")
	}
	if !strings.Contains(html, `title="Table of Contents"`) {
		t.Error("expected Table of Contents title")
	}
	if !strings.Contains(html, `filepos="00000000"`) {
		t.Error("expected filepos 00000000 for toc reference")
	}
	if !strings.Contains(html, `filepos="00000100"`) {
		t.Error("expected filepos 00000100 for text reference")
	}

	// Guide should be inside <head>
	headStart := strings.Index(html, "<head>")
	headEnd := strings.Index(html, "</head>")
	guideStart := strings.Index(html, "<guide>")
	if guideStart < headStart || guideStart > headEnd {
		t.Error("expected <guide> to be inside <head>")
	}
}

func TestGenerateNCXRecord_Empty(t *testing.T) {
	cfg := NCXRecordConfig{
		Title:   "Empty Book",
		Entries: nil,
	}
	result := GenerateNCXRecord(cfg)
	html := string(result)

	// Should still generate valid HTML
	if !strings.Contains(html, "<html>") {
		t.Error("expected <html> tag")
	}
	if !strings.Contains(html, "<h1>Empty Book</h1>") {
		t.Error("expected title")
	}

	// Should not have any list items
	if strings.Contains(html, "<li>") {
		t.Error("expected no <li> tags for empty entries")
	}
}

func TestGenerateNCXRecord_FilePosPadding(t *testing.T) {
	cfg := NCXRecordConfig{
		Title: "Test",
		Entries: []NCXEntry{
			{Label: "Small", FilePos: 5},
			{Label: "Large", FilePos: 12345678},
		},
	}
	result := GenerateNCXRecord(cfg)
	html := string(result)

	// FilePos should be 8-digit zero-padded
	if !strings.Contains(html, `filepos="00000005"`) {
		t.Errorf("expected zero-padded filepos for 5, got: %s", html)
	}
	if !strings.Contains(html, `filepos="12345678"`) {
		t.Errorf("expected filepos 12345678, got: %s", html)
	}
}

func TestGenerateNCXRecord_HTMLEscaping(t *testing.T) {
	cfg := NCXRecordConfig{
		Title: "Book <Special>",
		Entries: []NCXEntry{
			{Label: "Chapter & \"Quotes\"", FilePos: 100},
		},
	}
	result := GenerateNCXRecord(cfg)
	html := string(result)

	// Labels should be HTML-escaped
	if strings.Contains(html, "<Special>") {
		t.Error("title should be HTML-escaped")
	}
	if !strings.Contains(html, "&lt;Special&gt;") {
		t.Errorf("expected escaped title, got: %s", html)
	}

	if strings.Contains(html, `& "`) {
		t.Error("label should be HTML-escaped")
	}
}
