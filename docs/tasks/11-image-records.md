# Task 11: 画像レコード生成 & 参照変換

## 概要
EPUB内の画像をPDBレコードとして格納し、HTML内の画像参照を `kindle:embed:XXXX` 形式に変換する。

## 関連
- **spec.md参照**: §4.6, §7.4
- **依存タスク**: Task 01（最初の画像インデックスをMOBIヘッダーに記録）, Task 06（画像レコードのアセンブリ配置）
- **GitHub Issue**: —

## 背景
AZW3では画像はPDBレコードとして格納される。HTML内の `<img src="...">` はEPUB内の相対パスから `kindle:embed:XXXX` 形式（レコード番号の16進数4桁）に変換する必要がある。Phase 2では画像の最適化は行わず、元のバイナリをそのまま格納する（最適化はPhase 4のTask 16）。

## 実装場所
- 新規ファイル: `internal/mobi/image_record.go`
- テストファイル: `internal/mobi/image_record_test.go`

## 要件

### 画像レコードの仕様
- 各画像は1つのPDBレコードとして格納
- 画像データはそのままバイナリとしてレコードに追加
- JPEG, PNG, GIF をサポート

### 画像参照の変換
- HTML内: `<img src="images/photo.jpg">` → `<img src="kindle:embed:XXXX">`
- XXXX: レコード番号の4桁16進数（ゼロ埋め）
- レコード番号 = 最初の画像レコード番号 + 画像インデックス（0始まり）
- 例: 最初の画像レコードが50番目 → 最初の画像は `kindle:embed:0032` (0x0032 = 50)

### MOBIヘッダーとの連携
- 最初の画像インデックス（First Image Index）をMOBIヘッダーに設定
- 画像がない場合: 0xFFFFFFFF

### マッピングテーブル
- EPUB内画像パス → 画像インデックス → レコード番号 → kindle:embed値

## データ構造

### ImageRecord
- Data: []byte — 画像バイナリデータ
- OriginalPath: string — EPUB内の元パス
- MediaType: string — MIMEタイプ

### ImageMapper
- Images: []ImageRecord — 画像レコード配列
- PathToIndex: map[string]int — パス → インデックスのマッピング
- FirstRecordNumber: int — 最初の画像レコードのレコード番号

## 実装ガイドライン
- `epub.Reader` からマニフェストの画像アイテムを取得
- 画像パスの正規化（EPUB相対パスからの解決）
- HTMLBuilder が生成したHTMLに対して画像参照を変換
- 変換は `kindle:embed:XXXX` の文字列置換で実行
- スパイン内のXHTMLが参照する画像のみを対象（未参照画像もカバー画像用に保持）
- 画像レコードの順序は **マニフェストの出現順** を採用（HTML内の参照順で並べ替えない）

## テスト方針
- 画像パス → レコード番号のマッピングが正しいこと
- `kindle:embed:XXXX` の16進数フォーマットが正しいこと（4桁、ゼロ埋め）
- HTML内の img src が正しく変換されること
- 複数画像の参照が正しく変換されること
- 画像がない場合の処理
- 画像パスの正規化（相対パス解決）

## 完了条件
- [ ] 画像レコード生成関数
- [ ] 画像マッピングテーブル構築
- [ ] HTML内画像参照変換関数（`kindle:embed:XXXX` 形式）
- [ ] MOBIヘッダー用の最初の画像インデックス計算
- [ ] 全テストがパス
