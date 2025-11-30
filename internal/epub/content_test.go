package epub

import (
	"testing"
)

func TestLoadContent_SimpleXHTML(t *testing.T) {
	xhtmlContent := `<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE html>
<html xmlns="http://www.w3.org/1999/xhtml">
<head>
	<title>Chapter 1</title>
	<link rel="stylesheet" href="../css/style.css"/>
	<link rel="stylesheet" href="local.css"/>
</head>
<body>
	<h1>Chapter 1</h1>
	<p>This is a sample paragraph.</p>
	<img src="../images/photo.jpg" alt="Sample photo"/>
	<img src="diagrams/chart.png" alt="Chart"/>
</body>
</html>`

	content, err := LoadContent("chapter1", "text/chapter1.xhtml", []byte(xhtmlContent))
	if err != nil {
		t.Fatalf("LoadContent failed: %v", err)
	}

	// Test basic fields
	if content.ID != "chapter1" {
		t.Errorf("ID = %q, want %q", content.ID, "chapter1")
	}

	if content.Path != "text/chapter1.xhtml" {
		t.Errorf("Path = %q, want %q", content.Path, "text/chapter1.xhtml")
	}

	// Test CSS links collection
	expectedCSS := []string{"css/style.css", "text/local.css"}
	if len(content.CSSLinks) != len(expectedCSS) {
		t.Fatalf("CSSLinks count = %d, want %d", len(content.CSSLinks), len(expectedCSS))
	}

	for i, expected := range expectedCSS {
		if content.CSSLinks[i] != expected {
			t.Errorf("CSSLinks[%d] = %q, want %q", i, content.CSSLinks[i], expected)
		}
	}

	// Test image references collection
	expectedImages := []string{"images/photo.jpg", "text/diagrams/chart.png"}
	if len(content.ImageRefs) != len(expectedImages) {
		t.Fatalf("ImageRefs count = %d, want %d", len(content.ImageRefs), len(expectedImages))
	}

	for i, expected := range expectedImages {
		if content.ImageRefs[i] != expected {
			t.Errorf("ImageRefs[%d] = %q, want %q", i, content.ImageRefs[i], expected)
		}
	}

	// Test that Document is loaded
	if content.Document == nil {
		t.Error("Document is nil, want non-nil")
	}

	// Test that body content is accessible
	if content.Document != nil {
		bodyText := content.Document.Find("body").Text()
		if bodyText == "" {
			t.Error("Body text is empty")
		}
	}
}

func TestLoadContent_WithAnchorLinks(t *testing.T) {
	xhtmlContent := `<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE html>
<html xmlns="http://www.w3.org/1999/xhtml">
<head>
	<title>Chapter with Links</title>
</head>
<body>
	<p>See <a href="chapter2.xhtml">next chapter</a> for more.</p>
	<p>Visit <a href="http://example.com">external site</a>.</p>
	<img src="../images/logo.png" alt="Logo"/>
</body>
</html>`

	content, err := LoadContent("ch1", "text/chapter1.xhtml", []byte(xhtmlContent))
	if err != nil {
		t.Fatalf("LoadContent failed: %v", err)
	}

	// Should have one image
	if len(content.ImageRefs) != 1 {
		t.Errorf("ImageRefs count = %d, want 1", len(content.ImageRefs))
	}

	if content.ImageRefs[0] != "images/logo.png" {
		t.Errorf("ImageRefs[0] = %q, want %q", content.ImageRefs[0], "images/logo.png")
	}
}

func TestLoadContent_NestedPaths(t *testing.T) {
	// Test file in deeply nested directory
	xhtmlContent := `<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE html>
<html xmlns="http://www.w3.org/1999/xhtml">
<head>
	<link rel="stylesheet" href="../../styles/main.css"/>
</head>
<body>
	<img src="../../images/cover.jpg" alt="Cover"/>
	<img src="../sibling/image.png" alt="Sibling"/>
</body>
</html>`

	content, err := LoadContent("nested", "content/chapters/ch1/page.xhtml", []byte(xhtmlContent))
	if err != nil {
		t.Fatalf("LoadContent failed: %v", err)
	}

	// Test CSS path resolution
	expectedCSS := []string{"styles/main.css"}
	if len(content.CSSLinks) != len(expectedCSS) {
		t.Fatalf("CSSLinks count = %d, want %d", len(content.CSSLinks), len(expectedCSS))
	}
	if content.CSSLinks[0] != expectedCSS[0] {
		t.Errorf("CSSLinks[0] = %q, want %q", content.CSSLinks[0], expectedCSS[0])
	}

	// Test image path resolution
	expectedImages := []string{"images/cover.jpg", "content/chapters/sibling/image.png"}
	if len(content.ImageRefs) != len(expectedImages) {
		t.Fatalf("ImageRefs count = %d, want %d", len(content.ImageRefs), len(expectedImages))
	}

	for i, expected := range expectedImages {
		if content.ImageRefs[i] != expected {
			t.Errorf("ImageRefs[%d] = %q, want %q", i, content.ImageRefs[i], expected)
		}
	}
}

