package epub

import (
	"archive/zip"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"strconv"
	"testing"
)

func TestSplitFragment(t *testing.T) {
	tests := []struct {
		name         string
		src          string
		wantPath     string
		wantFragment string
	}{
		{
			name:         "path with fragment",
			src:          "chapter1.xhtml#sec1",
			wantPath:     "chapter1.xhtml",
			wantFragment: "sec1",
		},
		{
			name:         "path without fragment",
			src:          "chapter1.xhtml",
			wantPath:     "chapter1.xhtml",
			wantFragment: "",
		},
		{
			name:         "fragment only",
			src:          "#sec1",
			wantPath:     "",
			wantFragment: "sec1",
		},
		{
			name:         "empty string",
			src:          "",
			wantPath:     "",
			wantFragment: "",
		},
		{
			name:         "multiple hash signs",
			src:          "chapter1.xhtml#sec1#subsec2",
			wantPath:     "chapter1.xhtml",
			wantFragment: "sec1#subsec2",
		},
		{
			name:         "path with directory",
			src:          "text/chapter1.xhtml#anchor",
			wantPath:     "text/chapter1.xhtml",
			wantFragment: "anchor",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotPath, gotFragment := splitFragment(tt.src)
			if gotPath != tt.wantPath {
				t.Errorf("splitFragment(%q) path = %q, want %q", tt.src, gotPath, tt.wantPath)
			}
			if gotFragment != tt.wantFragment {
				t.Errorf("splitFragment(%q) fragment = %q, want %q", tt.src, gotFragment, tt.wantFragment)
			}
		})
	}
}

func TestParseNCX_FlatNavPoints(t *testing.T) {
	ncxXML := []byte(`<?xml version="1.0" encoding="UTF-8"?>
<ncx xmlns="http://www.daisy.org/z3986/2005/ncx/" version="2005-1">
  <head>
    <meta name="dtb:uid" content="test-uid-123"/>
    <meta name="dtb:depth" content="1"/>
  </head>
  <docTitle><text>Test Book</text></docTitle>
  <navMap>
    <navPoint id="np1" playOrder="1">
      <navLabel><text>Chapter 1</text></navLabel>
      <content src="chapter1.xhtml"/>
    </navPoint>
    <navPoint id="np2" playOrder="2">
      <navLabel><text>Chapter 2</text></navLabel>
      <content src="chapter2.xhtml"/>
    </navPoint>
    <navPoint id="np3" playOrder="3">
      <navLabel><text>Chapter 3</text></navLabel>
      <content src="chapter3.xhtml"/>
    </navPoint>
  </navMap>
</ncx>`)

	ncx, err := parseNCX(ncxXML, "OEBPS")
	if err != nil {
		t.Fatalf("parseNCX() error = %v", err)
	}

	if len(ncx.NavPoints) != 3 {
		t.Fatalf("got %d nav points, want 3", len(ncx.NavPoints))
	}

	want := []NavPoint{
		{ID: "np1", PlayOrder: 1, Label: "Chapter 1", ContentPath: "OEBPS/chapter1.xhtml"},
		{ID: "np2", PlayOrder: 2, Label: "Chapter 2", ContentPath: "OEBPS/chapter2.xhtml"},
		{ID: "np3", PlayOrder: 3, Label: "Chapter 3", ContentPath: "OEBPS/chapter3.xhtml"},
	}

	for i, np := range ncx.NavPoints {
		if np.ID != want[i].ID {
			t.Errorf("NavPoints[%d].ID = %q, want %q", i, np.ID, want[i].ID)
		}
		if np.PlayOrder != want[i].PlayOrder {
			t.Errorf("NavPoints[%d].PlayOrder = %d, want %d", i, np.PlayOrder, want[i].PlayOrder)
		}
		if np.Label != want[i].Label {
			t.Errorf("NavPoints[%d].Label = %q, want %q", i, np.Label, want[i].Label)
		}
		if np.ContentPath != want[i].ContentPath {
			t.Errorf("NavPoints[%d].ContentPath = %q, want %q", i, np.ContentPath, want[i].ContentPath)
		}
	}
}

