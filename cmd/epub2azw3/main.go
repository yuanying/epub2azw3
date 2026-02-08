package main

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
	"github.com/yuanying/epub2azw3/internal/converter"
)

var rootCmd = &cobra.Command{
	Use:   "epub2azw3",
	Short: "Convert EPUB files to AZW3 (Kindle) format",
	Long: `epub2azw3 is a command-line tool that converts EPUB ebooks to
Amazon Kindle compatible AZW3 (KF8) format.

It is a standalone implementation in Go without external dependencies
like Calibre.`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		inputPath := args[0]
		outputPath, _ := cmd.Flags().GetString("output")

		if outputPath == "" {
			outputPath = strings.TrimSuffix(inputPath, filepath.Ext(inputPath)) + ".azw3"
		}

		log.Printf("Converting: %s -> %s", inputPath, outputPath)

		p := converter.NewPipeline(converter.ConvertOptions{
			InputPath:  inputPath,
			OutputPath: outputPath,
		})

		if err := p.Convert(); err != nil {
			return fmt.Errorf("conversion failed: %w", err)
		}

		log.Printf("Done: %s", outputPath)
		return nil
	},
}

func init() {
	rootCmd.Flags().StringP("output", "o", "", "Output file path (default: input with .azw3 extension)")
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}
