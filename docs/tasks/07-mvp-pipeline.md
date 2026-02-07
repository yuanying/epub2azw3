# Task 07: MVP変換パイプライン

## 概要
EPUB入力からAZW3出力までの最小限の変換パイプラインを構築する。既存のEPUBパーサー、HTMLBuilder、および新規のMOBI生成コンポーネントを統合する。

## 関連
- **spec.md参照**: §5.2, §9.1
- **依存タスク**: Task 01〜06（全Phase 1タスク）
- **GitHub Issue**: —

## 背景
Phase 1の最終タスク。個別に実装された各コンポーネントを統合し、テキストのみのEPUBをKindle Paperwhiteで開けるAZW3に変換する最小パイプラインを構築する。

## 実装場所
- 新規ファイル: `internal/converter/pipeline.go`
- 更新ファイル: `cmd/epub2azw3/main.go`
- テストファイル: `internal/converter/pipeline_test.go`

## 要件

### パイプラインステージ（MVP）
1. **EPUB解析**: `epub.NewReader()` でZIP展開、OPF読み込み
2. **コンテンツ読み込み**: スパイン順に `epub.LoadContent()` で各XHTML読み込み
3. **HTML統合**: `converter.HTMLBuilder` で単一HTMLに統合
4. **テキストレコード生成**: 統合HTMLを4096バイトブロックに分割
5. **メタデータ変換**: `epub.Metadata` → MOBIヘッダー + EXTH
6. **固定レコード生成**: FDST, FLIS, FCIS, EOF
7. **AZW3書き込み**: 全レコードをファイルに出力

### CLI（最小構成）
- コマンド: `epub2azw3 <input.epub> [output.azw3]`
- 出力ファイル名: 省略時は入力ファイル名の拡張子を `.azw3` に変更
- エラー時: エラーメッセージを標準エラーに出力、終了コード1

### エラーハンドリング（MVP）
- EPUBファイルが存在しない → 致命的エラー
- ZIPが破損 → 致命的エラー
- OPFが見つからない → 致命的エラー
- XHTMLの読み込みエラー → 警告を出して該当章をスキップ

## データ構造

### ConvertOptions 構造体
- InputPath: string — 入力EPUBファイルパス
- OutputPath: string — 出力AZW3ファイルパス

### Pipeline
- Options: ConvertOptions
- Convert() error — 変換実行

## 実装ガイドライン
- 既存パッケージ（`epub`, `converter`, `mobi`）の公開APIを使用
- 各ステージを独立した関数として実装（将来のフェーズでの拡張に備える）
- CLIは `cobra` を使用（既存の `cmd/epub2azw3/main.go` に統合）
- MVPではCSS変換、画像処理、目次生成は省略
- ログ出力: `log` パッケージで最小限の進捗表示

## テスト方針
- テストデータ（`testdata/test.epub`）を入力としたE2Eテスト
- 生成されたAZW3ファイルのPDBヘッダー検証
- MOBIヘッダーの主要フィールド検証
- テキストレコードの存在確認
- FLIS/FCIS/EOFレコードの存在確認

## 完了条件
- [ ] Pipeline構造体と Convert() 関数
- [ ] 全ステージの統合
- [ ] CLI コマンド（最小構成）
- [ ] `testdata/test.epub` からAZW3を生成できること
- [ ] 生成されたAZW3がバイナリ的に整合していること
- [ ] 全テストがパス