func TestParseNCX_NestedNavPoints(t *testing.T) {
	ncxXML := []byte(`<?xml version="1.0" encoding="UTF-8"?>
<ncx xmlns="http://www.daisy.org/z3986/2005/ncx/" version="2005-1">
  <head>
    <meta name="dtb:uid" content="nested-uid"/>
    <meta name="dtb:depth" content="3"/>
  </head>
  <docTitle><text>Nested Book</text></docTitle>
  <navMap>
    <navPoint id="np1" playOrder="1">
      <navLabel><text>Part 1</text></navLabel>
      <content src="part1.xhtml"/>
      <navPoint id="np2" playOrder="2">
        <navLabel><text>Chapter 1.1</text></navLabel>
        <content src="ch1_1.xhtml"/>
        <navPoint id="np3" playOrder="3">
          <navLabel><text>Section 1.1.1</text></navLabel>
          <content src="ch1_1.xhtml#sec1"/>
        </navPoint>
      </navPoint>
      <navPoint id="np4" playOrder="4">
        <navLabel><text>Chapter 1.2</text></navLabel>
        <content src="ch1_2.xhtml"/>
      </navPoint>
    </navPoint>
    <navPoint id="np5" playOrder="5">
      <navLabel><text>Part 2</text></navLabel>
      <content src="part2.xhtml"/>
    </navPoint>
  </navMap>
</ncx>`)

	ncx, err := parseNCX(ncxXML, "OEBPS")
	if err != nil {
		t.Fatalf("parseNCX() error = %v", err)
	}

	if len(ncx.NavPoints) != 2 {
		t.Fatalf("got %d top-level nav points, want 2", len(ncx.NavPoints))
	}

	// Part 1
	p1 := ncx.NavPoints[0]
	if p1.Label != "Part 1" {
		t.Errorf("NavPoints[0].Label = %q, want %q", p1.Label, "Part 1")
	}
	if len(p1.Children) != 2 {
		t.Fatalf("NavPoints[0].Children = %d, want 2", len(p1.Children))
	}

	// Chapter 1.1
	ch11 := p1.Children[0]
	if ch11.Label != "Chapter 1.1" {
		t.Errorf("Children[0].Label = %q, want %q", ch11.Label, "Chapter 1.1")
	}
	if len(ch11.Children) != 1 {
		t.Fatalf("Children[0].Children = %d, want 1", len(ch11.Children))
	}

	// Section 1.1.1 (3rd level with fragment)
	sec := ch11.Children[0]
	if sec.Label != "Section 1.1.1" {
		t.Errorf("Section label = %q, want %q", sec.Label, "Section 1.1.1")
	}
	if sec.ContentPath != "OEBPS/ch1_1.xhtml" {
		t.Errorf("Section ContentPath = %q, want %q", sec.ContentPath, "OEBPS/ch1_1.xhtml")
	}
	if sec.Fragment != "sec1" {
		t.Errorf("Section Fragment = %q, want %q", sec.Fragment, "sec1")
	}

	// Part 2
	p2 := ncx.NavPoints[1]
	if p2.Label != "Part 2" {
		t.Errorf("NavPoints[1].Label = %q, want %q", p2.Label, "Part 2")
	}
	if len(p2.Children) != 0 {
		t.Errorf("NavPoints[1].Children = %d, want 0", len(p2.Children))
	}
}

