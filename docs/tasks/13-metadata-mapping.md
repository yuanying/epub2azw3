# Task 13: メタデータマッピング

## 概要
EPUBのDublin CoreメタデータをMOBI/AZW3のEXTHレコードにマッピングする。Task 02で作成したEXTH構造体に対して、EPUBメタデータからの自動変換機能を追加する。

## 関連
- **spec.md参照**: §6.5.1
- **依存タスク**: Task 02（EXTHレコード生成 — 本タスクはメタデータ→EXTH変換ロジックを追加）
- **GitHub Issue**: —

## 背景
EPUBのメタデータ（Dublin Core要素）とMOBI/AZW3のEXTHレコードは異なる形式だが、ほぼ1対1でマッピング可能。適切な変換ルールを適用してEXTHレコードを生成する。

## 実装場所
- 更新ファイル: `internal/mobi/exth.go`（Task 02で作成したファイルを拡張）
- テストファイル: `internal/mobi/exth_test.go`（追加テスト）

## 要件

### マッピングテーブル

| Dublin Core | EXTHタイプ | 変換規則 |
|------------|-----------|---------|
| dc:title | 503 | そのまま |
| dc:creator (role=aut) | 100 | 複数著者は " & " で結合 |
| dc:publisher | 101 | そのまま |
| dc:description | 103 | そのまま |
| dc:identifier (scheme=ISBN) | 104 | ISBNのみ抽出 |
| dc:subject | 105 | 複数の場合は "; " で結合 |
| dc:date | 106 | YYYY-MM-DD 形式に変換 |
| dc:language | 524 | 言語コード（例: "ja"） |
| dc:rights | 109 | そのまま |

### 特殊な変換ルール
- **著者（100）**: `Creator` の `Role` が "aut" のもの。Roleが空の場合もauthor扱い。複数の場合 " & " で結合。
- **出版日（106）**: ISO 8601形式（`2023-01-15T00:00:00Z` 等）を `YYYY-MM-DD` に変換。日付のみの場合はそのまま。
- **ISBN（104）**: `Identifier` からISBN形式（10桁または13桁）を検出して抽出。
- **タイトル（503）**: `dc:title` の値をそのまま使用。
- **サブジェクト（105）**: 複数の `dc:subject` を "; " で結合。

### KF8必須レコード（Task 02と連携）
- タイプ121（KF8境界）: Task 02で処理済み
- タイプ125（レコード数）: Task 02で処理済み
- タイプ131（カバーオフセット）: Task 15（カバー画像）で処理

## データ構造
- 既存の `epub.Metadata` 構造体を入力とする
- 既存の `EXTHHeader` 構造体（Task 02）に対してレコードを追加

## 実装ガイドライン
- `epub.Metadata` → `[]EXTHRecord` の変換関数を提供
- 空のフィールドはスキップ（EXTHレコードを生成しない）
- 日付フォーマットの正規化は `time.Parse()` でパースし `2006-01-02` でフォーマット
- ISBN検出は正規表現（`\d{13}|\d{10}` パターン）
- Task 02の `EXTHHeader.AddRecord()` 等のAPIを使用

## テスト方針
- 全フィールドが設定されたメタデータの変換
- 空のメタデータの処理
- 複数著者の結合
- 日付フォーマットの変換（各種ISO 8601形式）
- ISBN抽出
- 複数サブジェクトの結合
- 日本語メタデータの正しいUTF-8エンコーディング

## 完了条件
- [ ] `epub.Metadata` → `[]EXTHRecord` 変換関数
- [ ] 各Dublin Coreフィールドの変換ルール実装
- [ ] 日付フォーマット正規化
- [ ] ISBN抽出ロジック
- [ ] 空フィールドのスキップ
- [ ] 全テストがパス
