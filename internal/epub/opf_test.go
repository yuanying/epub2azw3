package epub

import (
	"strings"
	"testing"
)

func TestParseOPF_EPUB20(t *testing.T) {
	// EPUB 2.0 format OPF content
	opfContent := `<?xml version="1.0" encoding="UTF-8"?>
<package version="2.0" xmlns="http://www.idpf.org/2007/opf" unique-identifier="bookid">
  <metadata xmlns:dc="http://purl.org/dc/elements/1.1/" xmlns:opf="http://www.idpf.org/2007/opf">
    <dc:title>Sample Book Title</dc:title>
    <dc:creator opf:role="aut">John Doe</dc:creator>
    <dc:creator opf:role="edt">Jane Editor</dc:creator>
    <dc:language>en</dc:language>
    <dc:identifier id="bookid">urn:isbn:1234567890</dc:identifier>
    <dc:publisher>Test Publisher</dc:publisher>
    <dc:date>2024-01-01</dc:date>
    <dc:description>This is a sample book description.</dc:description>
    <dc:subject>Fiction</dc:subject>
    <dc:subject>Adventure</dc:subject>
    <dc:rights>Copyright 2024</dc:rights>
    <meta name="cover" content="cover-image"/>
  </metadata>
  <manifest>
    <item id="ncx" href="toc.ncx" media-type="application/x-dtbncx+xml"/>
    <item id="cover-image" href="images/cover.jpg" media-type="image/jpeg"/>
    <item id="chapter1" href="text/chapter1.xhtml" media-type="application/xhtml+xml"/>
    <item id="chapter2" href="text/chapter2.xhtml" media-type="application/xhtml+xml"/>
    <item id="stylesheet" href="css/style.css" media-type="text/css"/>
  </manifest>
  <spine toc="ncx">
    <itemref idref="chapter1"/>
    <itemref idref="chapter2" linear="no"/>
  </spine>
</package>`

	opf, err := ParseOPF([]byte(opfContent), "OEBPS/")
	if err != nil {
		t.Fatalf("ParseOPF failed: %v", err)
	}

	// Test metadata
	if opf.Metadata.Title != "Sample Book Title" {
		t.Errorf("Title = %q, want %q", opf.Metadata.Title, "Sample Book Title")
	}

	if len(opf.Metadata.Creators) != 2 {
		t.Fatalf("Creators count = %d, want 2", len(opf.Metadata.Creators))
	}

	if opf.Metadata.Creators[0].Name != "John Doe" {
		t.Errorf("Creator[0].Name = %q, want %q", opf.Metadata.Creators[0].Name, "John Doe")
	}

	if opf.Metadata.Creators[0].Role != "aut" {
		t.Errorf("Creator[0].Role = %q, want %q", opf.Metadata.Creators[0].Role, "aut")
	}

	if opf.Metadata.Creators[1].Name != "Jane Editor" {
		t.Errorf("Creator[1].Name = %q, want %q", opf.Metadata.Creators[1].Name, "Jane Editor")
	}

	if opf.Metadata.Creators[1].Role != "edt" {
		t.Errorf("Creator[1].Role = %q, want %q", opf.Metadata.Creators[1].Role, "edt")
	}

	if opf.Metadata.Language != "en" {
		t.Errorf("Language = %q, want %q", opf.Metadata.Language, "en")
	}

	if opf.Metadata.Identifier != "urn:isbn:1234567890" {
		t.Errorf("Identifier = %q, want %q", opf.Metadata.Identifier, "urn:isbn:1234567890")
	}

	if opf.Metadata.Publisher != "Test Publisher" {
		t.Errorf("Publisher = %q, want %q", opf.Metadata.Publisher, "Test Publisher")
	}

	if opf.Metadata.Date != "2024-01-01" {
		t.Errorf("Date = %q, want %q", opf.Metadata.Date, "2024-01-01")
	}

	if opf.Metadata.Description != "This is a sample book description." {
		t.Errorf("Description = %q, want %q", opf.Metadata.Description, "This is a sample book description.")
	}

	if len(opf.Metadata.Subjects) != 2 {
		t.Fatalf("Subjects count = %d, want 2", len(opf.Metadata.Subjects))
	}

	if opf.Metadata.Subjects[0] != "Fiction" {
		t.Errorf("Subjects[0] = %q, want %q", opf.Metadata.Subjects[0], "Fiction")
	}

	if opf.Metadata.Subjects[1] != "Adventure" {
		t.Errorf("Subjects[1] = %q, want %q", opf.Metadata.Subjects[1], "Adventure")
	}

	if opf.Metadata.Rights != "Copyright 2024" {
		t.Errorf("Rights = %q, want %q", opf.Metadata.Rights, "Copyright 2024")
	}

	// Test manifest
	if len(opf.Manifest) != 5 {
		t.Fatalf("Manifest count = %d, want 5", len(opf.Manifest))
	}

	coverItem, ok := opf.Manifest["cover-image"]
	if !ok {
		t.Fatal("cover-image not found in manifest")
	}

	if coverItem.Href != "OEBPS/images/cover.jpg" {
		t.Errorf("cover-image.Href = %q, want %q", coverItem.Href, "OEBPS/images/cover.jpg")
	}

	if coverItem.MediaType != "image/jpeg" {
		t.Errorf("cover-image.MediaType = %q, want %q", coverItem.MediaType, "image/jpeg")
	}

	chapter1, ok := opf.Manifest["chapter1"]
	if !ok {
		t.Fatal("chapter1 not found in manifest")
	}

	if chapter1.Href != "OEBPS/text/chapter1.xhtml" {
		t.Errorf("chapter1.Href = %q, want %q", chapter1.Href, "OEBPS/text/chapter1.xhtml")
	}

	// Test spine
	if len(opf.Spine) != 2 {
		t.Fatalf("Spine count = %d, want 2", len(opf.Spine))
	}

	if opf.Spine[0].IDRef != "chapter1" {
		t.Errorf("Spine[0].IDRef = %q, want %q", opf.Spine[0].IDRef, "chapter1")
	}

	if !opf.Spine[0].Linear {
		t.Errorf("Spine[0].Linear = false, want true")
	}

	if opf.Spine[1].IDRef != "chapter2" {
		t.Errorf("Spine[1].IDRef = %q, want %q", opf.Spine[1].IDRef, "chapter2")
	}

	if opf.Spine[1].Linear {
		t.Errorf("Spine[1].Linear = true, want false")
	}

	// Test NCX path
	if opf.NCXPath != "OEBPS/toc.ncx" {
		t.Errorf("NCXPath = %q, want %q", opf.NCXPath, "OEBPS/toc.ncx")
	}

	// Test ManifestOrder preserves document order
	if len(opf.ManifestOrder) != 5 {
		t.Fatalf("ManifestOrder count = %d, want 5", len(opf.ManifestOrder))
	}
	expectedOrder := []string{"ncx", "cover-image", "chapter1", "chapter2", "stylesheet"}
	for i, id := range expectedOrder {
		if opf.ManifestOrder[i] != id {
			t.Errorf("ManifestOrder[%d] = %q, want %q", i, opf.ManifestOrder[i], id)
		}
	}
}

