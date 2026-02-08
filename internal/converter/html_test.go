package converter

import (
	"fmt"
	"strings"
	"testing"

	"github.com/PuerkitoBio/goquery"
	"github.com/yuanying/epub2azw3/internal/epub"
)

// TestHTMLBuilder_Build tests the basic HTML integration functionality
func TestHTMLBuilder_Build(t *testing.T) {
	// Create test XHTML content
	chapter1HTML := `<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE html PUBLIC "-//W3C//DTD XHTML 1.1//EN" "http://www.w3.org/TR/xhtml11/DTD/xhtml11.dtd">
<html xmlns="http://www.w3.org/1999/xhtml">
<head><title>Chapter 1</title></head>
<body>
<h1>第1章</h1>
<p>これは第1章の内容です。</p>
</body>
</html>`

	chapter2HTML := `<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE html PUBLIC "-//W3C//DTD XHTML 1.1//EN" "http://www.w3.org/TR/xhtml11/DTD/xhtml11.dtd">
<html xmlns="http://www.w3.org/1999/xhtml">
<head><title>Chapter 2</title></head>
<body>
<h1>第2章</h1>
<p>これは第2章の内容です。</p>
</body>
</html>`

	// Load content using epub.LoadContent
	content1, err := epub.LoadContent("ch1", "text/chapter01.xhtml", []byte(chapter1HTML))
	if err != nil {
		t.Fatalf("Failed to load chapter 1: %v", err)
	}

	content2, err := epub.LoadContent("ch2", "text/chapter02.xhtml", []byte(chapter2HTML))
	if err != nil {
		t.Fatalf("Failed to load chapter 2: %v", err)
	}

	// Create HTMLBuilder
	builder := NewHTMLBuilder()

	// Add chapters
	err = builder.AddChapter(content1)
	if err != nil {
		t.Fatalf("Failed to add chapter 1: %v", err)
	}

	err = builder.AddChapter(content2)
	if err != nil {
		t.Fatalf("Failed to add chapter 2: %v", err)
	}

	// Build the integrated HTML
	result, err := builder.Build()
	if err != nil {
		t.Fatalf("Failed to build HTML: %v", err)
	}

	// Parse result to verify structure
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(result))
	if err != nil {
		t.Fatalf("Failed to parse result HTML: %v", err)
	}

	// Verify chapter 1 div exists with correct ID
	ch1 := doc.Find("#ch01")
	if ch1.Length() != 1 {
		t.Errorf("Expected 1 div with id='ch01', got %d", ch1.Length())
	}

	// Verify chapter 1 content
	if !strings.Contains(ch1.Text(), "第1章") {
		t.Errorf("Chapter 1 div should contain '第1章'")
	}
	if !strings.Contains(ch1.Text(), "これは第1章の内容です。") {
		t.Errorf("Chapter 1 div should contain chapter 1 text")
	}

	// Verify chapter 2 div exists with correct ID
	ch2 := doc.Find("#ch02")
	if ch2.Length() != 1 {
		t.Errorf("Expected 1 div with id='ch02', got %d", ch2.Length())
	}

	// Verify chapter 2 content
	if !strings.Contains(ch2.Text(), "第2章") {
		t.Errorf("Chapter 2 div should contain '第2章'")
	}
	if !strings.Contains(ch2.Text(), "これは第2章の内容です。") {
		t.Errorf("Chapter 2 div should contain chapter 2 text")
	}

	// Verify pagebreak elements
	pagebreaks := doc.Find("mbp\\:pagebreak")
	if pagebreaks.Length() != 2 {
		t.Errorf("Expected 2 pagebreak elements, got %d", pagebreaks.Length())
	}

	// Verify order (ch01 should come before ch02)
	allDivs := doc.Find("body > div")
	if allDivs.Length() != 2 {
		t.Errorf("Expected 2 chapter divs, got %d", allDivs.Length())
	}
	firstID, _ := allDivs.First().Attr("id")
	if firstID != "ch01" {
		t.Errorf("First chapter should have id='ch01', got '%s'", firstID)
	}
	lastID, _ := allDivs.Last().Attr("id")
	if lastID != "ch02" {
		t.Errorf("Second chapter should have id='ch02', got '%s'", lastID)
	}
}

