package converter

import (
	"bytes"
	"image"
	"image/color"
	"image/gif"
	"image/jpeg"
	"image/png"
	"testing"
)

func TestImageOptimizer_ResizeOverMaxWidth(t *testing.T) {
	src := makeSolidNRGBA(1200, 800, color.NRGBA{R: 20, G: 50, B: 200, A: 255})
	data := mustEncodeJPEG(t, src, 90)
	opt := NewImageOptimizer(ConvertOptions{MaxImageWidth: 600})

	out, err := opt.Optimize("img.jpg", "image/jpeg", data, false)
	if err != nil {
		t.Fatalf("Optimize() error = %v", err)
	}
	if out.Width != 600 || out.Height != 400 {
		t.Fatalf("got %dx%d, want 600x400", out.Width, out.Height)
	}
}

func TestImageOptimizer_NoResizeUnderMaxWidth(t *testing.T) {
	src := makeSolidNRGBA(500, 300, color.NRGBA{R: 100, G: 120, B: 140, A: 255})
	data := mustEncodeJPEG(t, src, 90)
	opt := NewImageOptimizer(ConvertOptions{MaxImageWidth: 600})

	out, err := opt.Optimize("img.jpg", "image/jpeg", data, false)
	if err != nil {
		t.Fatalf("Optimize() error = %v", err)
	}
	if out.Width != 500 || out.Height != 300 {
		t.Fatalf("got %dx%d, want 500x300", out.Width, out.Height)
	}
}

func TestImageOptimizer_PNGToJPEGWithoutAlpha(t *testing.T) {
	src := makeSolidNRGBA(700, 400, color.NRGBA{R: 10, G: 80, B: 180, A: 255})
	data := mustEncodePNG(t, src)
	opt := NewImageOptimizer(ConvertOptions{MaxImageWidth: 600})

	out, err := opt.Optimize("img.png", "image/png", data, false)
	if err != nil {
		t.Fatalf("Optimize() error = %v", err)
	}
	if out.Format != "jpeg" {
		t.Fatalf("format = %q, want jpeg", out.Format)
	}
}

func TestImageOptimizer_KeepTransparentPNG(t *testing.T) {
	src := makeSolidNRGBA(700, 400, color.NRGBA{R: 10, G: 80, B: 180, A: 120})
	data := mustEncodePNG(t, src)
	opt := NewImageOptimizer(ConvertOptions{MaxImageWidth: 600})

	out, err := opt.Optimize("alpha.png", "image/png", data, false)
	if err != nil {
		t.Fatalf("Optimize() error = %v", err)
	}
	if out.Format != "png" {
		t.Fatalf("format = %q, want png", out.Format)
	}
}

func TestImageOptimizer_GIFToJPEGWhenNotAnimated(t *testing.T) {
	p := image.NewPaletted(image.Rect(0, 0, 640, 320), color.Palette{
		color.RGBA{0, 0, 0, 255},
		color.RGBA{255, 255, 255, 255},
	})
	data := mustEncodeGIF(t, p)
	opt := NewImageOptimizer(ConvertOptions{})

	out, err := opt.Optimize("a.gif", "image/gif", data, false)
	if err != nil {
		t.Fatalf("Optimize() error = %v", err)
	}
	if out.Format != "jpeg" {
		t.Fatalf("format = %q, want jpeg", out.Format)
	}
}

func TestImageOptimizer_KeepAnimatedGIF(t *testing.T) {
	anim := &gif.GIF{
		Image: []*image.Paletted{
			image.NewPaletted(image.Rect(0, 0, 10, 10), color.Palette{color.Black, color.White}),
			image.NewPaletted(image.Rect(0, 0, 10, 10), color.Palette{color.Black, color.White}),
		},
		Delay: []int{5, 5},
	}
	var buf bytes.Buffer
	if err := gif.EncodeAll(&buf, anim); err != nil {
		t.Fatalf("gif.EncodeAll() error = %v", err)
	}

	opt := NewImageOptimizer(ConvertOptions{})
	out, err := opt.Optimize("anim.gif", "image/gif", buf.Bytes(), false)
	if err != nil {
		t.Fatalf("Optimize() error = %v", err)
	}
	if out.Format != "gif" {
		t.Fatalf("format = %q, want gif", out.Format)
	}
	if !bytes.Equal(out.Data, buf.Bytes()) {
		t.Fatal("animated gif should be passthrough")
	}
}

