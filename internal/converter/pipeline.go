package converter

// ConvertOptions holds options for the conversion pipeline.
type ConvertOptions struct {
	InputPath  string
	OutputPath string
}

// Pipeline orchestrates the EPUB to AZW3 conversion.
type Pipeline struct {
	Options ConvertOptions
}

// NewPipeline creates a new conversion pipeline.
func NewPipeline(opts ConvertOptions) *Pipeline {
	return &Pipeline{Options: opts}
}

// Convert executes the conversion pipeline.
func (p *Pipeline) Convert() error {
	return nil
}