// TestHTMLBuilder_ResolveLinks tests link resolution functionality
func TestHTMLBuilder_ResolveLinks(t *testing.T) {
	tests := []struct {
		name     string
		html     string
		expected map[string]string // href -> expected transformed href
	}{
		{
			name: "Internal chapter link",
			html: `<body><a href="chapter02.xhtml">Next Chapter</a></body>`,
			expected: map[string]string{
				"chapter02.xhtml": "#ch02",
			},
		},
		{
			name: "Internal chapter link with fragment",
			html: `<body><a href="chapter02.xhtml#section1">Section 1</a></body>`,
			expected: map[string]string{
				"chapter02.xhtml#section1": "#ch02-section1",
			},
		},
		{
			name: "Fragment only link",
			html: `<body><a href="#section1">Section 1</a></body>`,
			expected: map[string]string{
				"#section1": "#ch01-section1",
			},
		},
		{
			name: "External HTTP link",
			html: `<body><a href="http://example.com">External</a></body>`,
			expected: map[string]string{
				"http://example.com": "http://example.com",
			},
		},
		{
			name: "External HTTPS link",
			html: `<body><a href="https://example.com">External</a></body>`,
			expected: map[string]string{
				"https://example.com": "https://example.com",
			},
		},
		{
			name: "Internal link with fragment containing special characters",
			html: `<body><a href="chapter02.xhtml#section&param">Section</a></body>`,
			expected: map[string]string{
				"chapter02.xhtml#section&param": "#ch02-section%26param",
			},
		},
		{
			name: "Internal link with fragment containing non-ASCII",
			html: `<body><a href="chapter02.xhtml#第1節">Section</a></body>`,
			expected: map[string]string{
				"chapter02.xhtml#第1節": "#ch02-%E7%AC%AC1%E7%AF%80",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a simple XHTML document
			xhtml := `<?xml version="1.0" encoding="UTF-8"?>
<html xmlns="http://www.w3.org/1999/xhtml">
<head><title>Test</title></head>
` + tt.html + `
</html>`

			content, err := epub.LoadContent("ch1", "text/chapter01.xhtml", []byte(xhtml))
			if err != nil {
				t.Fatalf("Failed to load content: %v", err)
			}

			// Add a second chapter for link resolution
			chapter2HTML := `<?xml version="1.0" encoding="UTF-8"?>
<html xmlns="http://www.w3.org/1999/xhtml">
<head><title>Chapter 2</title></head>
<body><h1>Chapter 2</h1></body>
</html>`
			content2, err := epub.LoadContent("ch2", "text/chapter02.xhtml", []byte(chapter2HTML))
			if err != nil {
				t.Fatalf("Failed to load chapter 2: %v", err)
			}

			builder := NewHTMLBuilder()
			if err := builder.AddChapter(content); err != nil {
				t.Fatalf("Failed to add chapter 1: %v", err)
			}
			if err := builder.AddChapter(content2); err != nil {
				t.Fatalf("Failed to add chapter 2: %v", err)
			}

			result, err := builder.Build()
			if err != nil {
				t.Fatalf("Failed to build HTML: %v", err)
			}

			doc, err := goquery.NewDocumentFromReader(strings.NewReader(result))
			if err != nil {
				t.Fatalf("Failed to parse result: %v", err)
			}

			// Check each expected link transformation
			for original, expected := range tt.expected {
				found := false
				doc.Find("a").Each(func(i int, s *goquery.Selection) {
					if href, exists := s.Attr("href"); exists && href == expected {
						found = true
					}
				})
				if !found {
					t.Errorf("Expected to find link '%s' (transformed from '%s'), but not found", expected, original)
				}
			}
		})
	}
}

