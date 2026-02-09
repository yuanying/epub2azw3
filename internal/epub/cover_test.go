package epub

import "testing"

func TestDetectCover_Properties(t *testing.T) {
	opf := &OPF{
		Manifest: map[string]ManifestItem{
			"cover-img": {
				ID:         "cover-img",
				Href:       "images/cover.jpg",
				MediaType:  "image/jpeg",
				Properties: []string{"cover-image"},
			},
			"ch1": {
				ID:        "ch1",
				Href:      "text/ch1.xhtml",
				MediaType: "application/xhtml+xml",
			},
		},
		ManifestOrder: []string{"cover-img", "ch1"},
	}

	info := opf.DetectCover()
	if info == nil {
		t.Fatal("DetectCover() returned nil, want CoverInfo")
	}
	if info.ManifestID != "cover-img" {
		t.Errorf("ManifestID = %q, want %q", info.ManifestID, "cover-img")
	}
	if info.Href != "images/cover.jpg" {
		t.Errorf("Href = %q, want %q", info.Href, "images/cover.jpg")
	}
	if info.MediaType != "image/jpeg" {
		t.Errorf("MediaType = %q, want %q", info.MediaType, "image/jpeg")
	}
	if info.DetectionMethod != "properties" {
		t.Errorf("DetectionMethod = %q, want %q", info.DetectionMethod, "properties")
	}
}

func TestDetectCover_Meta(t *testing.T) {
	opf := &OPF{
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
		ManifestOrder: []string{"cover-image"},
	}

	info := opf.DetectCover()
	if info == nil {
		t.Fatal("DetectCover() returned nil, want CoverInfo")
	}
	if info.ManifestID != "cover-image" {
		t.Errorf("ManifestID = %q, want %q", info.ManifestID, "cover-image")
	}
	if info.DetectionMethod != "meta" {
		t.Errorf("DetectionMethod = %q, want %q", info.DetectionMethod, "meta")
	}
}

func TestDetectCover_Guide(t *testing.T) {
	opf := &OPF{
		Manifest: map[string]ManifestItem{
			"cover-img": {
				ID:        "cover-img",
				Href:      "OEBPS/images/cover.jpg",
				MediaType: "image/jpeg",
			},
			"cover-page": {
				ID:        "cover-page",
				Href:      "OEBPS/cover.xhtml",
				MediaType: "application/xhtml+xml",
			},
		},
		ManifestOrder: []string{"cover-img", "cover-page"},
		Guide: []GuideReference{
			{
				Type:  "cover",
				Title: "Cover",
				Href:  "OEBPS/images/cover.jpg",
			},
		},
	}

	info := opf.DetectCover()
	if info == nil {
		t.Fatal("DetectCover() returned nil, want CoverInfo")
	}
	if info.ManifestID != "cover-img" {
		t.Errorf("ManifestID = %q, want %q", info.ManifestID, "cover-img")
	}
	if info.DetectionMethod != "guide" {
		t.Errorf("DetectionMethod = %q, want %q", info.DetectionMethod, "guide")
	}
}

func TestDetectCover_GuideWithFragment(t *testing.T) {
	opf := &OPF{
		Manifest: map[string]ManifestItem{
			"cover-img": {
				ID:        "cover-img",
				Href:      "OEBPS/images/cover.jpg",
				MediaType: "image/jpeg",
			},
		},
		ManifestOrder: []string{"cover-img"},
		Guide: []GuideReference{
			{
				Type: "cover",
				Href: "OEBPS/images/cover.jpg#fragment",
			},
		},
	}

	info := opf.DetectCover()
	if info == nil {
		t.Fatal("DetectCover() returned nil, want CoverInfo")
	}
	if info.ManifestID != "cover-img" {
		t.Errorf("ManifestID = %q, want %q", info.ManifestID, "cover-img")
	}
	if info.DetectionMethod != "guide" {
		t.Errorf("DetectionMethod = %q, want %q", info.DetectionMethod, "guide")
	}
}

func TestDetectCover_GuidePointsToXHTML_SkipsToFilename(t *testing.T) {
	// Guide points to an XHTML page (not an image) → skip guide, fall through to filename
	opf := &OPF{
		Manifest: map[string]ManifestItem{
			"cover-page": {
				ID:        "cover-page",
				Href:      "OEBPS/cover.xhtml",
				MediaType: "application/xhtml+xml",
			},
			"img-cover": {
				ID:        "img-cover",
				Href:      "OEBPS/images/cover.png",
				MediaType: "image/png",
			},
		},
		ManifestOrder: []string{"cover-page", "img-cover"},
		Guide: []GuideReference{
			{
				Type: "cover",
				Href: "OEBPS/cover.xhtml",
			},
		},
	}

	info := opf.DetectCover()
	if info == nil {
		t.Fatal("DetectCover() returned nil, want CoverInfo")
	}
	// Should fall through to filename detection since guide points to XHTML
	if info.ManifestID != "img-cover" {
		t.Errorf("ManifestID = %q, want %q", info.ManifestID, "img-cover")
	}
	if info.DetectionMethod != "filename" {
		t.Errorf("DetectionMethod = %q, want %q", info.DetectionMethod, "filename")
	}
}

