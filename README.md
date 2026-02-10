# epub2azw3

EPUB to AZW3 converter - a standalone Go implementation for converting EPUB ebooks to Amazon Kindle compatible AZW3 (KF8) format without external dependencies like Calibre.

## Usage

```bash
epub2azw3 [flags] <input.epub>
```

### Flags

- `-o, --output`: output file path (default: `<input>.azw3`)
- `-q, --quality`: JPEG quality (`60-100`, default: `85`)
- `--max-image-size`: max image size in KB (default: `127`)
- `--max-image-width`: max image width in px (default: `600`)
- `--no-images`: remove all images from output
- `-l, --log-level`: `error|warn|info|debug` (default: `info`)
- `--log-format`: `text|json` (default: `text`)
- `--strict`: treat recoverable warnings as errors
- `-v, --verbose`: enable verbose output (forces debug logging)

## Development

### Build

```bash
go build -o epub2azw3 ./cmd/epub2azw3
```

### Test

```bash
go test ./...
```

### Lint

```bash
go tool golangci-lint run ./...
```