// TestHTMLBuilder_RewritesIDsAndFragments ensures IDs are namespaced and links point to the rewritten IDs
func TestHTMLBuilder_RewritesIDsAndFragments(t *testing.T) {
	ch1HTML := `<?xml version="1.0" encoding="UTF-8"?>
<html xmlns="http://www.w3.org/1999/xhtml">
<head><title>Chapter 1</title></head>
<body>
<h2 id="intro">Intro</h2>
<p><a href="#intro">Back to intro</a></p>
<p><a href="chapter02.xhtml#section1">Next section</a></p>
</body>
</html>`

	ch2HTML := `<?xml version="1.0" encoding="UTF-8"?>
<html xmlns="http://www.w3.org/1999/xhtml">
<head><title>Chapter 2</title></head>
<body>
<h2 id="section1">Section 1</h2>
</body>
</html>`

	content1, err := epub.LoadContent("ch1", "text/chapter01.xhtml", []byte(ch1HTML))
	if err != nil {
		t.Fatalf("Failed to load chapter 1: %v", err)
	}
	content2, err := epub.LoadContent("ch2", "text/chapter02.xhtml", []byte(ch2HTML))
	if err != nil {
		t.Fatalf("Failed to load chapter 2: %v", err)
	}

	builder := NewHTMLBuilder()
	if err := builder.AddChapter(content1); err != nil {
		t.Fatalf("Failed to add chapter 1: %v", err)
	}
	if err := builder.AddChapter(content2); err != nil {
		t.Fatalf("Failed to add chapter 2: %v", err)
	}

	result, err := builder.Build()
	if err != nil {
		t.Fatalf("Failed to build HTML: %v", err)
	}

	doc, err := goquery.NewDocumentFromReader(strings.NewReader(result))
	if err != nil {
		t.Fatalf("Failed to parse integrated HTML: %v", err)
	}

	// IDs should be namespaced with chapter IDs
	if doc.Find("#ch01-intro").Length() != 1 {
		t.Fatalf("Expected namespaced intro ID #ch01-intro to exist")
	}
	if doc.Find("#ch02-section1").Length() != 1 {
		t.Fatalf("Expected namespaced section ID #ch02-section1 to exist")
	}

	// Links should point at rewritten IDs
	if doc.Find(`a[href="#ch01-intro"]`).Length() != 1 {
		t.Errorf("Expected fragment-only link to rewrite to #ch01-intro")
	}
	if doc.Find(`a[href="#ch02-section1"]`).Length() != 1 {
		t.Errorf("Expected cross-chapter link to rewrite to #ch02-section1")
	}
}

// TestHTMLBuilder_KoboFragmentPreservation tests that #kobo.* fragments are preserved
func TestHTMLBuilder_KoboFragmentPreservation(t *testing.T) {
	tests := []struct {
		name     string
		html     string
		expected map[string]string
	}{
		{
			name: "Fragment-only kobo link preserved",
			html: `<body><a href="#kobo.1.1">Kobo Link</a></body>`,
			expected: map[string]string{
				"#kobo.1.1": "#kobo.1.1",
			},
		},
		{
			name: "Cross-chapter kobo link preserves fragment without sanitization",
			html: `<body><a href="chapter02.xhtml#kobo.2.3">Kobo Link</a></body>`,
			expected: map[string]string{
				"chapter02.xhtml#kobo.2.3": "#kobo.2.3",
			},
		},
		{
			name: "Normal fragment still transformed",
			html: `<body><a href="#section1">Normal Link</a></body>`,
			expected: map[string]string{
				"#section1": "#ch01-section1",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			xhtml := `<?xml version="1.0" encoding="UTF-8"?>
<html xmlns="http://www.w3.org/1999/xhtml">
<head><title>Test</title></head>
` + tt.html + `
</html>`

			content, err := epub.LoadContent("ch1", "text/chapter01.xhtml", []byte(xhtml))
			if err != nil {
				t.Fatalf("Failed to load content: %v", err)
			}

			chapter2HTML := `<?xml version="1.0" encoding="UTF-8"?>
<html xmlns="http://www.w3.org/1999/xhtml">
<head><title>Chapter 2</title></head>
<body><h1>Chapter 2</h1></body>
</html>`
			content2, err := epub.LoadContent("ch2", "text/chapter02.xhtml", []byte(chapter2HTML))
			if err != nil {
				t.Fatalf("Failed to load chapter 2: %v", err)
			}

			builder := NewHTMLBuilder()
			if err := builder.AddChapter(content); err != nil {
				t.Fatalf("Failed to add chapter 1: %v", err)
			}
			if err := builder.AddChapter(content2); err != nil {
				t.Fatalf("Failed to add chapter 2: %v", err)
			}

			result, err := builder.Build()
			if err != nil {
				t.Fatalf("Failed to build HTML: %v", err)
			}

			doc, err := goquery.NewDocumentFromReader(strings.NewReader(result))
			if err != nil {
				t.Fatalf("Failed to parse result: %v", err)
			}

			for original, expected := range tt.expected {
				found := false
				doc.Find("a").Each(func(i int, s *goquery.Selection) {
					if href, exists := s.Attr("href"); exists && href == expected {
						found = true
					}
				})
				if !found {
					// Debug: show all href values
					var hrefs []string
					doc.Find("a").Each(func(i int, s *goquery.Selection) {
						if href, exists := s.Attr("href"); exists {
							hrefs = append(hrefs, href)
						}
					})
					t.Errorf("Expected link '%s' (from '%s'), but not found. Actual hrefs: %v", expected, original, hrefs)
				}
			}
		})
	}
}