func TestParseNCX_HeadMeta(t *testing.T) {
	ncxXML := []byte(`<?xml version="1.0" encoding="UTF-8"?>
<ncx xmlns="http://www.daisy.org/z3986/2005/ncx/" version="2005-1">
  <head>
    <meta name="dtb:uid" content="unique-id-42"/>
    <meta name="dtb:depth" content="3"/>
    <meta name="dtb:totalPageCount" content="0"/>
    <meta name="dtb:maxPageNumber" content="0"/>
  </head>
  <docTitle><text>Meta Test Book</text></docTitle>
  <navMap/>
</ncx>`)

	ncx, err := parseNCX(ncxXML, "")
	if err != nil {
		t.Fatalf("parseNCX() error = %v", err)
	}

	if ncx.UID != "unique-id-42" {
		t.Errorf("UID = %q, want %q", ncx.UID, "unique-id-42")
	}
	if ncx.Depth != 3 {
		t.Errorf("Depth = %d, want %d", ncx.Depth, 3)
	}
	if ncx.DocTitle != "Meta Test Book" {
		t.Errorf("DocTitle = %q, want %q", ncx.DocTitle, "Meta Test Book")
	}
}

func TestParseNCX_PathNormalization(t *testing.T) {
	ncxXML := []byte(`<?xml version="1.0" encoding="UTF-8"?>
<ncx xmlns="http://www.daisy.org/z3986/2005/ncx/" version="2005-1">
  <head/>
  <navMap>
    <navPoint id="np1" playOrder="1">
      <navLabel><text>Chapter 1</text></navLabel>
      <content src="../text/chapter1.xhtml"/>
    </navPoint>
  </navMap>
</ncx>`)

	// NCX is in OEBPS/toc directory, content references ../text/chapter1.xhtml
	ncx, err := parseNCX(ncxXML, "OEBPS/toc")
	if err != nil {
		t.Fatalf("parseNCX() error = %v", err)
	}

	if len(ncx.NavPoints) != 1 {
		t.Fatalf("got %d nav points, want 1", len(ncx.NavPoints))
	}

	want := "OEBPS/text/chapter1.xhtml"
	if ncx.NavPoints[0].ContentPath != want {
		t.Errorf("ContentPath = %q, want %q", ncx.NavPoints[0].ContentPath, want)
	}
}

func TestParseNCX_FragmentSeparation(t *testing.T) {
	ncxXML := []byte(`<?xml version="1.0" encoding="UTF-8"?>
<ncx xmlns="http://www.daisy.org/z3986/2005/ncx/" version="2005-1">
  <head/>
  <navMap>
    <navPoint id="np1" playOrder="1">
      <navLabel><text>Section A</text></navLabel>
      <content src="chapter1.xhtml#sectionA"/>
    </navPoint>
    <navPoint id="np2" playOrder="2">
      <navLabel><text>No Fragment</text></navLabel>
      <content src="chapter2.xhtml"/>
    </navPoint>
  </navMap>
</ncx>`)

	ncx, err := parseNCX(ncxXML, "OEBPS")
	if err != nil {
		t.Fatalf("parseNCX() error = %v", err)
	}

	if ncx.NavPoints[0].ContentPath != "OEBPS/chapter1.xhtml" {
		t.Errorf("NavPoints[0].ContentPath = %q, want %q", ncx.NavPoints[0].ContentPath, "OEBPS/chapter1.xhtml")
	}
	if ncx.NavPoints[0].Fragment != "sectionA" {
		t.Errorf("NavPoints[0].Fragment = %q, want %q", ncx.NavPoints[0].Fragment, "sectionA")
	}

	if ncx.NavPoints[1].ContentPath != "OEBPS/chapter2.xhtml" {
		t.Errorf("NavPoints[1].ContentPath = %q, want %q", ncx.NavPoints[1].ContentPath, "OEBPS/chapter2.xhtml")
	}
	if ncx.NavPoints[1].Fragment != "" {
		t.Errorf("NavPoints[1].Fragment = %q, want empty", ncx.NavPoints[1].Fragment)
	}
}

