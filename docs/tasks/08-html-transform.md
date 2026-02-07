# Task 08: HTML変換

## 概要
EPUB内のXHTMLをKindle互換のHTMLに変換する。HTML5タグの置換、不要な属性の削除、AozoraEpub3互換要素の保持を行う。

## 関連
- **spec.md参照**: §6.2.1
- **依存タスク**: なし（既存HTMLBuilderの拡張として実装）
- **GitHub Issue**: —

## 背景
KindleデバイスはHTML5の一部タグを完全にはサポートしていない。`article`, `section`, `aside` などのHTML5セマンティック要素を `div` に変換し、Kindle非対応の属性を削除する必要がある。ただしAozoraEpub3生成のEPUBで使用される `ruby`, `rt`, `rp`, `class="tcy"`, `class="upr"` などは保持する。

## 実装場所
- 新規ファイル: `internal/converter/html_transform.go`
- テストファイル: `internal/converter/html_transform_test.go`

## 要件

### タグ変換マップ

| HTML5タグ | 変換後 |
|----------|--------|
| article | `<div class="article">` |
| section | `<div class="section">` |
| aside | `<div class="aside">` |
| nav | `<div class="nav">` |
| header | `<div class="header">` |
| footer | `<div class="footer">` |
| figure | `<div class="figure">` |
| figcaption | `<p class="figcaption">` |

### 削除する属性
- `contenteditable`
- `draggable`
- `hidden`
- `spellcheck`
- `translate`
- `data-*`（全てのdata属性）

### 保持する属性（AozoraEpub3互換）
- `class`（`vrtl`, `tcy`, `upr` を含む全class）
- `id`
- `lang`
- `xml:lang`
- `dir`

### 保持する要素（AozoraEpub3互換）
- `ruby`, `rt`, `rp` — ルビ関連要素
- `span` — class属性付きのspan（`tcy`, `upr` など）

### Paperwhite表示の許容範囲
- `ruby` のルビが位置ずれする場合があるが本文読解を優先
- `tcy` が横中横として正しく表示されない場合でも本文読解を許容

### 変換アルゴリズム
1. goquery でHTMLをロード
2. 各ノードを走査
3. タグ名を確認して変換マップに基づき変換
4. 属性をフィルタリング（削除対象を除去）
5. 修正されたHTMLを出力

## 実装ガイドライン
- goquery の `Find()`, `Each()` でDOM走査
- タグ名の変更は goquery のノード操作で実現
- `class` 属性は既存値を保持しつつ、変換元のタグ名classを追加
- 既存の `HTMLBuilder` のパイプラインに統合可能な設計
- `epub.Content` の `Document` に対してin-placeで変換

## AozoraEpub3互換の注意点
- `ruby`, `rt`, `rp` は一切変換・削除しない
- `span class="tcy"` や `span class="upr"` を削除しない
- `epub:type` 属性は削除対象に含めない（セマンティック情報として有用）

## テスト方針
- HTML5タグ（article, section等）が正しくdivに変換されること
- figcaptionがpに変換されること
- class属性がマージされること（既存class + タグ名class）
- data-* 属性が削除されること
- ruby/rt/rp が保持されること
- span class="tcy" が保持されること
- id, lang, class, dir 属性が保持されること
- contenteditable, draggable 等が削除されること

## 完了条件
- [ ] HTML5タグ → Kindle互換タグ変換関数
- [ ] 属性フィルタリング関数
- [ ] AozoraEpub3互換要素の保持
- [ ] HTMLBuilder パイプラインへの統合
- [ ] 全テストがパス