func TestImageOptimizer_LowerQualityProducesSmallerOutput(t *testing.T) {
	src := makePatternNRGBA(900, 700)
	data := mustEncodeJPEG(t, src, 95)

	high, err := NewImageOptimizer(ConvertOptions{JPEGQuality: 95}).Optimize("p.jpg", "image/jpeg", data, false)
	if err != nil {
		t.Fatalf("high quality Optimize() error = %v", err)
	}
	low, err := NewImageOptimizer(ConvertOptions{JPEGQuality: 70}).Optimize("p.jpg", "image/jpeg", data, false)
	if err != nil {
		t.Fatalf("low quality Optimize() error = %v", err)
	}

	if len(low.Data) >= len(high.Data) {
		t.Fatalf("low quality size = %d, high quality size = %d; want low < high", len(low.Data), len(high.Data))
	}
}

func TestImageOptimizer_SizeCapRecompression(t *testing.T) {
	src := makePatternNRGBA(1800, 1200)
	data := mustEncodeJPEG(t, src, 95)

	uncapped, err := NewImageOptimizer(ConvertOptions{
		JPEGQuality:       95,
		MaxImageSizeBytes: 0,
	}).Optimize("big.jpg", "image/jpeg", data, false)
	if err != nil {
		t.Fatalf("uncapped Optimize() error = %v", err)
	}

	cappedOpt := NewImageOptimizer(ConvertOptions{
		JPEGQuality:       95,
		MaxImageSizeBytes: 12 * 1024,
	})
	capped, err := cappedOpt.Optimize("big.jpg", "image/jpeg", data, false)
	if err != nil {
		t.Fatalf("capped Optimize() unexpected error = %v", err)
	}
	if len(capped.Data) >= len(uncapped.Data) {
		t.Fatalf("capped size = %d, uncapped size = %d; want capped < uncapped", len(capped.Data), len(uncapped.Data))
	}
}

func TestImageOptimizer_SizeExceedWarning(t *testing.T) {
	// サイズ上限を非常に小さくして、超過時に Warning が設定されることを確認
	src := makePatternNRGBA(400, 300)
	data := mustEncodePNG(t, src)

	opt := NewImageOptimizer(ConvertOptions{
		MaxImageSizeBytes: 100, // 非常に小さい
	})
	out, err := opt.Optimize("big.png", "image/png", data, false)
	if err != nil {
		t.Fatalf("Optimize() should not return error, got %v", err)
	}
	if out.Warning == "" {
		t.Fatal("expected warning for size exceed")
	}
	if len(out.Data) == 0 {
		t.Fatal("should still return optimized data even on size exceed")
	}
}

func TestImageOptimizer_CoverSkipsMaxWidthResize(t *testing.T) {
	src := makePatternNRGBA(1200, 800)
	data := mustEncodeJPEG(t, src, 95)
	opt := NewImageOptimizer(ConvertOptions{
		MaxImageWidth:     600,
		JPEGQuality:       80,
		MaxImageSizeBytes: 2 * 1024 * 1024,
	})

	out, err := opt.Optimize("cover.jpg", "image/jpeg", data, true)
	if err != nil {
		t.Fatalf("Optimize() error = %v", err)
	}
	if out.Width != 1200 || out.Height != 800 {
		t.Fatalf("cover size got %dx%d, want 1200x800", out.Width, out.Height)
	}
}

