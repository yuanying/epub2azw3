package main

import (
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
	"github.com/yuanying/epub2azw3/internal/converter"
)

const (
	defaultJPEGQuality   = 85
	defaultMaxImageSize  = 127
	defaultMaxImageWidth = 600
)

type CLIOptions struct {
	OutputPath    string
	JPEGQuality   int
	MaxImageSize  int
	MaxImageWidth int
	NoImages      bool
	LogLevel      string
	LogFormat     string
	Strict        bool
	Verbose       bool
}

func normalizeLogLevel(level string, verbose bool) string {
	normalized := strings.ToLower(strings.TrimSpace(level))
	if normalized == "" {
		normalized = "info"
	}
	if verbose {
		return "debug"
	}
	return normalized
}

func defaultOutputPath(inputPath string) string {
	return strings.TrimSuffix(inputPath, filepath.Ext(inputPath)) + ".azw3"
}

func validateCLIOptions(opts CLIOptions) error {
	if opts.JPEGQuality < 60 || opts.JPEGQuality > 100 {
		return fmt.Errorf("invalid --quality %d (expected 60-100)", opts.JPEGQuality)
	}
	if opts.MaxImageSize <= 0 {
		return fmt.Errorf("invalid --max-image-size %d (expected > 0)", opts.MaxImageSize)
	}
	if opts.MaxImageWidth <= 0 {
		return fmt.Errorf("invalid --max-image-width %d (expected > 0)", opts.MaxImageWidth)
	}

	switch strings.ToLower(strings.TrimSpace(opts.LogLevel)) {
	case "error", "warn", "info", "debug":
	default:
		return fmt.Errorf("invalid --log-level %q (expected error/warn/info/debug)", opts.LogLevel)
	}

	switch strings.ToLower(strings.TrimSpace(opts.LogFormat)) {
	case "text", "json":
	default:
		return fmt.Errorf("invalid --log-format %q (expected text/json)", opts.LogFormat)
	}

	return nil
}

func parseSlogLevel(s string) slog.Level {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "error":
		return slog.LevelError
	case "warn":
		return slog.LevelWarn
	case "debug":
		return slog.LevelDebug
	default:
		return slog.LevelInfo
	}
}

func buildLogger(writer io.Writer, levelStr string, format string) *slog.Logger {
	level := parseSlogLevel(levelStr)
	format = strings.ToLower(strings.TrimSpace(format))
	removeTime := func(groups []string, a slog.Attr) slog.Attr {
		if a.Key == slog.TimeKey && len(groups) == 0 {
			return slog.Attr{}
		}
		return a
	}
	var handler slog.Handler
	switch format {
	case "json":
		handler = slog.NewJSONHandler(writer, &slog.HandlerOptions{Level: level})
	default:
		handler = slog.NewTextHandler(writer, &slog.HandlerOptions{
			Level:       level,
			ReplaceAttr: removeTime,
		})
	}
	return slog.New(handler)
}

func readCLIOptions(cmd *cobra.Command, args []string) (converter.ConvertOptions, error) {
	inputPath := args[0]

	outputPath, _ := cmd.Flags().GetString("output")
	quality, _ := cmd.Flags().GetInt("quality")
	maxImageSize, _ := cmd.Flags().GetInt("max-image-size")
	maxImageWidth, _ := cmd.Flags().GetInt("max-image-width")
	noImages, _ := cmd.Flags().GetBool("no-images")
	logLevel, _ := cmd.Flags().GetString("log-level")
	logFormat, _ := cmd.Flags().GetString("log-format")
	strict, _ := cmd.Flags().GetBool("strict")
	verbose, _ := cmd.Flags().GetBool("verbose")

	cliOpts := CLIOptions{
		OutputPath:    outputPath,
		JPEGQuality:   quality,
		MaxImageSize:  maxImageSize,
		MaxImageWidth: maxImageWidth,
		NoImages:      noImages,
		LogLevel:      normalizeLogLevel(logLevel, verbose),
		LogFormat:     logFormat,
		Strict:        strict,
		Verbose:       verbose,
	}

	if cliOpts.OutputPath == "" {
		cliOpts.OutputPath = defaultOutputPath(inputPath)
	}

	if err := validateCLIOptions(cliOpts); err != nil {
		return converter.ConvertOptions{}, err
	}

	return converter.ConvertOptions{
		InputPath:         inputPath,
		OutputPath:        cliOpts.OutputPath,
		MaxImageWidth:     cliOpts.MaxImageWidth,
		JPEGQuality:       cliOpts.JPEGQuality,
		MaxImageSizeBytes: cliOpts.MaxImageSize * 1024,
		NoImages:          cliOpts.NoImages,
		Strict:            cliOpts.Strict,
		Logger:            buildLogger(os.Stderr, cliOpts.LogLevel, cliOpts.LogFormat),
	}, nil
}

func newRootCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "epub2azw3",
		Short: "Convert EPUB files to AZW3 (Kindle) format",
		Long: `epub2azw3 is a command-line tool that converts EPUB ebooks to
Amazon Kindle compatible AZW3 (KF8) format.

It is a standalone implementation in Go without external dependencies
like Calibre.`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			opts, err := readCLIOptions(cmd, args)
			if err != nil {
				return err
			}

			pipeline := converter.NewPipeline(opts)
			if err := pipeline.Convert(); err != nil {
				return fmt.Errorf("conversion failed: %w", err)
			}
			return nil
		},
	}

	cmd.SetErr(os.Stderr)
	cmd.Flags().StringP("output", "o", "", "Output file path (default: input with .azw3 extension)")
	cmd.Flags().IntP("quality", "q", defaultJPEGQuality, "JPEG quality (60-100)")
	cmd.Flags().Int("max-image-size", defaultMaxImageSize, "Max image size in KB")
	cmd.Flags().Int("max-image-width", defaultMaxImageWidth, "Max image width in pixels")
	cmd.Flags().Bool("no-images", false, "Remove all images from output")
	cmd.Flags().StringP("log-level", "l", "info", "Log level (error/warn/info/debug)")
	cmd.Flags().String("log-format", "text", "Log output format (text/json)")
	cmd.Flags().Bool("strict", false, "Treat recoverable warnings as errors")
	cmd.Flags().BoolP("verbose", "v", false, "Enable verbose output")
	return cmd
}

func main() {
	if err := newRootCmd().Execute(); err != nil {
		os.Exit(1)
	}
}