func TestLoadContent_RootLevelFile(t *testing.T) {
	// Test file at root level
	xhtmlContent := `<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE html>
<html xmlns="http://www.w3.org/1999/xhtml">
<head>
	<link rel="stylesheet" href="style.css"/>
</head>
<body>
	<img src="image.jpg" alt="Image"/>
</body>
</html>`

	content, err := LoadContent("root", "index.xhtml", []byte(xhtmlContent))
	if err != nil {
		t.Fatalf("LoadContent failed: %v", err)
	}

	// CSS should be at root
	if len(content.CSSLinks) != 1 || content.CSSLinks[0] != "style.css" {
		t.Errorf("CSSLinks = %v, want [style.css]", content.CSSLinks)
	}

	// Image should be at root
	if len(content.ImageRefs) != 1 || content.ImageRefs[0] != "image.jpg" {
		t.Errorf("ImageRefs = %v, want [image.jpg]", content.ImageRefs)
	}
}

func TestLoadContent_InvalidXML(t *testing.T) {
	invalidContent := `<html><body><p>Unclosed paragraph</body></html>`

	_, err := LoadContent("invalid", "test.xhtml", []byte(invalidContent))
	if err == nil {
		t.Error("LoadContent should fail with invalid XML, but succeeded")
	}
}

func TestLoadContent_NoReferences(t *testing.T) {
	// XHTML with no CSS or image references
	xhtmlContent := `<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE html>
<html xmlns="http://www.w3.org/1999/xhtml">
<head>
	<title>Simple Chapter</title>
</head>
<body>
	<h1>Title</h1>
	<p>Just text content.</p>
</body>
</html>`

	content, err := LoadContent("simple", "chapter.xhtml", []byte(xhtmlContent))
	if err != nil {
		t.Fatalf("LoadContent failed: %v", err)
	}

	if len(content.CSSLinks) != 0 {
		t.Errorf("CSSLinks count = %d, want 0", len(content.CSSLinks))
	}

	if len(content.ImageRefs) != 0 {
		t.Errorf("ImageRefs count = %d, want 0", len(content.ImageRefs))
	}
}

func TestLoadContent_MultipleStylesheets(t *testing.T) {
	xhtmlContent := `<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE html>
<html xmlns="http://www.w3.org/1999/xhtml">
<head>
	<link rel="stylesheet" href="css/reset.css"/>
	<link rel="stylesheet" href="css/main.css"/>
	<link rel="stylesheet" href="css/theme.css"/>
	<link rel="icon" href="favicon.ico"/>
</head>
<body>
	<p>Content</p>
</body>
</html>`

	content, err := LoadContent("multi", "text/page.xhtml", []byte(xhtmlContent))
	if err != nil {
		t.Fatalf("LoadContent failed: %v", err)
	}

	// Should have 3 CSS files (icon link should be ignored)
	if len(content.CSSLinks) != 3 {
		t.Errorf("CSSLinks count = %d, want 3", len(content.CSSLinks))
	}
}

func TestLoadContent_ComplexHTML(t *testing.T) {
	// Test with various HTML elements
	xhtmlContent := `<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE html>
<html xmlns="http://www.w3.org/1999/xhtml">
<head>
	<title>Complex Page</title>
	<link rel="stylesheet" href="../styles/book.css"/>
</head>
<body>
	<article>
		<header>
			<h1>Main Title</h1>
		</header>
		<section>
			<h2>Section Title</h2>
			<p>Text with <strong>bold</strong> and <em>italic</em>.</p>
			<figure>
				<img src="../images/figure1.jpg" alt="Figure 1"/>
				<figcaption>Figure caption</figcaption>
			</figure>
			<ul>
				<li>Item 1</li>
				<li>Item 2</li>
			</ul>
			<table>
				<thead>
					<tr><th>Header</th></tr>
				</thead>
				<tbody>
					<tr><td>Data</td></tr>
				</tbody>
			</table>
		</section>
		<footer>
			<p>Footer content</p>
		</footer>
	</article>
</body>
</html>`

	content, err := LoadContent("complex", "text/complex.xhtml", []byte(xhtmlContent))
	if err != nil {
		t.Fatalf("LoadContent failed: %v", err)
	}

	// Should find CSS
	if len(content.CSSLinks) != 1 {
		t.Errorf("CSSLinks count = %d, want 1", len(content.CSSLinks))
	}

	// Should find image
	if len(content.ImageRefs) != 1 {
		t.Errorf("ImageRefs count = %d, want 1", len(content.ImageRefs))
	}

	if content.ImageRefs[0] != "images/figure1.jpg" {
		t.Errorf("ImageRefs[0] = %q, want %q", content.ImageRefs[0], "images/figure1.jpg")
	}

	// Document should contain expected elements
	if content.Document != nil {
		h1Text := content.Document.Find("h1").Text()
		if h1Text != "Main Title" {
			t.Errorf("h1 text = %q, want %q", h1Text, "Main Title")
		}

		figcaption := content.Document.Find("figcaption").Text()
		if figcaption != "Figure caption" {
			t.Errorf("figcaption text = %q, want %q", figcaption, "Figure caption")
		}
	}
}