// TestHTMLBuilder_KoboIDsNotNamespaced ensures kobo.* IDs are preserved
func TestHTMLBuilder_KoboIDsNotNamespaced(t *testing.T) {
	ch1HTML := `<?xml version="1.0" encoding="UTF-8"?>
<html xmlns="http://www.w3.org/1999/xhtml">
<head><title>Chapter 1</title></head>
<body>
<h2 id="kobo.1.1">Kobo</h2>
</body>
</html>`

	content1, err := epub.LoadContent("ch1", "text/chapter01.xhtml", []byte(ch1HTML))
	if err != nil {
		t.Fatalf("Failed to load chapter 1: %v", err)
	}

	builder := NewHTMLBuilder()
	if err := builder.AddChapter(content1); err != nil {
		t.Fatalf("Failed to add chapter 1: %v", err)
	}

	result, err := builder.Build()
	if err != nil {
		t.Fatalf("Failed to build HTML: %v", err)
	}

	doc, err := goquery.NewDocumentFromReader(strings.NewReader(result))
	if err != nil {
		t.Fatalf("Failed to parse integrated HTML: %v", err)
	}

	if doc.Find(`[id="kobo.1.1"]`).Length() != 1 {
		t.Fatalf("Expected kobo ID to be preserved without namespacing")
	}
}

// TestHTMLBuilder_BodyAttrsInheritance tests that body/html attributes are applied to chapter divs
func TestHTMLBuilder_BodyAttrsInheritance(t *testing.T) {
	ch1HTML := `<?xml version="1.0" encoding="UTF-8"?>
<html xmlns="http://www.w3.org/1999/xhtml">
<head><title>Chapter 1</title></head>
<body class="vrtl" dir="rtl">
<h1>第1章</h1>
</body>
</html>`

	ch2HTML := `<?xml version="1.0" encoding="UTF-8"?>
<html xmlns="http://www.w3.org/1999/xhtml" lang="ja">
<head><title>Chapter 2</title></head>
<body>
<h1>第2章</h1>
</body>
</html>`

	content1, err := epub.LoadContent("ch1", "text/chapter01.xhtml", []byte(ch1HTML))
	if err != nil {
		t.Fatalf("Failed to load chapter 1: %v", err)
	}
	content2, err := epub.LoadContent("ch2", "text/chapter02.xhtml", []byte(ch2HTML))
	if err != nil {
		t.Fatalf("Failed to load chapter 2: %v", err)
	}

	builder := NewHTMLBuilder()
	if err := builder.AddChapter(content1); err != nil {
		t.Fatalf("Failed to add chapter 1: %v", err)
	}
	if err := builder.AddChapter(content2); err != nil {
		t.Fatalf("Failed to add chapter 2: %v", err)
	}

	result, err := builder.Build()
	if err != nil {
		t.Fatalf("Failed to build HTML: %v", err)
	}

	doc, err := goquery.NewDocumentFromReader(strings.NewReader(result))
	if err != nil {
		t.Fatalf("Failed to parse result: %v", err)
	}

	// Chapter 1 should have class="vrtl" and dir="rtl"
	ch1Div := doc.Find("#ch01")
	if ch1Div.Length() != 1 {
		t.Fatal("Expected #ch01 div to exist")
	}
	if class, _ := ch1Div.Attr("class"); class != "vrtl" {
		t.Errorf("ch01 class = %q, want %q", class, "vrtl")
	}
	if dir, _ := ch1Div.Attr("dir"); dir != "rtl" {
		t.Errorf("ch01 dir = %q, want %q", dir, "rtl")
	}

	// Chapter 2 should have lang="ja" (from html element fallback)
	ch2Div := doc.Find("#ch02")
	if ch2Div.Length() != 1 {
		t.Fatal("Expected #ch02 div to exist")
	}
	if lang, _ := ch2Div.Attr("lang"); lang != "ja" {
		t.Errorf("ch02 lang = %q, want %q", lang, "ja")
	}
}