func TestImageOptimizer_DecodeFailurePassthrough(t *testing.T) {
	raw := []byte("not-an-image")
	opt := NewImageOptimizer(ConvertOptions{})

	out, err := opt.Optimize("bad.jpg", "image/jpeg", raw, false)
	if err != nil {
		t.Fatalf("decode failure should not return error, got %v", err)
	}
	if out.Warning == "" {
		t.Fatal("expected warning for decode failure")
	}
	if !bytes.Equal(out.Data, raw) {
		t.Fatal("decode failure should passthrough original bytes")
	}
}

func TestImageOptimizer_HugeImagePassthrough(t *testing.T) {
	// DecodeConfig で巨大画像（総ピクセル数が上限超過）と判定された場合、
	// デコードせずパススルーされることを確認する。
	// 実際に巨大画像を作るとメモリを消費するため、MaxPixels を小さく設定してテストする。
	src := makeSolidNRGBA(200, 200, color.NRGBA{R: 100, G: 100, B: 100, A: 255})
	data := mustEncodeJPEG(t, src, 90)

	opt := NewImageOptimizer(ConvertOptions{MaxImageWidth: 600})
	opt.MaxPixels = 100 * 100 // 200x200 = 40000 > 10000 なのでパススルーされるはず

	out, err := opt.Optimize("huge.jpg", "image/jpeg", data, false)
	if err != nil {
		t.Fatalf("huge image should not return error, got %v", err)
	}
	if out.Warning == "" {
		t.Fatal("expected warning for huge image")
	}
	if !bytes.Equal(out.Data, data) {
		t.Fatal("huge image should passthrough original bytes")
	}
	if out.Width != 200 || out.Height != 200 {
		t.Fatalf("got %dx%d, want 200x200", out.Width, out.Height)
	}
}

func TestImageOptimizer_AspectRatioPreserved(t *testing.T) {
	src := makePatternNRGBA(1200, 800)
	data := mustEncodeJPEG(t, src, 90)
	opt := NewImageOptimizer(ConvertOptions{MaxImageWidth: 600})

	out, err := opt.Optimize("ratio.jpg", "image/jpeg", data, false)
	if err != nil {
		t.Fatalf("Optimize() error = %v", err)
	}
	if out.Width != 600 || out.Height != 400 {
		t.Fatalf("got %dx%d, want 600x400", out.Width, out.Height)
	}
}

func makeSolidNRGBA(w, h int, c color.NRGBA) *image.NRGBA {
	img := image.NewNRGBA(image.Rect(0, 0, w, h))
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			img.SetNRGBA(x, y, c)
		}
	}
	return img
}

func makePatternNRGBA(w, h int) *image.NRGBA {
	img := image.NewNRGBA(image.Rect(0, 0, w, h))
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			r := uint8((x*17 + y*11) % 256)
			g := uint8((x*7 + y*23) % 256)
			b := uint8((x*3 + y*13) % 256)
			img.SetNRGBA(x, y, color.NRGBA{R: r, G: g, B: b, A: 255})
		}
	}
	return img
}

func mustEncodeJPEG(t *testing.T, img image.Image, quality int) []byte {
	t.Helper()
	var buf bytes.Buffer
	if err := jpeg.Encode(&buf, img, &jpeg.Options{Quality: quality}); err != nil {
		t.Fatalf("jpeg.Encode() error = %v", err)
	}
	return buf.Bytes()
}

func mustEncodePNG(t *testing.T, img image.Image) []byte {
	t.Helper()
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		t.Fatalf("png.Encode() error = %v", err)
	}
	return buf.Bytes()
}

func mustEncodeGIF(t *testing.T, img *image.Paletted) []byte {
	t.Helper()
	var buf bytes.Buffer
	if err := gif.Encode(&buf, img, nil); err != nil {
		t.Fatalf("gif.Encode() error = %v", err)
	}
	return buf.Bytes()
}