func TestDetectCover_Filename(t *testing.T) {
	opf := &OPF{
		Manifest: map[string]ManifestItem{
			"img1": {
				ID:        "img1",
				Href:      "images/Cover_Image.jpg",
				MediaType: "image/jpeg",
			},
			"ch1": {
				ID:        "ch1",
				Href:      "text/ch1.xhtml",
				MediaType: "application/xhtml+xml",
			},
		},
		ManifestOrder: []string{"img1", "ch1"},
	}

	info := opf.DetectCover()
	if info == nil {
		t.Fatal("DetectCover() returned nil, want CoverInfo")
	}
	if info.ManifestID != "img1" {
		t.Errorf("ManifestID = %q, want %q", info.ManifestID, "img1")
	}
	if info.DetectionMethod != "filename" {
		t.Errorf("DetectionMethod = %q, want %q", info.DetectionMethod, "filename")
	}
}

func TestDetectCover_FilenameSVGExcluded(t *testing.T) {
	// SVG files should be excluded from filename detection
	opf := &OPF{
		Manifest: map[string]ManifestItem{
			"svg-cover": {
				ID:        "svg-cover",
				Href:      "images/cover.svg",
				MediaType: "image/svg+xml",
			},
		},
		ManifestOrder: []string{"svg-cover"},
	}

	info := opf.DetectCover()
	if info != nil {
		t.Errorf("DetectCover() = %+v, want nil (SVG should be excluded)", info)
	}
}

func TestDetectCover_NoCover(t *testing.T) {
	opf := &OPF{
		Manifest: map[string]ManifestItem{
			"ch1": {
				ID:        "ch1",
				Href:      "chapter.xhtml",
				MediaType: "application/xhtml+xml",
			},
		},
		ManifestOrder: []string{"ch1"},
	}

	info := opf.DetectCover()
	if info != nil {
		t.Errorf("DetectCover() = %+v, want nil", info)
	}
}

func TestDetectCover_Priority_PropertiesOverMeta(t *testing.T) {
	opf := &OPF{
		Metadata: Metadata{
			CoverID: "meta-cover",
		},
		Manifest: map[string]ManifestItem{
			"prop-cover": {
				ID:         "prop-cover",
				Href:       "images/prop-cover.jpg",
				MediaType:  "image/jpeg",
				Properties: []string{"cover-image"},
			},
			"meta-cover": {
				ID:        "meta-cover",
				Href:      "images/meta-cover.jpg",
				MediaType: "image/jpeg",
			},
		},
		ManifestOrder: []string{"prop-cover", "meta-cover"},
	}

	info := opf.DetectCover()
	if info == nil {
		t.Fatal("DetectCover() returned nil")
	}
	if info.ManifestID != "prop-cover" {
		t.Errorf("ManifestID = %q, want %q (properties should take priority over meta)", info.ManifestID, "prop-cover")
	}
	if info.DetectionMethod != "properties" {
		t.Errorf("DetectionMethod = %q, want %q", info.DetectionMethod, "properties")
	}
}

func TestDetectCover_Priority_MetaOverGuide(t *testing.T) {
	opf := &OPF{
		Metadata: Metadata{
			CoverID: "meta-cover",
		},
		Manifest: map[string]ManifestItem{
			"meta-cover": {
				ID:        "meta-cover",
				Href:      "images/meta-cover.jpg",
				MediaType: "image/jpeg",
			},
			"guide-cover": {
				ID:        "guide-cover",
				Href:      "images/guide-cover.jpg",
				MediaType: "image/jpeg",
			},
		},
		ManifestOrder: []string{"meta-cover", "guide-cover"},
		Guide: []GuideReference{
			{
				Type: "cover",
				Href: "images/guide-cover.jpg",
			},
		},
	}

	info := opf.DetectCover()
	if info == nil {
		t.Fatal("DetectCover() returned nil")
	}
	if info.ManifestID != "meta-cover" {
		t.Errorf("ManifestID = %q, want %q (meta should take priority over guide)", info.ManifestID, "meta-cover")
	}
	if info.DetectionMethod != "meta" {
		t.Errorf("DetectionMethod = %q, want %q", info.DetectionMethod, "meta")
	}
}

func TestFindCoverImage_DelegatesToDetectCover(t *testing.T) {
	opf := &OPF{
		Manifest: map[string]ManifestItem{
			"cover": {
				ID:         "cover",
				Href:       "images/cover.jpg",
				MediaType:  "image/jpeg",
				Properties: []string{"cover-image"},
			},
		},
		ManifestOrder: []string{"cover"},
	}

	href, ok := opf.FindCoverImage()
	if !ok {
		t.Fatal("FindCoverImage() ok = false, want true")
	}
	if href != "images/cover.jpg" {
		t.Errorf("FindCoverImage() href = %q, want %q", href, "images/cover.jpg")
	}
}

func TestFindCoverImage_NoCover(t *testing.T) {
	opf := &OPF{
		Manifest: map[string]ManifestItem{
			"ch1": {
				ID:        "ch1",
				Href:      "chapter.xhtml",
				MediaType: "application/xhtml+xml",
			},
		},
		ManifestOrder: []string{"ch1"},
	}

	href, ok := opf.FindCoverImage()
	if ok {
		t.Errorf("FindCoverImage() ok = true, want false")
	}
	if href != "" {
		t.Errorf("FindCoverImage() href = %q, want empty", href)
	}
}
