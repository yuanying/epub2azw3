# Task 15: カバー画像ハンドリング

## 概要
EPUBからカバー画像を特定し、AZW3のカバーとして設定する。複数の検出方法を優先順位付きで実装する。

## 関連
- **spec.md参照**: §6.5.2
- **依存タスク**: Task 11（画像レコード — カバー画像もレコードとして格納）, Task 02（EXTH — カバーオフセットをEXTHレコード131に設定）
- **GitHub Issue**: —

## 背景
Kindle端末のライブラリ画面でカバー画像を表示するために、AZW3ファイル内でカバー画像を特定してメタデータに記録する必要がある。EPUBではカバー画像の指定方法が複数存在するため、優先順位を付けて検出する。

## 実装場所
- 新規ファイル: `internal/converter/cover.go`
- テストファイル: `internal/converter/cover_test.go`

## 要件

### カバー画像検出方法（優先順位順）
1. **マニフェストの `properties="cover-image"`**（EPUB 3.0）
   - ManifestItem の Properties に "cover-image" を含むアイテム
2. **メタデータの `<meta name="cover" content="...">`**（EPUB 2.0）
   - Metadata.CoverID に対応するマニフェストアイテム
3. **ガイドの `<reference type="cover" ...>`**
   - OPFのguideセクション（現在未パース → パース追加が必要な場合あり）
4. **ファイル名パターン**
   - マニフェスト内で "cover" を含むファイル名（`cover.jpg`, `cover.png`, `Cover.jpeg` 等）

### AZW3でのカバー設定
- EXTHレコード タイプ131: カバー画像の画像レコード内オフセット（0始まりのインデックス）
  - 値 = カバー画像のレコード番号 - 最初の画像レコード番号
- カバー画像の推奨仕様:
  - 最小: 1000x625px
  - 推奨: 2500x1600px
  - フォーマット: JPEG
  - 品質: 90以上

## データ構造

### CoverInfo 構造体
- ManifestID: string — マニフェストアイテムID
- Href: string — ファイルパス
- MediaType: string — MIMEタイプ
- DetectionMethod: string — 検出方法（デバッグ用）

## 実装ガイドライン
- 既存の `epub.OPF` の Manifest と Metadata を使用
- 検出方法を優先順位順に試行し、最初に見つかったものを使用
- 見つからない場合は `nil` を返し、警告ログを出力
- EXTH レコード131の値は Task 11 の ImageMapper.PathToIndex を使用して計算
- Phase 4（Task 16）でカバー画像の最適化（リサイズ、品質調整）を実装

## AozoraEpub3互換の注意点
- AozoraEpub3はEPUB 3.0形式のため `properties="cover-image"` を使用することが多い
- `<meta name="cover">` も同時に存在する場合がある → 優先順位に従う

## テスト方針
- `properties="cover-image"` からの検出
- `<meta name="cover">` からの検出
- ファイル名パターンからの検出（"cover.jpg"）
- 複数の検出方法が該当する場合、優先順位が正しいこと
- カバー画像が存在しない場合の処理（nil + 警告）
- EXTHレコード131の値計算

## 完了条件
- [ ] カバー画像検出関数（優先順位付き）
- [ ] 各検出方法の実装
- [ ] EXTHレコード131の値計算
- [ ] カバー画像なし時の警告処理
- [ ] 全テストがパス