func TestParseNCX_Empty(t *testing.T) {
	ncxXML := []byte(`<?xml version="1.0" encoding="UTF-8"?>
<ncx xmlns="http://www.daisy.org/z3986/2005/ncx/" version="2005-1">
  <head>
    <meta name="dtb:uid" content="empty-uid"/>
    <meta name="dtb:depth" content="0"/>
  </head>
  <docTitle><text>Empty Book</text></docTitle>
  <navMap/>
</ncx>`)

	ncx, err := parseNCX(ncxXML, "OEBPS")
	if err != nil {
		t.Fatalf("parseNCX() error = %v", err)
	}

	if ncx.UID != "empty-uid" {
		t.Errorf("UID = %q, want %q", ncx.UID, "empty-uid")
	}
	if ncx.DocTitle != "Empty Book" {
		t.Errorf("DocTitle = %q, want %q", ncx.DocTitle, "Empty Book")
	}
	if !reflect.DeepEqual(ncx.NavPoints, []NavPoint(nil)) && len(ncx.NavPoints) != 0 {
		t.Errorf("NavPoints = %v, want empty", ncx.NavPoints)
	}
}

func TestFindNAVPath(t *testing.T) {
	tests := []struct {
		name     string
		opf      *OPF
		wantPath string
		wantOK   bool
	}{
		{
			name: "nav item exists",
			opf: &OPF{
				Manifest: map[string]ManifestItem{
					"nav": {ID: "nav", Href: "OEBPS/nav.xhtml", MediaType: "application/xhtml+xml", Properties: []string{"nav"}},
					"ch1": {ID: "ch1", Href: "OEBPS/ch1.xhtml", MediaType: "application/xhtml+xml"},
				},
			},
			wantPath: "OEBPS/nav.xhtml",
			wantOK:   true,
		},
		{
			name: "nav among multiple properties",
			opf: &OPF{
				Manifest: map[string]ManifestItem{
					"nav": {ID: "nav", Href: "nav.xhtml", MediaType: "application/xhtml+xml", Properties: []string{"cover-image", "nav"}},
				},
			},
			wantPath: "nav.xhtml",
			wantOK:   true,
		},
		{
			name: "no nav item",
			opf: &OPF{
				Manifest: map[string]ManifestItem{
					"ch1": {ID: "ch1", Href: "ch1.xhtml", MediaType: "application/xhtml+xml"},
				},
			},
			wantPath: "",
			wantOK:   false,
		},
		{
			name: "empty manifest",
			opf: &OPF{
				Manifest: map[string]ManifestItem{},
			},
			wantPath: "",
			wantOK:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotPath, gotOK := findNAVPath(tt.opf)
			if gotPath != tt.wantPath {
				t.Errorf("findNAVPath() path = %q, want %q", gotPath, tt.wantPath)
			}
			if gotOK != tt.wantOK {
				t.Errorf("findNAVPath() ok = %v, want %v", gotOK, tt.wantOK)
			}
		})
	}
}

func TestParseNAV_Basic(t *testing.T) {
	navHTML := []byte(`<?xml version="1.0" encoding="UTF-8"?>
<html xmlns="http://www.w3.org/1999/xhtml" xmlns:epub="http://www.idpf.org/2007/ops">
<head><title>Navigation</title></head>
<body>
<nav epub:type="toc">
  <h1>Table of Contents</h1>
  <ol>
    <li><a href="chapter1.xhtml">Chapter 1</a></li>
    <li><a href="chapter2.xhtml">Chapter 2</a></li>
    <li><a href="chapter3.xhtml">Chapter 3</a></li>
  </ol>
</nav>
</body>
</html>`)

	ncx, err := parseNAV(navHTML, "OEBPS")
	if err != nil {
		t.Fatalf("parseNAV() error = %v", err)
	}

	if len(ncx.NavPoints) != 3 {
		t.Fatalf("got %d nav points, want 3", len(ncx.NavPoints))
	}

	// Check auto-generated IDs
	for i, np := range ncx.NavPoints {
		wantID := "nav-" + strconv.Itoa(i+1)
		if np.ID != wantID {
			t.Errorf("NavPoints[%d].ID = %q, want %q", i, np.ID, wantID)
		}
		if np.PlayOrder != i+1 {
			t.Errorf("NavPoints[%d].PlayOrder = %d, want %d", i, np.PlayOrder, i+1)
		}
	}

	if ncx.NavPoints[0].Label != "Chapter 1" {
		t.Errorf("NavPoints[0].Label = %q, want %q", ncx.NavPoints[0].Label, "Chapter 1")
	}
	if ncx.NavPoints[0].ContentPath != "OEBPS/chapter1.xhtml" {
		t.Errorf("NavPoints[0].ContentPath = %q, want %q", ncx.NavPoints[0].ContentPath, "OEBPS/chapter1.xhtml")
	}
}