func TestParseOPF_EPUB30(t *testing.T) {
	// EPUB 3.0 format OPF content with properties
	opfContent := `<?xml version="1.0" encoding="UTF-8"?>
<package version="3.0" xmlns="http://www.idpf.org/2007/opf" unique-identifier="bookid">
  <metadata xmlns:dc="http://purl.org/dc/elements/1.1/">
    <dc:title>EPUB 3.0 Sample</dc:title>
    <dc:creator id="creator01">Author Name</dc:creator>
    <meta refines="#creator01" property="role" scheme="marc:relators">aut</meta>
    <dc:language>ja</dc:language>
    <dc:identifier id="bookid">urn:uuid:12345678-1234-1234-1234-123456789012</dc:identifier>
    <meta property="dcterms:modified">2024-01-15T12:00:00Z</meta>
  </metadata>
  <manifest>
    <item id="nav" href="nav.xhtml" media-type="application/xhtml+xml" properties="nav"/>
    <item id="cover" href="images/cover.png" media-type="image/png" properties="cover-image"/>
    <item id="ch1" href="text/ch1.xhtml" media-type="application/xhtml+xml"/>
  </manifest>
  <spine>
    <itemref idref="ch1"/>
  </spine>
</package>`

	opf, err := ParseOPF([]byte(opfContent), "content/")
	if err != nil {
		t.Fatalf("ParseOPF failed: %v", err)
	}

	// Test metadata
	if opf.Metadata.Title != "EPUB 3.0 Sample" {
		t.Errorf("Title = %q, want %q", opf.Metadata.Title, "EPUB 3.0 Sample")
	}

	if opf.Metadata.Language != "ja" {
		t.Errorf("Language = %q, want %q", opf.Metadata.Language, "ja")
	}

	// Test manifest properties
	navItem, ok := opf.Manifest["nav"]
	if !ok {
		t.Fatal("nav not found in manifest")
	}

	if len(navItem.Properties) != 1 || navItem.Properties[0] != "nav" {
		t.Errorf("nav.Properties = %v, want [nav]", navItem.Properties)
	}

	coverItem, ok := opf.Manifest["cover"]
	if !ok {
		t.Fatal("cover not found in manifest")
	}

	if len(coverItem.Properties) != 1 || coverItem.Properties[0] != "cover-image" {
		t.Errorf("cover.Properties = %v, want [cover-image]", coverItem.Properties)
	}

	if coverItem.Href != "content/images/cover.png" {
		t.Errorf("cover.Href = %q, want %q", coverItem.Href, "content/images/cover.png")
	}
}

