# タスク一覧 — EPUB to AZW3 変換ツール

## プロジェクト概要

EPUB フォーマットの電子書籍を Amazon Kindle 互換の AZW3（KF8）フォーマットに変換するコマンドラインツール。Go 言語で完全独立実装し、外部依存ツール（Calibre 等）を使用しない。

### 完了済み機能

| パッケージ | ファイル | 機能 |
|-----------|---------|------|
| `internal/epub` | `reader.go` | EPUB ZIP アーカイブの読み込み |
| `internal/epub` | `opf.go` | OPF パース（メタデータ、マニフェスト、スパイン、NCXパス、page-progression-direction） |
| `internal/epub` | `content.go` | XHTML/CSS 読み込み、body/html 属性抽出 |
| `internal/epub` | `models.go` | EPUB データ構造定義 |
| `internal/converter` | `html.go` | HTML 統合（HTMLBuilder）、ID 名前空間化、リンク解決、CSS 統合 |
| `internal/mobi` | `pdb.go` | PDB ヘッダー・レコードリスト生成 |
| `internal/mobi` | `mobi_header.go` | MOBI ヘッダー生成 |
| `internal/mobi` | `exth.go` | EXTH メタデータレコード生成 |
| `internal/mobi` | `text_record.go` | テキストレコード分割（無圧縮） |
| `internal/mobi` | `fixed_records.go` | FLIS/FCIS/EOF 固定レコード生成 |
| `internal/mobi` | `fdst.go` | FDST フローデスクリプタ生成 |
| `internal/util` | — | パス処理、エンコーディング、Palm エポック時刻変換 |

## パッケージ構造

```
epub2azw3/
├── cmd/epub2azw3/          # CLI エントリポイント（Cobra）
├── internal/
│   ├── epub/               # EPUB 解析
│   │   ├── reader.go       # ZIP アーカイブ読み込み
│   │   ├── opf.go          # OPF パース
│   │   ├── ncx.go          # NCX パース（Task 12 で実装）
│   │   ├── content.go      # XHTML/CSS 読み込み
│   │   └── models.go       # データ構造
│   ├── converter/           # 変換処理
│   │   ├── html.go          # HTML 統合（HTMLBuilder）
│   │   ├── html_transform.go # HTML 変換（Task 08）
│   │   ├── css.go           # CSS 処理（Task 09）
│   │   ├── image.go         # 画像最適化（Task 16）
│   │   ├── cover.go         # カバー画像（Task 15）
│   │   ├── toc.go           # 目次生成（Task 14）
│   │   └── pipeline.go     # 変換パイプライン（Task 07）
│   ├── mobi/                # AZW3/MOBI 生成
│   │   ├── pdb.go           # PDB 構造（実装済み）
│   │   ├── mobi_header.go   # MOBI ヘッダー（Task 01）
│   │   ├── exth.go          # EXTH レコード（Task 02）
│   │   ├── text_record.go   # テキストレコード（Task 03）
│   │   ├── fixed_records.go # FLIS/FCIS/EOF（Task 04）
│   │   ├── fdst.go          # FDST（Task 05）
│   │   ├── compression.go   # PalmDoc 圧縮（Task 10）
│   │   ├── image_record.go  # 画像レコード（Task 11）
│   │   ├── ncx_record.go    # NCX レコード（Task 14）
│   │   └── writer.go        # AZW3 書き込み（Task 06）
│   └── util/                # ユーティリティ
└── pkg/epub2azw3/           # 公開 API
```

## 既存データ構造

### epub.OPF
メタデータ、マニフェスト（map[string]ManifestItem）、スパイン（[]SpineItem）、NCXPath、PageProgressionDirection を保持。

### epub.Metadata
Title, Creators([]Creator), Language, Identifier, Publisher, Date, Description, Subjects([]string), Rights, CoverID を保持。`Identifier` は EXTH 104 変換用途を優先し、ISBN が見つかる場合は ISBN 系の `dc:identifier` 値を優先的に保持する。

### converter.HTMLBuilder
複数 XHTML を単一 HTML に統合する中核コンポーネント。chapters, cssContent, chapterIDs のマッピングを管理。ID 名前空間化（`ch01-` プレフィックス）、`<mbp:pagebreak/>` 挿入、リンク解決、CSS IDセレクタ名前空間化を担当。

### mobi.PDB
PDBHeader（78 バイト固定）と RecordEntry の配列を保持。`NewPDB()` でレコードサイズから自動的にオフセットを計算。

## コーディング規約

- **TDD**: テストを先に作成し、実装を進める
- **ビッグエンディアン**: 全バイナリデータは `encoding/binary` + `binary.BigEndian`
- **UTF-8**: テキストエンコーディングは UTF-8（MOBI encoding type 65001）
- **パス正規化**: `filepath.Join()` + `filepath.ToSlash()` でスラッシュ区切り
- **AozoraEpub3 互換**: 縦書きクラス（`vrtl`, `tcy`, `upr`）、`writing-mode` 系 CSS、`ruby`/`rt`/`rp`、`#kobo.*` フラグメントを保持
- **KF8-only**: MOBI7 セクションは生成しない（2011年以降の Kindle のみサポート）

## フェーズ別タスク一覧

### Phase 1: MVP — Kindle Paperwhite で開ける AZW3

