# Task 14: TOC生成

## 概要
NCX/NAVデータからMOBI形式の目次（NCXレコード）を生成する。HTML形式の目次を構築し、各エントリにfilepos属性（バイトオフセット）を計算して付与する。

## 関連
- **spec.md参照**: §4.7, §6.4, §7.5
- **依存タスク**: Task 12（NCX/NAVパース）, Task 03（テキストレコード — filepos計算の基準となるHTML）
- **GitHub Issue**: —

## 背景
Kindleの目次は、HTML形式で記述されたNCXレコードとして格納される。各目次エントリには `filepos` 属性があり、圧縮前の連結HTMLバイト列における該当位置のバイトオフセットを指す。

## 実装場所
- 新規ファイル: `internal/converter/toc.go`
- 新規ファイル: `internal/mobi/ncx_record.go`
- テストファイル: `internal/converter/toc_test.go`
- テストファイル: `internal/mobi/ncx_record_test.go`

## 要件

### MOBI NCXレコードのHTML形式
```html
<html>
<body>
<h1>目次</h1>
<ul>
  <li><a filepos="12345">第1章</a></li>
  <li><a filepos="23456">第2章</a>
    <ul>
      <li><a filepos="23500">2.1節</a></li>
    </ul>
  </li>
</ul>
</body>
</html>
```

### filepos計算
- 基準: **最終的にテキストレコード生成に用いる「圧縮前の連結HTMLバイト列」**の先頭からのバイトオフセット
- 章IDの位置: `id="ch01"` のタグ開始位置のバイトオフセット
- フラグメント付き: `id="ch01-section1"` の位置（IDは名前空間化済み）
- UTF-8バイトオフセットで計算

### filepos計算アルゴリズム
1. HTMLBuilder.Build() で生成された統合HTMLに対し、**本文内HTML目次の挿入を行い最終HTMLを確定**
2. 最終HTMLバイト列を取得
3. 各目次エントリの対象ID（章ID or フラグメントID）を特定
4. 最終HTMLバイト列内でそのIDの出現位置を検索
5. バイトオフセットを記録
6. NCXレコードのHTMLを生成し、各 `<a>` に `filepos` を設定

### HTML目次（本文内）
- NCXからHTML目次を生成して本文の先頭付近に配置
- AozoraEpub3の `nav.xhtml` はデザイン用として扱い、本文内にはNCXから生成した目次を使用
- タイトルは `ncx.DocTitle` を使用
- **filepos計算は本文内HTML目次挿入後の最終HTMLに対して行う**

### ナビゲーションポイント（`<guide>` セクション）
```html
<guide>
  <reference type="toc" title="Table of Contents" filepos="00000XXX"/>
</guide>
```
- **NCXレコードHTMLの `<head>` に配置**
- `filepos` はインラインTOC div の開始位置のバイトオフセット

## データ構造

### TOCEntry 構造体
- Label: string — 表示テキスト
- FilePos: uint32 — バイトオフセット
- Children: []TOCEntry — ネストされたエントリ

### TOCGenerator
- NavPoints: []NavPoint（Task 12のデータ構造）
- ChapterIDs: map[string]string — ファイルパス→章ID
- HTMLBytes: []byte — 統合HTMLバイト列

## 実装ガイドライン
- `converter.HTMLBuilder` の出力に本文内HTML目次を挿入した **最終HTMLバイト列** を基準にfileposを計算
- filepos検索: `bytes.Index()` で `id="targetID"` パターンを検索
- NCXレコードは独立したPDBレコードとして格納し、**テキスト/画像レコードの後、FDSTレコードの前**に配置
- NCXの NavPoint.ContentPath をHTMLBuilderの chapterIDs マップで章IDに変換
- フラグメント付きの場合: `章ID-フラグメント` 形式のIDを検索
- ネスト構造は `<ul><li>` で表現

## テスト方針
- 基本的なfilepos計算（単一章）
- 複数章のfilepos計算
- フラグメント付きfilepos計算
- ネストされた目次のHTML生成
- fileposがバイトオフセットとして正しいこと（UTF-8考慮）
- `<guide>` セクションの生成
- 空の目次の処理

## 完了条件
- [x] filepos計算関数
- [x] NCXレコードHTML生成関数
- [x] HTML目次（本文内）生成関数
- [x] `<guide>` セクション生成
- [x] NavPoint → TOCEntry変換
- [x] 全テストがパス