func TestParseNAV_Nested(t *testing.T) {
	navHTML := []byte(`<?xml version="1.0" encoding="UTF-8"?>
<html xmlns="http://www.w3.org/1999/xhtml" xmlns:epub="http://www.idpf.org/2007/ops">
<body>
<nav epub:type="toc">
  <ol>
    <li>
      <a href="part1.xhtml">Part 1</a>
      <ol>
        <li><a href="ch1.xhtml">Chapter 1</a></li>
        <li><a href="ch2.xhtml">Chapter 2</a></li>
      </ol>
    </li>
    <li><a href="part2.xhtml">Part 2</a></li>
  </ol>
</nav>
</body>
</html>`)

	ncx, err := parseNAV(navHTML, "OEBPS")
	if err != nil {
		t.Fatalf("parseNAV() error = %v", err)
	}

	if len(ncx.NavPoints) != 2 {
		t.Fatalf("got %d top-level nav points, want 2", len(ncx.NavPoints))
	}

	// Part 1 with children
	p1 := ncx.NavPoints[0]
	if p1.Label != "Part 1" {
		t.Errorf("NavPoints[0].Label = %q, want %q", p1.Label, "Part 1")
	}
	if p1.PlayOrder != 1 {
		t.Errorf("NavPoints[0].PlayOrder = %d, want 1", p1.PlayOrder)
	}
	if len(p1.Children) != 2 {
		t.Fatalf("NavPoints[0].Children = %d, want 2", len(p1.Children))
	}

	ch1 := p1.Children[0]
	if ch1.Label != "Chapter 1" {
		t.Errorf("Children[0].Label = %q, want %q", ch1.Label, "Chapter 1")
	}
	if ch1.PlayOrder != 2 {
		t.Errorf("Children[0].PlayOrder = %d, want 2", ch1.PlayOrder)
	}

	ch2 := p1.Children[1]
	if ch2.PlayOrder != 3 {
		t.Errorf("Children[1].PlayOrder = %d, want 3", ch2.PlayOrder)
	}

	// Part 2
	p2 := ncx.NavPoints[1]
	if p2.PlayOrder != 4 {
		t.Errorf("NavPoints[1].PlayOrder = %d, want 4", p2.PlayOrder)
	}
	if len(p2.Children) != 0 {
		t.Errorf("NavPoints[1].Children = %d, want 0", len(p2.Children))
	}
}

func TestParseNAV_PathNormalization(t *testing.T) {
	navHTML := []byte(`<?xml version="1.0" encoding="UTF-8"?>
<html xmlns="http://www.w3.org/1999/xhtml" xmlns:epub="http://www.idpf.org/2007/ops">
<body>
<nav epub:type="toc">
  <ol>
    <li><a href="../text/chapter1.xhtml#sec1">Chapter 1</a></li>
  </ol>
</nav>
</body>
</html>`)

	ncx, err := parseNAV(navHTML, "OEBPS/nav")
	if err != nil {
		t.Fatalf("parseNAV() error = %v", err)
	}

	if len(ncx.NavPoints) != 1 {
		t.Fatalf("got %d nav points, want 1", len(ncx.NavPoints))
	}

	if ncx.NavPoints[0].ContentPath != "OEBPS/text/chapter1.xhtml" {
		t.Errorf("ContentPath = %q, want %q", ncx.NavPoints[0].ContentPath, "OEBPS/text/chapter1.xhtml")
	}
	if ncx.NavPoints[0].Fragment != "sec1" {
		t.Errorf("Fragment = %q, want %q", ncx.NavPoints[0].Fragment, "sec1")
	}
}