// TestHTMLBuilder_IntegrateCSS tests CSS integration functionality
func TestHTMLBuilder_IntegrateCSS(t *testing.T) {
	css1 := `body { margin: 0; padding: 0; }`
	css2 := `h1 { font-size: 2em; color: #333; }`

	builder := NewHTMLBuilder()
	builder.AddCSS(css1)
	builder.AddCSS(css2)

	// Add a simple chapter
	chapterHTML := `<?xml version="1.0" encoding="UTF-8"?>
<html xmlns="http://www.w3.org/1999/xhtml">
<head><title>Chapter 1</title></head>
<body><h1>Test</h1></body>
</html>`

	content, err := epub.LoadContent("ch1", "text/chapter01.xhtml", []byte(chapterHTML))
	if err != nil {
		t.Fatalf("Failed to load content: %v", err)
	}

	if err := builder.AddChapter(content); err != nil {
		t.Fatalf("Failed to add chapter: %v", err)
	}

	result, err := builder.Build()
	if err != nil {
		t.Fatalf("Failed to build HTML: %v", err)
	}

	// Verify that CSS is included in a style tag
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(result))
	if err != nil {
		t.Fatalf("Failed to parse result: %v", err)
	}

	styleTag := doc.Find("head style")
	if styleTag.Length() != 1 {
		t.Errorf("Expected 1 style tag in head, got %d", styleTag.Length())
	}

	styleContent := styleTag.Text()
	if !strings.Contains(styleContent, "margin: 0") {
		t.Errorf("Style tag should contain CSS from css1")
	}
	if !strings.Contains(styleContent, "font-size: 2em") {
		t.Errorf("Style tag should contain CSS from css2")
	}
}

