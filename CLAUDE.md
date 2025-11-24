# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

EPUB to AZW3 converter - a standalone Go implementation for converting EPUB ebooks to Amazon Kindle compatible AZW3 (KF8) format without external dependencies like Calibre.

## Build Commands

```bash
# Build the CLI
go build -o epub2azw3 ./cmd/epub2azw3

# Run tests
go test ./...

# Run a specific test
go test -v -run TestFunctionName ./internal/epub

# Run tests with coverage
go test -coverprofile=coverage.out ./...
go tool cover -html=coverage.out
```

## Architecture

The project follows a pipeline architecture for converting EPUB to AZW3:

```
EPUB Input → Parse → Validate → Transform HTML/CSS → Optimize Images → Generate TOC → Build AZW3 → Output
```

### Package Structure

- **cmd/epub2azw3**: CLI entry point using Cobra
- **internal/epub**: EPUB parsing (ZIP extraction, OPF/NCX/container.xml parsing, content loading)
- **internal/converter**: Transformation logic (HTML→Kindle-compatible, CSS optimization, image processing, TOC generation)
- **internal/mobi**: AZW3/MOBI file generation (PDB structure, MOBI header, EXTH records, PalmDoc compression, record assembly)
- **internal/util**: Utilities (path resolution, encoding, Palm epoch time conversion)
- **pkg/epub2azw3**: Public API for library usage

### Key Technical Details

- All binary data uses **big-endian** byte order
- Text encoding is **UTF-8** (MOBI encoding type 65001)
- PalmDoc compression is the recommended compression method
- Image references use `kindle:embed:XXXX` format (4-digit hex record number)
- TOC entries use `filepos` attributes pointing to byte offsets in uncompressed HTML

### Implementation Reference

See `spec.md` for complete technical specifications including:
- EPUB format parsing details
- AZW3/MOBI binary format structure
- PalmDoc compression algorithm
- EXTH metadata mapping
- Implementation phases and priorities

## Permission Rules

### ファイル操作時の規約
すべてのファイル操作（Read, Write, Edit, およびBashコマンドのrm, mv, cp, mkdir, touch, chmod）を行う際は、必ず `./` プレフィックスを付けてリポジトリ内の相対パスを指定すること。

```bash
# 正しい例
Read: ./README.md
Write: ./internal/epub/reader.go
Edit: ./cmd/epub2azw3/main.go
rm ./temp.txt
mv ./old.go ./new.go
mkdir -p ./internal/newpkg

# 誤った例（許可されない）
Read: /home/user/src/project/README.md
rm README.md
```

この制約により、リポジトリ外のファイルへの誤操作を防止する。

## Development Workflow

### Issue Management

実装計画の各フェーズの項目はGitHub Issueとして管理する。

**Issue作成時のガイドライン:**
- 具体的なソースコードは含めない
- 実装に必要な情報を過不足なくドキュメントとして記載する
  - 実装場所（ファイルパス）
  - 要件・仕様
  - データ構造の定義
  - 注意点
  - 参考セクション（spec.mdへの参照）
  - 完了条件

### Implementation Flow

Issue番号が指定されたら、以下の手順で実装を行う:

1. **Issueの確認**: 指定されたIssue番号の内容を `gh issue view <番号>` で確認
2. **ブランチ作成**: `git checkout -b feature/<issue番号>-<簡潔な説明>`
3. **TDDで実装**: テストを先に作成し、実装を進める
4. **コミット**: 適切な粒度でコミット
5. **PR作成**: `gh pr create` でプルリクエストを作成
   - PRの説明にはIssueへの参照を含める（`Closes #<番号>`）