// createNCXTestEPUB creates a minimal EPUB file for LoadNCX testing.
// files is a map of path -> content to include in the ZIP.
func createNCXTestEPUB(t *testing.T, files map[string]string) string {
	t.Helper()
	dir := t.TempDir()
	epubPath := filepath.Join(dir, "test.epub")
	f, err := os.Create(epubPath)
	if err != nil {
		t.Fatalf("failed to create test epub: %v", err)
	}
	defer f.Close()

	w := zip.NewWriter(f)
	defer w.Close()

	// mimetype (must be uncompressed/stored)
	mw, err := w.CreateHeader(&zip.FileHeader{
		Name:   "mimetype",
		Method: zip.Store,
	})
	if err != nil {
		t.Fatalf("failed to create mimetype: %v", err)
	}
	mw.Write([]byte("application/epub+zip"))

	// META-INF/container.xml
	cw, err := w.Create("META-INF/container.xml")
	if err != nil {
		t.Fatalf("failed to create container.xml: %v", err)
	}
	cw.Write([]byte(`<?xml version="1.0" encoding="UTF-8"?>
<container version="1.0" xmlns="urn:oasis:names:tc:opendocument:xmlns:container">
  <rootfiles>
    <rootfile full-path="OEBPS/content.opf" media-type="application/oebps-package+xml"/>
  </rootfiles>
</container>`))

	for path, content := range files {
		fw, err := w.Create(path)
		if err != nil {
			t.Fatalf("failed to create %s: %v", path, err)
		}
		fw.Write([]byte(content))
	}

	return epubPath
}

func TestLoadNCX_NCXPriority(t *testing.T) {
	ncxContent := `<?xml version="1.0" encoding="UTF-8"?>
<ncx xmlns="http://www.daisy.org/z3986/2005/ncx/" version="2005-1">
  <head>
    <meta name="dtb:uid" content="ncx-uid"/>
    <meta name="dtb:depth" content="1"/>
  </head>
  <docTitle><text>NCX Book</text></docTitle>
  <navMap>
    <navPoint id="np1" playOrder="1">
      <navLabel><text>NCX Chapter 1</text></navLabel>
      <content src="chapter1.xhtml"/>
    </navPoint>
  </navMap>
</ncx>`

	navContent := `<?xml version="1.0" encoding="UTF-8"?>
<html xmlns="http://www.w3.org/1999/xhtml" xmlns:epub="http://www.idpf.org/2007/ops">
<body>
<nav epub:type="toc">
  <ol>
    <li><a href="chapter1.xhtml">NAV Chapter 1</a></li>
  </ol>
</nav>
</body>
</html>`

	epubPath := createNCXTestEPUB(t, map[string]string{
		"OEBPS/toc.ncx":   ncxContent,
		"OEBPS/nav.xhtml": navContent,
	})

	reader, err := Open(epubPath)
	if err != nil {
		t.Fatalf("Open() error = %v", err)
	}
	defer reader.Close()

	opf := &OPF{
		NCXPath: "OEBPS/toc.ncx",
		Manifest: map[string]ManifestItem{
			"ncx": {ID: "ncx", Href: "OEBPS/toc.ncx", MediaType: "application/x-dtbncx+xml"},
			"nav": {ID: "nav", Href: "OEBPS/nav.xhtml", MediaType: "application/xhtml+xml", Properties: []string{"nav"}},
		},
	}

	ncx, err := LoadNCX(reader, opf)
	if err != nil {
		t.Fatalf("LoadNCX() error = %v", err)
	}
	if ncx == nil {
		t.Fatal("LoadNCX() returned nil")
	}

	// Should use NCX, not NAV
	if ncx.UID != "ncx-uid" {
		t.Errorf("UID = %q, want %q (NCX should be prioritized)", ncx.UID, "ncx-uid")
	}
	if len(ncx.NavPoints) != 1 {
		t.Fatalf("got %d nav points, want 1", len(ncx.NavPoints))
	}
	if ncx.NavPoints[0].Label != "NCX Chapter 1" {
		t.Errorf("Label = %q, want %q", ncx.NavPoints[0].Label, "NCX Chapter 1")
	}
}

