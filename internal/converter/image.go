package converter

import (
	"bytes"
	"fmt"
	"image"
	"image/gif"
	"image/jpeg"
	"image/png"
	"strings"

	"github.com/disintegration/imaging"
)

const (
	defaultMaxImageWidth    = 600
	defaultJPEGQuality      = 85
	defaultMaxImageSize     = 256 * 1024 // KF8 format target; Kindle旧世代互換は 127KB
	minJPEGQuality          = 60
	defaultCoverJPEGQuality = 90
	defaultMaxPixels        = 100 * 1000 * 1000 // 100 megapixels
)

// ImageOptimizer optimizes raster images for Kindle output.
type ImageOptimizer struct {
	MaxWidth         int
	JPEGQuality      int
	MaxFileSize      int
	MinJPEGQuality   int
	CoverJPEGQuality int
	MaxPixels        int // Total pixel count limit for decode (width * height)
}

// OptimizedImage holds optimized image data and metadata.
// Warning is set (non-empty) when the image was returned as-is (passthrough)
// or when optimization completed but constraints like size limits were not met.
// In both cases Data is usable; Warning provides diagnostic information.
type OptimizedImage struct {
	Data         []byte
	Width        int
	Height       int
	Format       string
	OriginalPath string
	Warning      string
}

// NewImageOptimizer creates an image optimizer with defaults.
func NewImageOptimizer(opts ConvertOptions) *ImageOptimizer {
	maxWidth := opts.MaxImageWidth
	if maxWidth <= 0 {
		maxWidth = defaultMaxImageWidth
	}

	quality := opts.JPEGQuality
	if quality <= 0 {
		quality = defaultJPEGQuality
	}
	if quality > 100 {
		quality = 100
	}

	maxSize := opts.MaxImageSizeBytes
	if maxSize <= 0 {
		maxSize = defaultMaxImageSize
	}

	return &ImageOptimizer{
		MaxWidth:         maxWidth,
		JPEGQuality:      quality,
		MaxFileSize:      maxSize,
		MinJPEGQuality:   minJPEGQuality,
		CoverJPEGQuality: defaultCoverJPEGQuality,
		MaxPixels:        defaultMaxPixels,
	}
}

// Optimize decodes and optimizes image data.
// On decode failure or size constraint violation, it sets Warning on the result
// and returns the best available data (passthrough or optimized).
// Only encoding errors that prevent producing any output return a non-nil error.
func (o *ImageOptimizer) Optimize(path, mediaType string, input []byte, isCover bool) (OptimizedImage, error) {
	out := OptimizedImage{
		Data:         input,
		Format:       mediaTypeToFormat(mediaType),
		OriginalPath: path,
	}

	cfg, cfgFormat, cfgErr := image.DecodeConfig(bytes.NewReader(input))
	if cfgErr == nil {
		out.Width = cfg.Width
		out.Height = cfg.Height
		if out.Format == "" {
			out.Format = strings.ToLower(cfgFormat)
		}
		pixels := uint64(cfg.Width) * uint64(cfg.Height)
		if o.MaxPixels > 0 && pixels > uint64(o.MaxPixels) {
			out.Warning = fmt.Sprintf("image too large to decode: %dx%d (%d pixels)", cfg.Width, cfg.Height, pixels)
			return out, nil
		}
	}

	if strings.EqualFold(mediaType, "image/gif") {
		animated, err := isAnimatedGIF(input)
		if err == nil && animated {
			return out, nil
		}
	}

	src, decodedFormat, err := image.Decode(bytes.NewReader(input))
	if err != nil {
		out.Warning = fmt.Sprintf("image decode failed: %v", err)
		return out, nil
	}
	if out.Format == "" {
		out.Format = strings.ToLower(decodedFormat)
	}

	processed := src
	if !isCover && o.MaxWidth > 0 && src.Bounds().Dx() > o.MaxWidth {
		processed = imaging.Resize(src, o.MaxWidth, 0, imaging.Lanczos)
	}

	targetFormat := chooseTargetFormat(mediaType, out.Format, processed)
	var data []byte
	var qualityUsed int

	switch targetFormat {
	case "jpeg", "jpg":
		quality := o.JPEGQuality
		if isCover && quality < o.CoverJPEGQuality {
			quality = o.CoverJPEGQuality
		}
		data, qualityUsed, err = o.encodeJPEGWithSizeLimit(processed, quality, isCover)
		if err != nil {
			return out, err
		}
		targetFormat = "jpeg"
	case "png":
		data, err = encodePNG(processed)
		if err != nil {
			return out, fmt.Errorf("png encode failed: %w", err)
		}
	case "gif":
		data = input
	default:
		data = input
	}

	out.Data = data
	out.Width = processed.Bounds().Dx()
	out.Height = processed.Bounds().Dy()
	out.Format = targetFormat

	if o.MaxFileSize > 0 && len(out.Data) > o.MaxFileSize {
		if targetFormat == "jpeg" {
			out.Warning = fmt.Sprintf("jpeg size %d exceeds limit %d bytes at quality %d", len(out.Data), o.MaxFileSize, qualityUsed)
		} else {
			out.Warning = fmt.Sprintf("image size %d exceeds limit %d bytes", len(out.Data), o.MaxFileSize)
		}
	}

	return out, nil
}

