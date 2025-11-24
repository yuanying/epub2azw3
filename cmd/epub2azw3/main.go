package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
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
			// Default output path: replace .epub with .azw3
			outputPath = inputPath[:len(inputPath)-5] + ".azw3"
		}

		fmt.Printf("Converting: %s -> %s\n", inputPath, outputPath)

		// TODO: Implement conversion
		return fmt.Errorf("conversion not yet implemented")
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
