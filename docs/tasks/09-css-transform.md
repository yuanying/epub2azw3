# Task 09: CSS変換

## 概要
EPUB内のCSSをKindle互換に変換する。禁止プロパティの削除、単位変換を行い、AozoraEpub3互換の縦書き系プロパティは保持する。

## 関連
- **spec.md参照**: §6.2.2, §6.2.3, §7.3
- **依存タスク**: なし（既存HTMLBuilderのCSS処理を拡張）
- **GitHub Issue**: —

## 背景
KindleデバイスはCSSの一部プロパティをサポートしていない。`position: fixed/absolute`, `transform`, `transition`, `animation`, 負のマージンなどは削除する必要がある。ただし、AozoraEpub3生成EPUBで使用される縦書き系プロパティ（`writing-mode`, `text-orientation` など）は明確に例外として保持する。

## 実装場所
- 新規ファイル: `internal/converter/css.go`
- テストファイル: `internal/converter/css_test.go`

## 要件

### 禁止プロパティ（削除対象）
- `position: fixed`
- `position: absolute`
- `transform: *`
- `transition: *`
- `animation: *`
- `animation-*: *`（全animation関連プロパティ）
- 負のマージン: `margin: -Npx`, `margin-top: -Npx` 等

### 保持するプロパティ（AozoraEpub3互換、禁止対象外）
- `writing-mode`, `-epub-writing-mode`, `-webkit-writing-mode`
- `text-orientation`, `-epub-text-orientation`, `-webkit-text-orientation`
- `text-combine-upright`, `-epub-text-combine`, `-webkit-text-combine`
- `text-emphasis-style`, `text-emphasis-position`
- `-epub-text-emphasis-style`, `-epub-text-emphasis-position`
- `-webkit-text-emphasis-style`, `-webkit-text-emphasis-position`
- `ruby-position`

### 単位変換
- `px` → `em` (1em = 16px)
- `pt` → `em` (1em = 12pt)
- `%`, `em`, `rem` はそのまま

### CSS統合戦略
- 全CSSを単一の `<style>` タグに統合（戦略1を採用）
- XHTML内の `<link>` 出現順を保持（AozoraEpub3互換）
- 同一セレクタの競合は後勝ち

### CSS名前空間化（既存HTMLBuilderとの連携）
- 既にHTMLBuilder.AddChapterCSS()でIDセレクタの名前空間化は実装済み
- 本タスクではCSS変換（禁止プロパティ削除、単位変換）に集中

## 実装ガイドライン
- CSSパーサーは不要。正規表現による文字列置換で対応
- 禁止プロパティの検出: `property:\s*value` パターンで正規表現マッチ
- 単位変換: `(\d+(?:\.\d+)?)(px|pt)` パターンで検出し変換
- 負のマージンの検出: `margin(-top|-right|-bottom|-left)?:\s*-` パターン
- AozoraEpub3互換プロパティはホワイトリストで管理
- 処理順序: 1) 禁止プロパティ削除 → 2) 単位変換
- 正規表現処理のため、コメントや文字列内の一致を完全には区別できない。CSSが破損するケースがあるため、対象は一般的なEPUB CSSに限定する

## AozoraEpub3互換の注意点
- 縦書き系プロパティのベンダープレフィックス版も全て保持
- `writing-mode: vertical-rl` 等の値も保持
- AozoraEpub3は複数CSSファイルを使用するため、統合順序を維持

## テスト方針
- `position: fixed` が削除されること
- `position: absolute` が削除されること
- `position: relative` は保持されること
- `transform`, `transition`, `animation` が削除されること
- 負のマージンが削除されること
- `writing-mode: vertical-rl` が保持されること
- `-epub-writing-mode` が保持されること
- `text-combine-upright` が保持されること
- px→em変換（16px → 1em, 32px → 2em）
- pt→em変換（12pt → 1em, 24pt → 2em）
- 複数CSSの統合順序が維持されること

## 完了条件
- [x] 禁止プロパティ削除関数
- [x] AozoraEpub3互換プロパティのホワイトリスト
- [x] 単位変換関数（px→em, pt→em）
- [x] CSS変換パイプライン（削除 → 変換）
- [x] 全テストがパス