func (o *ImageOptimizer) encodeJPEGWithSizeLimit(img image.Image, startQuality int, isCover bool) ([]byte, int, error) {
	quality := startQuality
	if quality > 100 {
		quality = 100
	}

	minQuality := o.MinJPEGQuality
	if isCover && minQuality < o.CoverJPEGQuality {
		minQuality = o.CoverJPEGQuality
	}
	if quality < minQuality {
		quality = minQuality
	}

	best, err := encodeJPEG(img, quality)
	if err != nil {
		return nil, 0, fmt.Errorf("jpeg encode failed: %w", err)
	}

	if o.MaxFileSize <= 0 || len(best) <= o.MaxFileSize {
		return best, quality, nil
	}

	bestQuality := quality
	for q := quality - 5; q >= minQuality; q -= 5 {
		candidate, encErr := encodeJPEG(img, q)
		if encErr != nil {
			return nil, 0, fmt.Errorf("jpeg re-encode failed at quality %d: %w", q, encErr)
		}
		best = candidate
		bestQuality = q
		if len(candidate) <= o.MaxFileSize {
			return candidate, q, nil
		}
	}

	return best, bestQuality, nil
}

// chooseTargetFormat determines the output format for an image.
// Transparent PNGs are kept as PNG to preserve alpha; opaque PNGs are
// converted to JPEG for smaller file size. An alternative approach would be
// to composite transparent PNGs onto a white background and convert to JPEG,
// but PNG preservation avoids quality loss and keeps transparency information.
func chooseTargetFormat(mediaType, detected string, img image.Image) string {
	switch strings.ToLower(mediaType) {
	case "image/jpeg", "image/jpg":
		return "jpeg"
	case "image/png":
		if hasAlpha(img) {
			return "png"
		}
		return "jpeg"
	case "image/gif":
		return "jpeg"
	}

	switch strings.ToLower(detected) {
	case "jpeg", "jpg":
		return "jpeg"
	case "png":
		if hasAlpha(img) {
			return "png"
		}
		return "jpeg"
	case "gif":
		return "jpeg"
	}

	return "jpeg"
}

func mediaTypeToFormat(mediaType string) string {
	switch strings.ToLower(mediaType) {
	case "image/jpeg", "image/jpg":
		return "jpeg"
	case "image/png":
		return "png"
	case "image/gif":
		return "gif"
	default:
		return ""
	}
}

func encodeJPEG(img image.Image, quality int) ([]byte, error) {
	var buf bytes.Buffer
	if err := jpeg.Encode(&buf, img, &jpeg.Options{Quality: quality}); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func encodePNG(img image.Image) ([]byte, error) {
	var buf bytes.Buffer
	encoder := png.Encoder{CompressionLevel: png.BestCompression}
	if err := encoder.Encode(&buf, img); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func isAnimatedGIF(data []byte) (bool, error) {
	g, err := gif.DecodeAll(bytes.NewReader(data))
	if err != nil {
		return false, err
	}
	return len(g.Image) > 1, nil
}

func hasAlpha(img image.Image) bool {
	bounds := img.Bounds()
	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			_, _, _, a := img.At(x, y).RGBA()
			if a < 0xFFFF {
				return true
			}
		}
	}
	return false
}
