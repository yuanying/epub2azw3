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

# Run linter
go tool golangci-lint run ./...
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

### HTML Integration Flow (converter ↔ epub)

HTMLBuilder は複数 XHTML ファイルを単一 HTML に統合する中核ロジック:

1. `epub.LoadContent()` — XHTML を goquery で解析、CSS/画像参照を収集、body/html 属性を抽出
2. `HTMLBuilder.AddChapter()` — 章 ID（ch01, ch02, ...）を割り当て、パスマッピングを保存
3. `HTMLBuilder.AddCSS()` / `AddChapterCSS()` — グローバル CSS はそのまま、章別 CSS は ID セレクタをネームスペース化
4. `HTMLBuilder.Build()` — 章ごとに `<div id="chXX">` でラップ、ID をネームスペース化、`<mbp:pagebreak/>` 挿入、リンク解決

**ID ネームスペース化**: HTML 内の `id="cover"` → `id="ch01-cover"` に変換。CSS 内の `#cover` → `#ch01-cover` も対応（`AddChapterCSS` 使用時）。色コード（`#333`）は `{}` ブロック内のため変換されない。

**パス正規化**: `filepath.Join()` + `filepath.ToSlash()` で常にスラッシュ区切り（EPUB 標準）。`joinPath`（opf.go）と `resolvePath`（content.go）の両方で統一。

### AozoraEpub3 互換性

日本語縦書き EPUB（AozoraEpub3 生成）との互換性を維持する:
- `class="vrtl"`, `class="tcy"`, `class="upr"` を削除・正規化しない
- `writing-mode`, `text-orientation` 等の CSS プロパティを禁止プロパティ扱いにしない
- `ruby`, `rt`, `rp` を保持
- `#kobo.*` フラグメントはリンク変換をスキップしそのまま保持
- 章 div に元の `<body>`/`<html>` の `class`/`dir`/`lang`/`xml:lang` 属性を引き継ぐ
- `page-progression-direction` をメタデータとして保持

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
6. **レビュー対応**: PRにレビューコメントがあった場合
   - 各修正は対応する元のコミットに対する `fixup` コミットとして作成
   - 例: `git commit --fixup=<元のコミットハッシュ>`
   - 複数の修正がある場合は、それぞれ個別のfixupコミットを作成
   - マージ前に `git rebase --autosquash` でfixupコミットを統合することも可能
7. **完了チェック**: タスクドキュメント（`docs/tasks/`）に完了条件チェックリストがある場合、実装完了時にチェックボックスを `[x]` に更新する

### Manual Testing

実装した機能を実際のデータで手動確認する場合、`cmd/test/` 配下にテストプログラムを作成する。

**テストプログラムのガイドライン:**
- ファイル名: `cmd/test/<機能名>/main.go`
- コメントに関連するIssueとPRを記載
- 使用方法をコメントに記述
- 確認すべき項目を明示的に出力

**例:**
```go
// Test program for EPUB ZIP reader functionality
// Related: Issue #1, PR #10
//
// Usage:
//   go run ./cmd/test/epub_reader/main.go <epub-file-path>
```

**実行例:**
```bash
# テストプログラムを実行
go run ./cmd/test/epub_reader/main.go ~/Downloads/sample.epub
```
