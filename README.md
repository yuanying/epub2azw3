# epub2azw3

EPUB to AZW3 converter - a standalone Go implementation for converting EPUB ebooks to Amazon Kindle compatible AZW3 (KF8) format without external dependencies like Calibre.

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