// TestHTMLBuilder_AddChapterCSS tests CSS ID selector namespacing
func TestHTMLBuilder_AddChapterCSS(t *testing.T) {
	tests := []struct {
		name      string
		chapterID string
		css       string
		want      string
	}{
		{
			name:      "single ID selector",
			chapterID: "ch01",
			css:       "#cover { width: 100% }",
			want:      "#ch01-cover { width: 100% }",
		},
		{
			name:      "multiple ID selectors",
			chapterID: "ch01",
			css:       "#intro, #outro { color: red }",
			want:      "#ch01-intro, #ch01-outro { color: red }",
		},
		{
			name:      "class selector not transformed",
			chapterID: "ch01",
			css:       ".myclass { font-size: 12px }",
			want:      ".myclass { font-size: 0.75em }",
		},
		{
			name:      "ID selector with class descendant",
			chapterID: "ch01",
			css:       "#header .content { margin: 0 }",
			want:      "#ch01-header .content { margin: 0 }",
		},
		{
			name:      "no ID selectors",
			chapterID: "ch01",
			css:       "body { margin: 0 }",
			want:      "body { margin: 0 }",
		},
		{
			name:      "color codes not affected",
			chapterID: "ch01",
			css:       "h1 { color: #333; background: #ff0000 }",
			want:      "h1 { color: #333; background: #ff0000 }",
		},
		{
			name:      "ID selector with color code in same rule",
			chapterID: "ch02",
			css:       "#title { color: #333 }",
			want:      "#ch02-title { color: #333 }",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			builder := NewHTMLBuilder()
			builder.AddChapterCSS(tt.chapterID, tt.css)

			// Build with a dummy chapter to get result
			chapterHTML := `<?xml version="1.0" encoding="UTF-8"?>
<html xmlns="http://www.w3.org/1999/xhtml">
<head><title>Test</title></head>
<body><h1>Test</h1></body>
</html>`
			content, err := epub.LoadContent("ch1", "text/chapter01.xhtml", []byte(chapterHTML))
			if err != nil {
				t.Fatalf("Failed to load content: %v", err)
			}
			if err := builder.AddChapter(content); err != nil {
				t.Fatalf("Failed to add chapter: %v", err)
			}

			result, err := builder.Build()
			if err != nil {
				t.Fatalf("Failed to build HTML: %v", err)
			}

			if !strings.Contains(result, tt.want) {
				t.Errorf("Result does not contain expected CSS %q\nFull result:\n%s", tt.want, result)
			}
		})
	}
}

// TestHTMLBuilder_AddCSS_GlobalCSS tests that global CSS is not namespaced
func TestHTMLBuilder_AddCSS_GlobalCSS(t *testing.T) {
	builder := NewHTMLBuilder()
	builder.AddCSS("#global { font-size: 14px }")

	chapterHTML := `<?xml version="1.0" encoding="UTF-8"?>
<html xmlns="http://www.w3.org/1999/xhtml">
<head><title>Test</title></head>
<body><h1>Test</h1></body>
</html>`
	content, err := epub.LoadContent("ch1", "text/chapter01.xhtml", []byte(chapterHTML))
	if err != nil {
		t.Fatalf("Failed to load content: %v", err)
	}
	if err := builder.AddChapter(content); err != nil {
		t.Fatalf("Failed to add chapter: %v", err)
	}

	result, err := builder.Build()
	if err != nil {
		t.Fatalf("Failed to build HTML: %v", err)
	}

	// Global CSS should NOT be namespaced (px values are converted to em)
	if !strings.Contains(result, "#global { font-size: 0.875em }") {
		t.Errorf("Global CSS should not be namespaced. Result:\n%s", result)
	}
}

// TestHTMLBuilder_ChapterIDMapping tests that chapter IDs are correctly mapped
func TestHTMLBuilder_ChapterIDMapping(t *testing.T) {
	builder := NewHTMLBuilder()

	// Add chapters with different paths
	for i, path := range []string{
		"text/chapter01.xhtml",
		"text/chapter02.xhtml",
		"content/part1.xhtml",
	} {
		html := `<?xml version="1.0" encoding="UTF-8"?>
<html xmlns="http://www.w3.org/1999/xhtml">
<head><title>Chapter</title></head>
<body><h1>Chapter</h1></body>
</html>`
		content, err := epub.LoadContent(fmt.Sprintf("ch%d", i+1), path, []byte(html))
		if err != nil {
			t.Fatalf("Failed to load content: %v", err)
		}
		if err := builder.AddChapter(content); err != nil {
			t.Fatalf("Failed to add chapter: %v", err)
		}
	}

	// Check the chapter ID mapping
	expectedMappings := map[string]string{
		"text/chapter01.xhtml": "ch01",
		"text/chapter02.xhtml": "ch02",
		"content/part1.xhtml":  "ch03",
	}

	for path, expectedID := range expectedMappings {
		actualID := builder.GetChapterID(path)
		if actualID != expectedID {
			t.Errorf("Expected chapter ID for '%s' to be '%s', got '%s'", path, expectedID, actualID)
		}
	}
}