func TestLoadNCX_NAVFallback(t *testing.T) {
	navContent := `<?xml version="1.0" encoding="UTF-8"?>
<html xmlns="http://www.w3.org/1999/xhtml" xmlns:epub="http://www.idpf.org/2007/ops">
<body>
<nav epub:type="toc">
  <ol>
    <li><a href="chapter1.xhtml">NAV Chapter 1</a></li>
    <li><a href="chapter2.xhtml">NAV Chapter 2</a></li>
  </ol>
</nav>
</body>
</html>`

	epubPath := createNCXTestEPUB(t, map[string]string{
		"OEBPS/nav.xhtml": navContent,
	})

	reader, err := Open(epubPath)
	if err != nil {
		t.Fatalf("Open() error = %v", err)
	}
	defer reader.Close()

	opf := &OPF{
		NCXPath: "", // No NCX
		Manifest: map[string]ManifestItem{
			"nav": {ID: "nav", Href: "OEBPS/nav.xhtml", MediaType: "application/xhtml+xml", Properties: []string{"nav"}},
		},
	}

	ncx, err := LoadNCX(reader, opf)
	if err != nil {
		t.Fatalf("LoadNCX() error = %v", err)
	}
	if ncx == nil {
		t.Fatal("LoadNCX() returned nil")
	}

	if len(ncx.NavPoints) != 2 {
		t.Fatalf("got %d nav points, want 2", len(ncx.NavPoints))
	}
	if ncx.NavPoints[0].Label != "NAV Chapter 1" {
		t.Errorf("Label = %q, want %q", ncx.NavPoints[0].Label, "NAV Chapter 1")
	}
}

func TestLoadNCX_NeitherExists(t *testing.T) {
	epubPath := createNCXTestEPUB(t, map[string]string{
		"OEBPS/chapter1.xhtml": "<html><body>content</body></html>",
	})

	reader, err := Open(epubPath)
	if err != nil {
		t.Fatalf("Open() error = %v", err)
	}
	defer reader.Close()

	opf := &OPF{
		NCXPath: "",
		Manifest: map[string]ManifestItem{
			"ch1": {ID: "ch1", Href: "OEBPS/chapter1.xhtml", MediaType: "application/xhtml+xml"},
		},
	}

	ncx, err := LoadNCX(reader, opf)
	if err != nil {
		t.Fatalf("LoadNCX() error = %v", err)
	}
	if ncx != nil {
		t.Errorf("LoadNCX() = %v, want nil", ncx)
	}
}

func TestParseNAV_EpubTypeMultipleTokens(t *testing.T) {
	navHTML := []byte(`<?xml version="1.0" encoding="UTF-8"?>
<html xmlns="http://www.w3.org/1999/xhtml" xmlns:epub="http://www.idpf.org/2007/ops">
<body>
<nav epub:type="landmarks toc">
  <ol><li><a href="ch1.xhtml">Ch1</a></li></ol>
</nav>
</body>
</html>`)

	ncx, err := parseNAV(navHTML, "OEBPS")
	if err != nil {
		t.Fatalf("parseNAV() error = %v", err)
	}
	if len(ncx.NavPoints) != 1 {
		t.Fatalf("got %d nav points, want 1", len(ncx.NavPoints))
	}
	if ncx.NavPoints[0].Label != "Ch1" {
		t.Errorf("Label = %q, want %q", ncx.NavPoints[0].Label, "Ch1")
	}
}

