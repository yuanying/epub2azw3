# Task 17: CLI強化 & エラーハンドリング

## 概要
CLIの機能を拡充し、詳細なオプション、進捗表示、構造化エラーハンドリングを追加する。

## 関連
- **spec.md参照**: §5.3, §9.4
- **依存タスク**: Task 07（MVPパイプライン — 基本CLIの拡張）
- **GitHub Issue**: —

## 背景
Phase 1のMVPパイプラインで実装した最小CLIを拡張し、画像品質、ログレベル、Strictモードなどの詳細なオプションを追加する。エラーハンドリングを構造化し、回復可能なエラーと致命的エラーを区別する。

## 実装場所
- 更新ファイル: `cmd/epub2azw3/main.go`
- テストファイル: `cmd/epub2azw3/main_test.go`（必要に応じて）

## 要件

### CLIオプション

| オプション | 短縮 | 説明 | デフォルト |
|-----------|------|------|----------|
| `--output` | `-o` | 出力ファイルパス | 入力ファイル名.azw3 |
| `--quality` | `-q` | JPEG品質 (60-100) | 85 |
| `--max-image-size` | | 最大画像サイズ（KB） | 127 |
| `--max-image-width` | | 最大画像幅（px） | 600 |
| `--no-images` | | 画像を含めない | false |
| `--log-level` | `-l` | ログレベル（error/warn/info/debug） | info |
| `--strict` | | Strictモード（警告もエラー扱い） | false |
| `--verbose` | `-v` | 詳細出力 | false |

### 進捗表示
- 各ステージの開始・完了を表示
- 処理中の章数/総章数
- 処理中の画像数/総画像数
- 最終的なファイルサイズの報告

### エラーハンドリング

#### エラー分類
1. **致命的エラー**（変換中止）:
   - EPUBファイルが存在しない
   - ZIPアーカイブが破損
   - OPFが見つからない、または解析不可
   - 必須メタデータが欠落

2. **回復可能エラー**（警告を出して継続）:
   - 一部の画像が見つからない
   - CSSの構文エラー
   - NCXが存在しない（NAVから生成）

3. **許容エラー**（ログのみ）:
   - 未使用のリソース
   - メタデータの一部欠落

#### Strictモード
- 回復可能エラーも致命的エラーとして扱う
- すべての警告を収集して最後に一括表示

### ログ出力
- `log/slog` または `log` パッケージを使用
- レベル: ERROR, WARN, INFO, DEBUG
- フォーマット: `[LEVEL] context: message`

## データ構造

### CLIOptions
- OutputPath: string
- JPEGQuality: int
- MaxImageSize: int
- MaxImageWidth: int
- NoImages: bool
- LogLevel: string
- Strict: bool
- Verbose: bool

### ConvertError
- Level: ErrorLevel（Fatal/Recoverable/Acceptable）
- Context: string（発生箇所）
- Message: string
- Cause: error（元のエラー）

## 実装ガイドライン
- `cobra` のフラグ定義で各オプションを追加
- `CLIOptions` を `ConvertOptions`（Task 07）に変換してパイプラインに渡す
- エラー収集はスライスに蓄積、最後にまとめて表示
- Strictモードフラグはパイプラインの各ステージに伝播
- 進捗表示は標準エラー出力に出力（パイプライン対応）

## テスト方針
- 各CLIオプションが正しくパースされること
- デフォルト値が正しいこと
- 無効な値（品質0, 品質101等）のバリデーション
- Strictモードで回復可能エラーが中止すること
- 通常モードで回復可能エラーが継続すること

## 完了条件
- [ ] CLIオプション追加（cobra フラグ）
- [ ] オプションバリデーション
- [ ] 構造化エラーハンドリング（エラー分類）
- [ ] Strictモード実装
- [ ] 進捗表示
- [ ] ログレベル切り替え
- [ ] 全テストがパス