func TestParseOPF_MinimalRequired(t *testing.T) {
	// Minimal OPF with only required elements
	opfContent := `<?xml version="1.0" encoding="UTF-8"?>
<package version="2.0" xmlns="http://www.idpf.org/2007/opf" unique-identifier="uid">
  <metadata xmlns:dc="http://purl.org/dc/elements/1.1/">
    <dc:title>Minimal Book</dc:title>
    <dc:language>en</dc:language>
    <dc:identifier id="uid">minimal-001</dc:identifier>
  </metadata>
  <manifest>
    <item id="chapter" href="chapter.xhtml" media-type="application/xhtml+xml"/>
  </manifest>
  <spine>
    <itemref idref="chapter"/>
  </spine>
</package>`

	opf, err := ParseOPF([]byte(opfContent), "")
	if err != nil {
		t.Fatalf("ParseOPF failed: %v", err)
	}

	if opf.Metadata.Title != "Minimal Book" {
		t.Errorf("Title = %q, want %q", opf.Metadata.Title, "Minimal Book")
	}

	if opf.Metadata.Language != "en" {
		t.Errorf("Language = %q, want %q", opf.Metadata.Language, "en")
	}

	if opf.Metadata.Identifier != "minimal-001" {
		t.Errorf("Identifier = %q, want %q", opf.Metadata.Identifier, "minimal-001")
	}

	if len(opf.Manifest) != 1 {
		t.Errorf("Manifest count = %d, want 1", len(opf.Manifest))
	}

	if len(opf.Spine) != 1 {
		t.Errorf("Spine count = %d, want 1", len(opf.Spine))
	}
}