| # | タスク | 実装ファイル | 依存 | Issue |
|---|--------|------------|------|-------|
| [01](./01-mobi-header.md) | MOBI ヘッダー生成 | `internal/mobi/mobi_header.go` | — | #6 |
| [02](./02-exth-records.md) | EXTH レコード生成 | `internal/mobi/exth.go` | Task 01 | — |
| [03](./03-text-records.md) | テキストレコード生成（無圧縮） | `internal/mobi/text_record.go` | Task 01 | #7 |
| [04](./04-flis-fcis-eof.md) | FLIS/FCIS/EOF レコード生成 | `internal/mobi/fixed_records.go` | Task 01 | — |
| [05](./05-fdst.md) | FDST レコード生成 | `internal/mobi/fdst.go` | Task 01 | — |
| [06](./06-azw3-assembly.md) | AZW3 ファイルアセンブリ | `internal/mobi/writer.go` | Task 01-05 | #8 |
| [07](./07-mvp-pipeline.md) | MVP 変換パイプライン | `internal/converter/pipeline.go`, `cmd/epub2azw3/main.go` | Task 01-06 | — |

### Phase 2: コンテンツ変換

| # | タスク | 実装ファイル | 依存 |
|---|--------|------------|------|
| [08](./08-html-transform.md) | HTML 変換 | `internal/converter/html_transform.go` | — |
| [09](./09-css-transform.md) | CSS 変換 | `internal/converter/css.go` | — |
| [10](./10-palmdoc-compression.md) | PalmDoc 圧縮 | `internal/mobi/compression.go` | Task 03 |
| [11](./11-image-records.md) | 画像レコード生成 & 参照変換 | `internal/mobi/image_record.go` | Task 01, 06 |

### Phase 3: メタデータ & 目次

| # | タスク | 実装ファイル | 依存 |
|---|--------|------------|------|
| [12](./12-ncx-nav-parse.md) | NCX/NAV パース | `internal/epub/ncx.go` | — |
| [13](./13-metadata-mapping.md) | メタデータマッピング | `internal/mobi/exth.go`（拡張） | Task 02 |
| [14](./14-toc-generation.md) | TOC 生成 | `internal/converter/toc.go`, `internal/mobi/ncx_record.go` | Task 12, 03 |
| [15](./15-cover-handling.md) | カバー画像ハンドリング | `internal/converter/cover.go` | Task 11, 02 |

### Phase 4: 最適化 & 品質

| # | タスク | 実装ファイル | 依存 |
|---|--------|------------|------|
| [16](./16-image-optimization.md) | 画像最適化 | `internal/converter/image.go` | Task 11 |
| [17](./17-cli-enhancement.md) | CLI 強化 & エラーハンドリング | `cmd/epub2azw3/main.go` | Task 07 |
| [18](./18-concurrent-processing.md) | 並行処理 | 各パッケージ | Task 10, 16 |

### Phase 5: 高度な機能（オプション）

| # | タスク | 実装ファイル | 依存 |
|---|--------|------------|------|
| [19](./19-advanced-features.md) | 高度な機能 | 各パッケージ | Task 01-18 |

## 依存関係図

```
Phase 1 (MVP)
  Task 01 (MOBI Header) ──┬──→ Task 02 (EXTH)
                          ├──→ Task 03 (Text Records)
                          ├──→ Task 04 (FLIS/FCIS/EOF)
                          └──→ Task 05 (FDST)
                                    ↓
  Task 01-05 ──────────────→ Task 06 (AZW3 Assembly)
                                    ↓
  Task 01-06 ──────────────→ Task 07 (MVP Pipeline)

Phase 2 (Content)
  Task 08 (HTML Transform) ── 独立
  Task 09 (CSS Transform) ─── 独立
  Task 03 ──→ Task 10 (PalmDoc Compression)
  Task 01, 06 ──→ Task 11 (Image Records)

Phase 3 (Metadata & TOC)
  Task 12 (NCX/NAV Parse) ── 独立
  Task 02 ──→ Task 13 (Metadata Mapping)
  Task 12, 03 ──→ Task 14 (TOC Generation)
  Task 11, 02 ──→ Task 15 (Cover Handling)

Phase 4 (Optimization)
  Task 11 ──→ Task 16 (Image Optimization)
  Task 07 ──→ Task 17 (CLI Enhancement)
  Task 10, 16 ──→ Task 18 (Concurrent Processing)

Phase 5 (Advanced)
  Task 01-18 ──→ Task 19 (Advanced Features)
```

## spec.md 参照マップ

| spec.md セクション | 対応タスク |
|-------------------|-----------|
| §3.2.4 NCX | Task 12 |
| §3.2.5 NAV | Task 12 |
| §4.3 MOBI ヘッダー | Task 01 |
| §4.4 EXTH レコード | Task 02, 13 |
| §4.5 テキストレコード | Task 03, 10 |
| §4.6 画像レコード | Task 11 |
| §4.7 NCX（MOBI 形式） | Task 14 |
| §4.9 FDST | Task 05 |
| §4.10 FLIS/FCIS/EOF | Task 04 |
| §5.2 データフロー | Task 07 |
| §5.3 エラーハンドリング | Task 17 |
| §5.4 並行処理 | Task 18 |
| §6.2.1 HTML 変換 | Task 08 |
| §6.2.2 CSS 処理 | Task 09 |
| §6.3 画像最適化 | Task 16 |
| §6.4 目次生成 | Task 14 |
| §6.5.1 メタデータマッピング | Task 13 |
| §6.5.2 カバー画像 | Task 15 |
| §6.6 PalmDoc 圧縮 | Task 10 |
| §6.7 AZW3 生成 | Task 06 |
| §7.4 画像参照変換 | Task 11 |
| §7.5 filepos 計算 | Task 14 |
| §9.5 高度な機能 | Task 19 |