func TestParseNAV_WrappedLink(t *testing.T) {
	navHTML := []byte(`<?xml version="1.0" encoding="UTF-8"?>
<html xmlns="http://www.w3.org/1999/xhtml" xmlns:epub="http://www.idpf.org/2007/ops">
<body>
<nav epub:type="toc">
  <ol>
    <li><span><a href="ch1.xhtml">Ch1</a></span></li>
  </ol>
</nav>
</body>
</html>`)

	ncx, err := parseNAV(navHTML, "OEBPS")
	if err != nil {
		t.Fatalf("parseNAV() error = %v", err)
	}
	if len(ncx.NavPoints) != 1 {
		t.Fatalf("got %d nav points, want 1", len(ncx.NavPoints))
	}
	if ncx.NavPoints[0].Label != "Ch1" {
		t.Errorf("Label = %q, want %q", ncx.NavPoints[0].Label, "Ch1")
	}
}

func TestParseNAV_HeadingWithoutLink(t *testing.T) {
	navHTML := []byte(`<?xml version="1.0" encoding="UTF-8"?>
<html xmlns="http://www.w3.org/1999/xhtml" xmlns:epub="http://www.idpf.org/2007/ops">
<body>
<nav epub:type="toc">
  <ol>
    <li>Part 1
      <ol><li><a href="ch1.xhtml">Ch1</a></li></ol>
    </li>
  </ol>
</nav>
</body>
</html>`)

	ncx, err := parseNAV(navHTML, "OEBPS")
	if err != nil {
		t.Fatalf("parseNAV() error = %v", err)
	}
	if len(ncx.NavPoints) != 1 {
		t.Fatalf("got %d top-level nav points, want 1", len(ncx.NavPoints))
	}
	if ncx.NavPoints[0].Label != "Part 1" {
		t.Errorf("Label = %q, want %q", ncx.NavPoints[0].Label, "Part 1")
	}
	if len(ncx.NavPoints[0].Children) != 1 {
		t.Fatalf("got %d children, want 1", len(ncx.NavPoints[0].Children))
	}
	if ncx.NavPoints[0].Children[0].Label != "Ch1" {
		t.Errorf("Child label = %q, want %q", ncx.NavPoints[0].Children[0].Label, "Ch1")
	}
}

func TestLoadNCX_NCXReadError(t *testing.T) {
	// NCXPath is set but file doesn't exist in the EPUB
	// With ErrFileNotFound, LoadNCX should fallback to NAV
	epubPath := createNCXTestEPUB(t, map[string]string{
		"OEBPS/chapter1.xhtml": "<html><body>content</body></html>",
	})

	reader, err := Open(epubPath)
	if err != nil {
		t.Fatalf("Open() error = %v", err)
	}
	defer reader.Close()

	opf := &OPF{
		NCXPath: "OEBPS/missing.ncx",
		Manifest: map[string]ManifestItem{
			"ch1": {ID: "ch1", Href: "OEBPS/chapter1.xhtml", MediaType: "application/xhtml+xml"},
		},
	}

	// NCX file not found, no NAV → should return nil, nil (file not found is expected)
	ncx, err := LoadNCX(reader, opf)
	if err != nil {
		t.Fatalf("LoadNCX() error = %v", err)
	}
	if ncx != nil {
		t.Errorf("LoadNCX() = %v, want nil", ncx)
	}
}

func TestErrFileNotFound(t *testing.T) {
	epubPath := createNCXTestEPUB(t, map[string]string{
		"OEBPS/chapter1.xhtml": "<html><body>content</body></html>",
	})

	reader, err := Open(epubPath)
	if err != nil {
		t.Fatalf("Open() error = %v", err)
	}
	defer reader.Close()

	_, err = reader.ReadFile("nonexistent.file")
	if !errors.Is(err, ErrFileNotFound) {
		t.Errorf("ReadFile() error = %v, want ErrFileNotFound", err)
	}
}
