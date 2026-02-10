package main

import (
	"bytes"
	"context"
	"log/slog"
	"strings"
	"testing"
)

func readConvertOptionsForTest(t *testing.T, flagArgs ...string) error {
	t.Helper()
	cmd := newRootCmd()
	if err := cmd.ParseFlags(flagArgs); err != nil {
		return err
	}
	_, err := readCLIOptions(cmd, []string{"./input/book.epub"})
	return err
}

func TestReadCLIOptions_Defaults(t *testing.T) {
	cmd := newRootCmd()
	opts, err := readCLIOptions(cmd, []string{"./input/book.epub"})
	if err != nil {
		t.Fatalf("readCLIOptions() error = %v", err)
	}

	if opts.OutputPath != "./input/book.azw3" {
		t.Fatalf("OutputPath = %q, want %q", opts.OutputPath, "./input/book.azw3")
	}
	if opts.JPEGQuality != defaultJPEGQuality {
		t.Fatalf("JPEGQuality = %d, want %d", opts.JPEGQuality, defaultJPEGQuality)
	}
	if opts.MaxImageWidth != defaultMaxImageWidth {
		t.Fatalf("MaxImageWidth = %d, want %d", opts.MaxImageWidth, defaultMaxImageWidth)
	}
	if opts.MaxImageSizeBytes != defaultMaxImageSize*1024 {
		t.Fatalf("MaxImageSizeBytes = %d, want %d", opts.MaxImageSizeBytes, defaultMaxImageSize*1024)
	}
	if opts.Logger == nil {
		t.Fatal("Logger is nil, want non-nil")
	}
	if !opts.Logger.Enabled(context.Background(), slog.LevelInfo) {
		t.Fatal("Logger should be enabled at INFO level by default")
	}
}

func TestReadCLIOptions_CustomFlags(t *testing.T) {
	cmd := newRootCmd()
	if err := cmd.ParseFlags([]string{
		"--output", "./out/custom.azw3",
		"--format", "mobi",
		"--quality", "90",
		"--max-image-size", "200",
		"--max-image-width", "720",
		"--no-images",
		"--log-level", "warn",
		"--strict",
		"--verbose",
	}); err != nil {
		t.Fatalf("ParseFlags() error = %v", err)
	}

	opts, err := readCLIOptions(cmd, []string{"./input/book.epub"})
	if err != nil {
		t.Fatalf("readCLIOptions() error = %v", err)
	}

	if opts.OutputPath != "./out/custom.azw3" {
		t.Fatalf("OutputPath = %q", opts.OutputPath)
	}
	if opts.JPEGQuality != 90 {
		t.Fatalf("JPEGQuality = %d", opts.JPEGQuality)
	}
	if opts.MaxImageSizeBytes != 200*1024 {
		t.Fatalf("MaxImageSizeBytes = %d", opts.MaxImageSizeBytes)
	}
	if opts.MaxImageWidth != 720 {
		t.Fatalf("MaxImageWidth = %d", opts.MaxImageWidth)
	}
	if !opts.NoImages {
		t.Fatal("NoImages = false, want true")
	}
	// --verbose overrides log-level to debug
	if !opts.Logger.Enabled(context.Background(), slog.LevelDebug) {
		t.Fatal("Logger should be enabled at DEBUG level when --verbose is set")
	}
	if !opts.Strict {
		t.Fatal("Strict = false, want true")
	}
}

func TestReadCLIOptions_InvalidQuality(t *testing.T) {
	err := readConvertOptionsForTest(t, "--quality", "59")
	if err == nil || !strings.Contains(err.Error(), "--quality") {
		t.Fatalf("expected quality validation error, got %v", err)
	}

	err = readConvertOptionsForTest(t, "--quality", "101")
	if err == nil || !strings.Contains(err.Error(), "--quality") {
		t.Fatalf("expected quality validation error, got %v", err)
	}
}

func TestReadCLIOptions_InvalidMaxImageSize(t *testing.T) {
	err := readConvertOptionsForTest(t, "--max-image-size", "0")
	if err == nil || !strings.Contains(err.Error(), "--max-image-size") {
		t.Fatalf("expected max-image-size validation error, got %v", err)
	}
}

func TestReadCLIOptions_InvalidMaxImageWidth(t *testing.T) {
	err := readConvertOptionsForTest(t, "--max-image-width", "0")
	if err == nil || !strings.Contains(err.Error(), "--max-image-width") {
		t.Fatalf("expected max-image-width validation error, got %v", err)
	}
}

func TestReadCLIOptions_InvalidLogLevel(t *testing.T) {
	err := readConvertOptionsForTest(t, "--log-level", "trace")
	if err == nil || !strings.Contains(err.Error(), "--log-level") {
		t.Fatalf("expected log-level validation error, got %v", err)
	}
}

func TestReadCLIOptions_InvalidLogFormat(t *testing.T) {
	err := readConvertOptionsForTest(t, "--log-format", "yaml")
	if err == nil || !strings.Contains(err.Error(), "--log-format") {
		t.Fatalf("expected log-format validation error, got %v", err)
	}
}

func TestReadCLIOptions_InvalidOutputFormat(t *testing.T) {
	err := readConvertOptionsForTest(t, "--format", "pdf")
	if err == nil || !strings.Contains(err.Error(), "--format") {
		t.Fatalf("expected format validation error, got %v", err)
	}
}

func TestReadCLIOptions_JSONFormat(t *testing.T) {
	cmd := newRootCmd()
	if err := cmd.ParseFlags([]string{"--log-format", "json"}); err != nil {
		t.Fatalf("ParseFlags() error = %v", err)
	}

	opts, err := readCLIOptions(cmd, []string{"./input/book.epub"})
	if err != nil {
		t.Fatalf("readCLIOptions() error = %v", err)
	}

	if opts.Logger == nil {
		t.Fatal("Logger is nil, want non-nil")
	}
	if !opts.Logger.Enabled(context.Background(), slog.LevelInfo) {
		t.Fatal("Logger should be enabled at INFO level")
	}
}

func TestBuildLogger_FormatNormalization(t *testing.T) {
	var buf bytes.Buffer
	logger := buildLogger(&buf, "info", "JSON")
	logger.Info("test message")
	// JSON format should produce JSON output (starts with '{')
	output := buf.String()
	if len(output) == 0 || output[0] != '{' {
		t.Fatalf("expected JSON output for format 'JSON', got: %s", output)
	}
}

func TestDefaultOutputPath(t *testing.T) {
	got := defaultOutputPath("./books/sample.epub", "azw3")
	if got != "./books/sample.azw3" {
		t.Fatalf("defaultOutputPath() = %q", got)
	}
}

func TestDefaultOutputPath_MOBI(t *testing.T) {
	got := defaultOutputPath("./books/sample.epub", "mobi")
	if got != "./books/sample.mobi" {
		t.Fatalf("defaultOutputPath() = %q", got)
	}
}