func TestParseOPF_PageProgressionDirection(t *testing.T) {
	tests := []struct {
		name      string
		spineAttr string
		want      string
	}{
		{
			name:      "rtl",
			spineAttr: ` page-progression-direction="rtl"`,
			want:      "rtl",
		},
		{
			name:      "ltr",
			spineAttr: ` page-progression-direction="ltr"`,
			want:      "ltr",
		},
		{
			name:      "no attribute",
			spineAttr: "",
			want:      "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			opfContent := `<?xml version="1.0" encoding="UTF-8"?>
<package version="3.0" xmlns="http://www.idpf.org/2007/opf" unique-identifier="uid">
  <metadata xmlns:dc="http://purl.org/dc/elements/1.1/">
    <dc:title>Test</dc:title>
    <dc:language>ja</dc:language>
    <dc:identifier id="uid">test-001</dc:identifier>
  </metadata>
  <manifest>
    <item id="ch1" href="ch1.xhtml" media-type="application/xhtml+xml"/>
  </manifest>
  <spine` + tt.spineAttr + `>
    <itemref idref="ch1"/>
  </spine>
</package>`

			opf, err := ParseOPF([]byte(opfContent), "")
			if err != nil {
				t.Fatalf("ParseOPF failed: %v", err)
			}

			if opf.PageProgressionDirection != tt.want {
				t.Errorf("PageProgressionDirection = %q, want %q", opf.PageProgressionDirection, tt.want)
			}
		})
	}
}

func TestParseOPF_IdentifierPrefersSchemeISBN(t *testing.T) {
	opfContent := `<?xml version="1.0" encoding="UTF-8"?>
<package version="2.0" xmlns="http://www.idpf.org/2007/opf" unique-identifier="uid">
  <metadata xmlns:dc="http://purl.org/dc/elements/1.1/" xmlns:opf="http://www.idpf.org/2007/opf">
    <dc:title>Identifier Priority</dc:title>
    <dc:language>en</dc:language>
    <dc:identifier id="uid">urn:uuid:12345678-1234-1234-1234-123456789012</dc:identifier>
    <dc:identifier opf:scheme="ISBN">978-4-12345678-0</dc:identifier>
  </metadata>
  <manifest>
    <item id="chapter" href="chapter.xhtml" media-type="application/xhtml+xml"/>
  </manifest>
  <spine>
    <itemref idref="chapter"/>
  </spine>
</package>`

	opf, err := ParseOPF([]byte(opfContent), "")
	if err != nil {
		t.Fatalf("ParseOPF failed: %v", err)
	}

	if opf.Metadata.Identifier != "978-4-12345678-0" {
		t.Errorf("Identifier = %q, want %q", opf.Metadata.Identifier, "978-4-12345678-0")
	}
}

func TestParseOPF_IdentifierPrefersISBNPattern(t *testing.T) {
	opfContent := `<?xml version="1.0" encoding="UTF-8"?>
<package version="3.0" xmlns="http://www.idpf.org/2007/opf" unique-identifier="uid">
  <metadata xmlns:dc="http://purl.org/dc/elements/1.1/">
    <dc:title>Identifier Pattern</dc:title>
    <dc:language>ja</dc:language>
    <dc:identifier id="uid">urn:uuid:aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee</dc:identifier>
    <dc:identifier>urn:isbn:9784123456780</dc:identifier>
  </metadata>
  <manifest>
    <item id="chapter" href="chapter.xhtml" media-type="application/xhtml+xml"/>
  </manifest>
  <spine>
    <itemref idref="chapter"/>
  </spine>
</package>`

	opf, err := ParseOPF([]byte(opfContent), "")
	if err != nil {
		t.Fatalf("ParseOPF failed: %v", err)
	}

	if opf.Metadata.Identifier != "urn:isbn:9784123456780" {
		t.Errorf("Identifier = %q, want %q", opf.Metadata.Identifier, "urn:isbn:9784123456780")
	}
}

func TestParseOPF_IdentifierFallsBackToUniqueIDWithoutISBN(t *testing.T) {
	opfContent := `<?xml version="1.0" encoding="UTF-8"?>
<package version="2.0" xmlns="http://www.idpf.org/2007/opf" unique-identifier="uid">
  <metadata xmlns:dc="http://purl.org/dc/elements/1.1/">
    <dc:title>No ISBN</dc:title>
    <dc:language>en</dc:language>
    <dc:identifier>non-unique-first</dc:identifier>
    <dc:identifier id="uid">unique-value</dc:identifier>
  </metadata>
  <manifest>
    <item id="chapter" href="chapter.xhtml" media-type="application/xhtml+xml"/>
  </manifest>
  <spine>
    <itemref idref="chapter"/>
  </spine>
</package>`

	opf, err := ParseOPF([]byte(opfContent), "")
	if err != nil {
		t.Fatalf("ParseOPF failed: %v", err)
	}

	if opf.Metadata.Identifier != "unique-value" {
		t.Errorf("Identifier = %q, want %q", opf.Metadata.Identifier, "unique-value")
	}
}

func TestParseOPF_GuideReferences(t *testing.T) {
	opfContent := `<?xml version="1.0" encoding="UTF-8"?>
<package version="2.0" xmlns="http://www.idpf.org/2007/opf" unique-identifier="uid">
  <metadata xmlns:dc="http://purl.org/dc/elements/1.1/">
    <dc:title>Guide Test</dc:title>
    <dc:language>en</dc:language>
    <dc:identifier id="uid">guide-001</dc:identifier>
  </metadata>
  <manifest>
    <item id="cover-page" href="text/cover.xhtml" media-type="application/xhtml+xml"/>
  </manifest>
  <spine>
    <itemref idref="cover-page"/>
  </spine>
  <guide>
    <reference type="cover" title="Cover" href="text/cover.xhtml#top"/>
    <reference type="toc" title="Table of Contents" href="toc.xhtml"/>
  </guide>
</package>`

	opf, err := ParseOPF([]byte(opfContent), "OEBPS")
	if err != nil {
		t.Fatalf("ParseOPF failed: %v", err)
	}

	if len(opf.Guide) != 2 {
		t.Fatalf("Guide count = %d, want 2", len(opf.Guide))
	}

	if opf.Guide[0].Type != "cover" {
		t.Errorf("Guide[0].Type = %q, want %q", opf.Guide[0].Type, "cover")
	}
	if opf.Guide[0].Href != "OEBPS/text/cover.xhtml#top" {
		t.Errorf("Guide[0].Href = %q, want %q", opf.Guide[0].Href, "OEBPS/text/cover.xhtml#top")
	}
	if opf.Guide[1].Href != "OEBPS/toc.xhtml" {
		t.Errorf("Guide[1].Href = %q, want %q", opf.Guide[1].Href, "OEBPS/toc.xhtml")
	}
}

func TestJoinPath_SlashNormalization(t *testing.T) {
	tests := []struct {
		name string
		base string
		rel  string
		want string
	}{
		{
			name: "basic join with trailing slash",
			base: "OEBPS/",
			rel:  "text/chapter1.xhtml",
			want: "OEBPS/text/chapter1.xhtml",
		},
		{
			name: "empty base returns rel as-is",
			base: "",
			rel:  "chapter.xhtml",
			want: "chapter.xhtml",
		},
		{
			name: "base without trailing slash",
			base: "OEBPS",
			rel:  "images/cover.jpg",
			want: "OEBPS/images/cover.jpg",
		},
		{
			name: "result uses forward slashes",
			base: "content",
			rel:  "text/ch1.xhtml",
			want: "content/text/ch1.xhtml",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := joinPath(tt.base, tt.rel)
			if got != tt.want {
				t.Errorf("joinPath(%q, %q) = %q, want %q", tt.base, tt.rel, got, tt.want)
			}
			// Verify no backslashes in result (forward slash normalization)
			if strings.Contains(got, "\\") {
				t.Errorf("joinPath(%q, %q) = %q contains backslash", tt.base, tt.rel, got)
			}
		})
	}
}

func TestFindCoverImage(t *testing.T) {
	tests := []struct {
		name     string
		opf      *OPF
		wantHref string
		wantOK   bool
	}{
		{
			name: "cover-image property in EPUB 3.0",
			opf: &OPF{
				Manifest: map[string]ManifestItem{
					"cover": {
						ID:         "cover",
						Href:       "images/cover.jpg",
						MediaType:  "image/jpeg",
						Properties: []string{"cover-image"},
					},
				},
			},
			wantHref: "images/cover.jpg",
			wantOK:   true,
		},
		{
			name: "cover via meta name in EPUB 2.0",
			opf: &OPF{
				Metadata: Metadata{
					CoverID: "cover-image",
				},
				Manifest: map[string]ManifestItem{
					"cover-image": {
						ID:        "cover-image",
						Href:      "OEBPS/images/cover.jpg",
						MediaType: "image/jpeg",
					},
				},
			},
			wantHref: "OEBPS/images/cover.jpg",
			wantOK:   true,
		},
		{
			name: "no cover image",
			opf: &OPF{
				Manifest: map[string]ManifestItem{
					"chapter": {
						ID:        "chapter",
						Href:      "chapter.xhtml",
						MediaType: "application/xhtml+xml",
					},
				},
			},
			wantHref: "",
			wantOK:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			href, ok := tt.opf.FindCoverImage()
			if ok != tt.wantOK {
				t.Errorf("FindCoverImage() ok = %v, want %v", ok, tt.wantOK)
			}
			if href != tt.wantHref {
				t.Errorf("FindCoverImage() href = %v, want %v", href, tt.wantHref)
			}
		})
	}
}
