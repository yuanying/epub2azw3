# EPUB to AZW3 変換ツール - 完全独立実装仕様書

**バージョン**: 1.0  
**最終更新**: 2025-11-24  
**対象**: Claude Code による完全Go言語実装

---

## 目次

1. [プロジェクト概要](#1-プロジェクト概要)
2. [システム要件](#2-システム要件)
3. [EPUBフォーマット詳細仕様](#3-epubフォーマット詳細仕様)
4. [AZW3/MOBIフォーマット詳細仕様](#4-azw3mobiフォーマット詳細仕様)
5. [アーキテクチャ設計](#5-アーキテクチャ設計)
6. [実装詳細](#6-実装詳細)
7. [変換アルゴリズム](#7-変換アルゴリズム)
8. [テスト要件](#8-テスト要件)
9. [実装優先順位](#9-実装優先順位)
10. [参考資料](#10-参考資料)

---

## 1. プロジェクト概要

### 1.1 目的

EPUBフォーマットの電子書籍をAmazon Kindle互換のAZW3（KF8）フォーマットに変換するコマンドラインツールをGo言語で完全独立実装する。外部依存ツール（Calibre等）を使用せず、全機能をGoコードで実現する。

### 1.2 目標

- **完全独立性**: 外部変換ツールに依存しない
- **品質**: Kindle実機で正しく表示される高品質な出力
- **パフォーマンス**: 大容量EPUB（100MB+）も高速処理
- **保守性**: 明確な設計と十分なテスト
- **拡張性**: 将来の機能追加が容易

### 1.3 スコープ

**含まれる機能**:
- EPUB 2.0/3.0 の完全パース
- AZW3（KF8形式）の生成
- HTML/CSS の Kindle互換形式への変換
- 画像の最適化とリサイズ
- メタデータの変換
- 目次（NCX）の生成
- 埋め込みフォントのサポート

**含まれない機能**:
- DRM処理（入力・出力とも）
- KFX形式への変換（仕様非公開のため）
- MOBI7のみの生成（AZW3に統合）
- PDFやその他形式からの変換

### 1.4 技術選択理由

**Go言語を選択した理由**:
- 優れた標準ライブラリ（archive/zip, encoding/xml, encoding/binary）
- 高速な処理性能
- バイナリ配布が容易
- 並行処理によるパフォーマンス向上

**AZW3を選択した理由**:
- Amazonの現行標準（MOBI7は2023年に廃止）
- 2011年以降の全Kindleデバイス対応
- 十分な技術ドキュメントとリバースエンジニアリング情報
- HTML5/CSS3のサブセット対応

---

## 2. システム要件

### 2.1 開発環境

- **Go言語**: 1.21以上
- **OS**: Linux, macOS, Windows（クロスコンパイル対応）
- **メモリ**: 最小512MB、推奨1GB以上
- **ストレージ**: 作業用に入力ファイルサイズの3倍

### 2.2 依存ライブラリ

必須ライブラリのみ使用。以下を推奨:

1. **CLI フレームワーク**: `github.com/spf13/cobra`
2. **画像処理**: `github.com/disintegration/imaging`
3. **HTML解析**: `golang.org/x/net/html`, `github.com/PuerkitoBio/goquery`

※EPUB/MOBI処理は既存ライブラリを使わず自作実装

### 2.3 パフォーマンス目標

- 10MBのEPUB: 5秒以内
- 100MBのEPUB: 30秒以内
- メモリ使用量: 入力ファイルサイズの2倍以内
- 並行処理: 画像変換などCPU使用を最適化

---

## 3. EPUBフォーマット詳細仕様

### 3.1 EPUB構造の概要

EPUBは以下の3層構造を持つZIPアーカイブ:

1. **コンテナ層**: ZIPアーカイブとmimetypeファイル
2. **パッケージ層**: OPFファイル（メタデータ、マニフェスト、スパイン）
3. **コンテンツ層**: XHTML、CSS、画像などのリソース

### 3.2 必須ファイルとその役割

#### 3.2.1 mimetype ファイル

- **場所**: ZIPアーカイブのルート（最初のエントリ）
- **内容**: 固定文字列 `application/epub+zip`（改行なし）
- **圧縮**: 無圧縮で格納（ZIPのStored方式）
- **目的**: EPUBファイルであることの識別

**実装上の注意**:
- ZIPの最初のエントリとして読み取る
- 内容が正確に一致するか検証
- 圧縮されていないことを確認

#### 3.2.2 META-INF/container.xml

- **目的**: OPFファイルの場所を指定
- **必須要素**: `<rootfile>` 要素の `full-path` 属性

**XMLスキーマ概要**:
```
<container>
  └── <rootfiles>
       └── <rootfile full-path="OEBPS/content.opf" 
                      media-type="application/oebps-package+xml"/>
```

**実装要件**:
- XMLパーサーで `full-path` 属性を抽出
- 相対パス形式で保存（例: "OEBPS/content.opf"）
- 複数の `<rootfile>` がある場合は最初のもの、または `media-type` が正しいものを選択

#### 3.2.3 OPF（Open Package Format）ファイル

EPUBの中核となるメタデータファイル。3つの主要セクションで構成:

**A. メタデータセクション**

Dublin Core 要素を含む書籍情報:

- **dc:title**: 書籍タイトル（必須）
- **dc:creator**: 著者名（複数可、`role`属性で役割指定）
- **dc:language**: 言語コード（必須、例: "ja", "en"）
- **dc:identifier**: 一意識別子（必須、`id`属性でマーク）
- **dc:publisher**: 出版社
- **dc:date**: 出版日（ISO 8601形式）
- **dc:description**: 説明文
- **dc:subject**: カテゴリ/タグ（複数可）
- **dc:rights**: 著作権情報

EPUB 3.0 追加メタデータ:

- **meta 要素**: `property` 属性で拡張情報
  - `dcterms:modified`: 最終更新日時（必須）
  - `calibre:series`: シリーズ名
  - `calibre:series_index`: シリーズ番号

**実装要件**:
- 全てのDublin Core要素を抽出
- `xml:lang` 属性を考慮（多言語対応）
- EPUB 2.0と3.0の両方に対応
- meta要素の `property`, `refines` 属性を正しく処理

**B. マニフェストセクション**

全リソースファイルのリスト:

- **item 要素**: 各リソースを定義
  - `id`: 一意識別子（スパインから参照）
  - `href`: ファイルパス（OPFからの相対パス）
  - `media-type`: MIMEタイプ
  - `properties`: 特殊プロパティ（空白区切りで複数可）

**主要なメディアタイプ**:
- `application/xhtml+xml`: XHTML文書
- `text/css`: スタイルシート
- `image/jpeg`, `image/png`, `image/gif`: 画像
- `application/x-font-ttf`: TrueTypeフォント
- `application/vnd.ms-opentype`: OpenTypeフォント

**特殊プロパティ**:
- `nav`: ナビゲーション文書（EPUB 3.0）
- `cover-image`: カバー画像
- `mathml`: MathML含有
- `svg`: SVG含有
- `scripted`: JavaScriptを含む

**実装要件**:
- 全itemをマップ構造で保存（idをキー）
- hrefパスをOPFディレクトリからの相対パスとして正規化
- プロパティを解析してフラグとして保存
- カバー画像の特定（properties="cover-image" または meta要素での指定）

**C. スパインセクション**

読書順序の定義:

- **itemref 要素**: コンテンツの順序
  - `idref`: マニフェストitemのid参照
  - `linear`: "yes"（通常の流れ）または "no"（参照のみ）

**toc 属性**（EPUB 2.0）:
- NCXファイルへの参照（マニフェストid）

**実装要件**:
- itemref の順序を保持（配列/スライス）
- linear="no" のアイテムを識別
- toc属性からNCXを特定

#### 3.2.4 NCX（Navigation Control file for XML）

EPUB 2.0の目次ファイル。EPUB 3.0では任意だが、Kindle互換性のため重要。

**構造**:
```
<ncx>
  └── <head>
       └── <meta name="dtb:uid" content="..."/>
       └── <meta name="dtb:depth" content="2"/>
  └── <docTitle><text>タイトル</text></docTitle>
  └── <navMap>
       └── <navPoint id="..." playOrder="1">
            └── <navLabel><text>第1章</text></navLabel>
            └── <content src="chapter01.xhtml"/>
            └── <navPoint>（ネスト可能）</navPoint>
```

**重要な属性**:
- **playOrder**: グローバルな順序番号（1から開始、連番）
- **dtb:depth**: 目次の最大階層深度
- **src**: コンテンツファイルへの相対パス（フラグメント識別子可）

**実装要件**:
- 再帰的な navPoint 構造を完全に解析
- playOrder の順序を保持
- content の src を正規化
- 階層深度を計算

#### 3.2.5 EPUB 3.0 ナビゲーション文書

EPUB 3.0で導入されたHTMLベースの目次。NCXの後継。

**構造**（XHTMLの一部）:
```
<nav epub:type="toc">
  <h1>目次</h1>
  <ol>
    <li><a href="chapter01.xhtml">第1章</a>
      <ol>
        <li><a href="chapter01.xhtml#sec1">セクション1.1</a></li>
      </ol>
    </li>
  </ol>
</nav>
```

**実装要件**:
- `epub:type="toc"` の nav 要素を検出
- ol/li の階層構造を再帰的に解析
- a 要素の href とテキストを抽出
- 内部的にNCX相当のデータ構造に変換

### 3.3 コンテンツファイル（XHTML）

**ファイル特性**:
- XML形式の整形式（Well-formed）HTMLが必須
- DTD宣言は任意だが、XML宣言は推奨
- 名前空間: `http://www.w3.org/1999/xhtml`

**一般的な構造**:
```
<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE html>
<html xmlns="http://www.w3.org/1999/xhtml">
<head>
  <title>章タイトル</title>
  <link rel="stylesheet" href="../css/style.css"/>
</head>
<body>
  <h1>第1章</h1>
  <p>本文...</p>
</body>
</html>
```

**使用される主要HTML要素**:
- 構造: `html, head, body, div, span`
- 見出し: `h1, h2, h3, h4, h5, h6`
- テキスト: `p, blockquote, pre`
- 強調: `em, strong, b, i, u, s, sup, sub`
- リスト: `ul, ol, li, dl, dt, dd`
- テーブル: `table, thead, tbody, tr, th, td`
- リンク: `a`（href属性）
- 画像: `img`（src, alt属性）
- ブレーク: `br, hr`

**EPUB 3.0 の拡張**:
- HTML5要素: `article, section, aside, nav, header, footer, figure, figcaption`
- `epub:type` 属性: セマンティック情報

**実装要件**:
- XML/HTMLパーサーで解析（goquery推奨）
- CSSへのリンクを収集
- 画像srcを収集して参照解決
- 相対パスを正しく処理（baseディレクトリを考慮）

### 3.4 CSSスタイルシート

**一般的なプロパティ**:
- フォント: `font-family, font-size, font-weight, font-style`
- テキスト: `text-align, text-indent, line-height, letter-spacing`
- レイアウト: `margin, padding, border`
- 色: `color, background-color`
- 表示: `display, width, height`

**EPUB特有の注意点**:
- ページサイズが可変なため、絶対値よりも相対値を推奨
- `position: fixed, absolute` は読者環境によって動作が異なる
- メディアクエリのサポートは限定的

**実装要件**:
- CSS文字列をそのまま保存（パースは不要、文字列置換で対応）
- @import, url() での外部リソース参照を検出
- @font-face ルールを抽出

### 3.5 画像ファイル

**対応フォーマット**:
- JPEG: 写真に最適
- PNG: 透明度が必要な画像
- GIF: アニメーションは非推奨だが対応
- SVG: EPUB 3.0で対応（Kindle対応は限定的）

**一般的な配置**:
- `images/` または `Images/` ディレクトリ
- マニフェストで media-type 指定

**実装要件**:
- 全画像をメモリまたは一時ディレクトリに展開
- サイズ情報を取得（幅・高さ）
- フォーマット検証

### 3.6 フォントファイル

**対応フォーマット**:
- TrueType (.ttf)
- OpenType (.otf)
- WOFF/WOFF2（EPUB 3.0）

**埋め込み方法**:
- @font-face CSSルールで定義
- `font/` ディレクトリに配置が一般的

**実装要件**:
- フォントファイルのバイナリ保持
- @font-face ルールとの関連付け
- Kindle対応確認（TTF推奨）

### 3.7 パス解決のルール

**相対パス解決の基準**:
- OPFファイルからの相対パス: マニフェストの href
- XHTMLファイルからの相対パス: CSS, 画像などのリソース
- CSSファイルからの相対パス: フォント、画像

**実装要件**:
- 各ファイルの基準ディレクトリを追跡
- `../` による親ディレクトリ参照を正しく処理
- パスを正規化（重複スラッシュ、`.` の除去）
- ZIPアーカイブ内のパス区切りは `/`（OSに依存しない）

---

## 4. AZW3/MOBIフォーマット詳細仕様

### 4.1 フォーマット階層

AZW3（KF8）は以下の構造を持つ:

```
AZW3ファイル（.azw3）
├── PalmDB コンテナ
│   ├── PDBヘッダー（78バイト）
│   ├── レコードリスト
│   └── レコードデータ
│       ├── レコード0: MOBIヘッダー + EXTHレコード
│       ├── レコード1〜N: 圧縮テキスト
│       ├── レコードN+1〜M: 画像データ
│       └── KF8セクション（MOBI8）
```

### 4.2 PalmDB（Palm Database）構造

#### 4.2.1 PDBヘッダー（78バイト）

**バイトレイアウト**:

| オフセット | サイズ | 内容 | 説明 |
|---------|-------|-----|------|
| 0 | 32 | データベース名 | NULL埋めのASCII文字列 |
| 32 | 2 | 属性フラグ | 通常は 0x0000 |
| 34 | 2 | バージョン | 通常は 0x0000 |
| 36 | 4 | 作成日時 | Palmエポック秒（1904年1月1日起点） |
| 40 | 4 | 更新日時 | Palmエポック秒 |
| 44 | 4 | バックアップ日時 | 通常は 0x00000000 |
| 48 | 4 | 修正番号 | 通常は 0x00000000 |
| 52 | 4 | appInfo オフセット | 通常は 0x00000000（未使用） |
| 56 | 4 | sortInfo オフセット | 通常は 0x00000000（未使用） |
| 60 | 4 | タイプ | "BOOK" (0x424F4F4B) |
| 64 | 4 | クリエイター | "MOBI" (0x4D4F4249) |
| 68 | 4 | 一意シード | 通常は 0x00000000 |
| 72 | 4 | 次レコードリスト | 通常は 0x00000000 |
| 76 | 2 | レコード数 | 全レコードの総数 |

**実装要件**:
- データベース名: 書籍タイトルを使用（31バイトまで、NULLパディング）
- タイプ/クリエイター: 固定値を使用
- レコード数: 動的に計算
- 日時: 現在時刻をPalmエポックに変換
  - Palmエポック = 1904年1月1日 00:00:00 UTC
  - 計算式: `Unix秒 + 2082844800`

#### 4.2.2 レコードリスト

各レコードのエントリ（8バイト）:

| オフセット | サイズ | 内容 |
|---------|-------|-----|
| 0 | 4 | レコードのデータオフセット |
| 4 | 1 | 属性フラグ（通常 0x00） |
| 5 | 3 | 一意ID（0, 1, 2, ... と連番） |

**実装要件**:
- レコード数分のエントリを生成
- オフセットは PDBヘッダー + レコードリスト全体のサイズ から開始
- 各レコードのサイズを累積してオフセット計算
- 最後に2バイトのパディング（0x0000）を追加

### 4.3 MOBIヘッダー（レコード0の一部）

#### 4.3.1 PalmDOCヘッダー（16バイト）

MOBIヘッダーの前に配置:

| オフセット | サイズ | 内容 | 値 |
|---------|-------|-----|---|
| 0 | 2 | 圧縮タイプ | 1=無圧縮, 2=PalmDoc, 17480=HUFF/CDIC |
| 2 | 2 | 未使用 | 0x0000 |
| 4 | 4 | テキスト長 | 解凍後のテキストバイト数 |
| 8 | 2 | テキストレコード数 | 圧縮テキストのレコード数 |
| 10 | 2 | 最大レコードサイズ | 通常 4096バイト |
| 12 | 2 | 暗号化タイプ | 0=なし |
| 14 | 2 | 未使用 | 0x0000 |

**実装要件**:
- 圧縮タイプ: PalmDoc (2) を推奨（実装が簡単）
- テキスト長: HTMLを連結した総バイト数
- テキストレコード数: 総バイト数を4096で割った切り上げ
- HUFF圧縮は複雑なので初期は非対応でも可

#### 4.3.2 MOBIヘッダー本体

識別子 "MOBI" (0x4D4F4249) で始まる可変長ヘッダー:

**主要フィールド**（オフセットはMOBI識別子からの相対）:

| オフセット | サイズ | 内容 | 説明 |
|---------|-------|-----|------|
| 0 | 4 | 識別子 | "MOBI" (0x4D4F4249) |
| 4 | 4 | ヘッダー長 | このヘッダーのバイト数（通常232〜） |
| 8 | 4 | MOBIタイプ | 2=Mobipocket Book, 3=PalmDoc Book |
| 12 | 4 | テキストエンコーディング | 1252=CP1252, 65001=UTF-8（推奨） |
| 16 | 4 | 一意ID | ランダム値 |
| 20 | 4 | ファイルバージョン | 6（KF8対応は8） |
| ... | ... | ... | ... |
| 80 | 4 | 最初の画像インデックス | 最初の画像レコードの番号 |
| 84 | 4 | 最初のHUFFインデックス | HUFF/CDICの開始（未使用時は0xFFFFFFFF） |
| 88 | 4 | HUFFレコード数 | HUFF/CDICのレコード数（未使用時は0） |
| ... | ... | ... | ... |
| 108 | 4 | EXTHフラグ | bit 6がセット（0x40）でEXTHあり |
| ... | ... | ... | ... |
| 192 | 2 | DRMオフセット | 通常 0xFFFF（DRM未使用） |
| 194 | 2 | DRMカウント | 通常 0（DRM未使用） |
| ... | ... | ... | ... |

**KF8（MOBI8）追加フィールド**（オフセット200以降）:

| オフセット | サイズ | 内容 |
|---------|-------|-----|
| 208 | 4 | FDST開始 | FDSTレコードのオフセット |
| 212 | 4 | FDSTレコード数 | FDSTレコードの数 |
| ... | ... | ... |
| 242 | 4 | KF8境界 | KF8セクション開始のレコード番号 |
| ... | ... | ... |

**実装要件**:
- ヘッダー長: 最低232バイト、KF8対応は248バイト以上
- テキストエンコーディング: UTF-8 (65001) を使用
- 一意ID: 乱数生成器で生成
- ファイルバージョン: KF8対応なら 8
- EXTHフラグ: 0x50 (bit 6とbit 4をセット)
- 未使用フィールドは0でパディング

#### 4.3.3 完全タイトル（Full Name）

MOBIヘッダーの直後、EXTHの前に配置:

- 可変長のUTF-8文字列
- 長さはMOBIヘッダーのフィールドで指定（オフセット84）
- 書籍のフルタイトルを格納

**実装要件**:
- タイトルのUTF-8バイト列
- 長さを正確にMOBIヘッダーに記録

### 4.4 EXTHレコード（拡張メタデータ）

**構造**:
```
EXTHヘッダー（12バイト）
├── 識別子: "EXTH" (0x45585448)
├── ヘッダー長: 12 + 全レコード長の合計
├── レコード数: N
└── レコード配列（各レコード8+バイト）
     ├── レコードタイプ（4バイト）
     ├── レコード長（4バイト、この8バイトを含む）
     └── データ（可変長）
```

**主要なレコードタイプ**:

| タイプ | 内容 | データ形式 |
|-------|-----|----------|
| 100 | 著者 | UTF-8文字列 |
| 101 | 出版社 | UTF-8文字列 |
| 103 | 説明 | UTF-8文字列 |
| 104 | ISBN | UTF-8文字列 |
| 105 | サブジェクト | UTF-8文字列 |
| 106 | 出版日 | YYYY-MM-DD |
| 108 | 貢献者 | UTF-8文字列 |
| 109 | 権利 | UTF-8文字列 |
| 503 | 更新タイトル | UTF-8文字列 |
| 524 | 言語 | 言語コード（例: "ja", "en"） |

**KF8用の特殊レコード**:

| タイプ | 内容 | データ形式 |
|-------|-----|----------|
| 121 | KF8境界オフセット | 4バイト整数（MOBI7終了位置） |
| 125 | レコード数 | 4バイト整数 |
| 131 | カバーオフセット | 4バイト整数（画像レコード番号） |

**実装要件**:
- 各メタデータフィールドをEXTHレコードに変換
- レコード長: 8 + データバイト数
- 全体をパディング（4バイト境界に合わせる）
- KF8対応の場合、タイプ121と125を必ず含める

### 4.5 テキストレコード（HTML コンテンツ）

#### 4.5.1 圧縮方式

**PalmDoc圧縮（推奨）**:

シンプルなLZ77ベースの圧縮。各バイトまたはバイト列を以下のように扱う:

1. **非圧縮バイト** (0x01-0x08):
   - バイト値 N は、次の N バイトが非圧縮であることを示す
   - 続けて N バイトのリテラルデータ

2. **リテラルバイト** (0x09-0x7F):
   - そのまま出力

3. **スペース + リテラル** (0x80-0xBF):
   - 0x80を引いた値がリテラル文字
   - スペース (0x20) + リテラル文字 を出力

4. **後方参照** (0xC0-0xFF):
   - 2バイトシーケンス: `[高位バイト][低位バイト]`
   - 距離: `((高位バイト & 0x3F) << 8) | 低位バイト` の下位13ビット
   - 長さ: `(((高位バイト >> 6) & 0x03) + 3)`
   - 過去のバッファから「距離」だけ戻った位置から「長さ」バイトをコピー

**実装要件**:
- 各テキストレコードは最大4096バイト（圧縮後）
- HTML文字列を連結してバイト列化
- PalmDoc圧縮を適用
- レコードリストに追加

#### 4.5.2 HTML構造化

Kindleに送るHTMLは以下の形式:

```
<html>
<head>
  <guide>
    <reference type="toc" title="目次" href="#toc"/>
    <reference type="text" title="本文開始" href="#start"/>
  </guide>
</head>
<body>
  <mbp:pagebreak/>
  <div id="toc">
    <h1>目次</h1>
    <ul>
      <li><a href="#ch1">第1章</a></li>
      ...
    </ul>
  </mbp:pagebreak/>
  <div id="ch1">
    ... 本文 ...
  </div>
  <mbp:pagebreak/>
  ...
</body>
</html>
```

**重要なKindle特有要素**:
- `<mbp:pagebreak/>`: 改ページ（章の区切りに使用）
- `<guide>`: ナビゲーションポイント
- `<a filepos="...">`: ファイル内絶対位置へのリンク（オプション）

**実装要件**:
- 各章のXHTMLを抽出して結合
- `<mbp:pagebreak/>` を章の前後に挿入
- 相対リンクを同一文書内のアンカーに変換
- CSSはインラインスタイルまたは `<style>` タグ内に埋め込み

### 4.6 画像レコード

各画像は1つのレコードとして格納:

**実装要件**:
- JPEG推奨（Kindleの最良サポート）
- サイズ制限:
  - Kindle旧世代互換: 127KB (127*1024バイト) まで
  - KF8: 256KB まで
- 推奨サイズ: 幅600px、DPI 96
- そのままバイナリをレコードとして追加

**HTMLからの参照**:
```
<img src="kindle:embed:0001" alt="説明"/>
```

- `kindle:embed:XXXX`: レコード番号（4桁の16進数）
- 最初の画像レコード番号はMOBIヘッダーで指定

### 4.7 NCX（MOBI形式での目次）

**NCXレコードの構造**:

HTML形式の目次を生成:

```
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

**filepos の計算**:
- HTMLテキストのバイトオフセット
- 圧縮前の位置を指定
- 各章の開始タグの位置を記録

**実装要件**:
- EPUBのNCXまたはナビゲーション文書を変換
- 各エントリに filepos を計算して付与
- 階層構造を `<ul><li>` のネストで表現
- NCXレコードをテキストレコードの後ろに配置

### 4.8 INDX（インデックステーブル）

Kindleの目次やスケルトンに使用される高度な構造。

**実装の優先度**: 低（基本機能ではNCXで十分）

**概要**:
- バイナリ形式の索引データ
- タイトル、著者、目次エントリなどをインデックス化
- 検索やナビゲーションの高速化

**実装方針**:
- 初期バージョンでは省略可能
- NCXで基本的な目次は実現可能
- 将来的な拡張として検討

### 4.9 FDST（フローデータ）

KF8で使用される、テキストの論理的な流れを定義する構造。

**構造**:
- FDSTレコード: 各セクションの開始位置を記録
- 各エントリ: 4バイトのオフセット値

**実装要件**:
- 各章の開始位置（バイトオフセット）を記録
- MOBIヘッダーでFDSTレコードの位置と数を指定
- KF8セクションでは必須

### 4.10 KF8デュアルフォーマット構造

**レイアウト**:
```
PDB ヘッダー
├── MOBI7 セクション
│   ├── MOBIヘッダー（KF8境界を含む）
│   ├── テキストレコード（基本HTML）
│   └── 画像レコード
├── 境界マーカー（EXTHレコード121で指定）
└── KF8 セクション
    ├── MOBIヘッダー（KF8拡張）
    ├── テキストレコード（拡張HTML/CSS）
    ├── 画像レコード
    ├── FDSTレコード
    └── INDXレコード
```

**実装戦略**:
- MOBI7セクション: 基本的なHTMLを生成（古いデバイス用）
- KF8セクション: 同じ内容だが拡張機能を使用
- EXTHレコード121: KF8セクションの開始レコード番号を指定

**簡略化の選択肢**:
- KF8のみ生成（MOBI7省略）
  - 2011年以降のKindleのみサポート
  - 実装が簡単
  - 推奨アプローチ（現実的に問題なし）

---

## 5. アーキテクチャ設計

### 5.1 プロジェクト構造

```
epub2azw3/
├── cmd/
│   └── epub2azw3/
│       └── main.go              # エントリポイント
├── internal/                     # 内部パッケージ（非公開）
│   ├── epub/                     # EPUB処理
│   │   ├── reader.go            # ZIPアーカイブ読み込み
│   │   ├── container.go         # container.xml パース
│   │   ├── opf.go               # OPF パース
│   │   ├── ncx.go               # NCX パース
│   │   ├── content.go           # XHTML/CSS 読み込み
│   │   └── models.go            # EPUBデータ構造
│   ├── converter/                # 変換処理
│   │   ├── pipeline.go          # 変換パイプライン
│   │   ├── html.go              # HTML変換
│   │   ├── css.go               # CSS処理
│   │   ├── image.go             # 画像最適化
│   │   ├── metadata.go          # メタデータ変換
│   │   └── toc.go               # 目次変換
│   ├── mobi/                     # MOBI/AZW3 生成
│   │   ├── writer.go            # AZW3ファイル書き込み
│   │   ├── pdb.go               # PDB構造
│   │   ├── mobi_header.go       # MOBIヘッダー
│   │   ├── exth.go              # EXTH生成
│   │   ├── compression.go       # PalmDoc圧縮
│   │   ├── text_record.go       # テキストレコード生成
│   │   ├── image_record.go      # 画像レコード生成
│   │   ├── ncx_record.go        # NCXレコード生成
│   │   ├── fdst.go              # FDST生成
│   │   └── models.go            # MOBIデータ構造
│   └── util/                     # ユーティリティ
│       ├── path.go              # パス処理
│       ├── encoding.go          # エンコーディング
│       └── time.go              # 時刻変換
├── pkg/                          # 公開パッケージ（API）
│   └── epub2azw3/
│       └── convert.go           # 公開API
├── testdata/                     # テストデータ
│   ├── samples/                 # サンプルEPUB
│   └── expected/                # 期待される出力
├── go.mod
├── go.sum
├── README.md
└── LICENSE
```

### 5.2 データフローアーキテクチャ

**パイプラインステージ**:

```
入力 EPUB
    ↓
[Stage 1] EPUB 解析
    - ZIPアーカイブ展開
    - OPF, NCX, コンテンツ読み込み
    ↓
[Stage 2] 検証と正規化
    - HTML整形式確認
    - パス解決
    - リソース整合性確認
    ↓
[Stage 3] HTML変換
    - Kindle非対応タグの変換
    - 属性のクリーンアップ
    - 相対リンクの解決
    ↓
[Stage 4] CSS最適化
    - 非対応プロパティの削除
    - 単位変換 (px → em)
    - インライン化
    ↓
[Stage 5] 画像最適化
    - リサイズ
    - 形式変換（JPEG推奨）
    - 圧縮
    ↓
[Stage 6] 目次生成
    - NCX/NAV → MOBI NCX
    - filepos 計算
    ↓
[Stage 7] メタデータ変換
    - Dublin Core → EXTH
    ↓
[Stage 8] AZW3 生成
    - PDB構造構築
    - MOBI/EXTHヘッダー生成
    - テキスト圧縮
    - レコード配置
    - ファイル書き込み
    ↓
出力 AZW3
```

### 5.3 エラーハンドリング戦略

**エラー分類**:

1. **致命的エラー** (変換を中止):
   - EPUBファイルが存在しない
   - ZIPアーカイブが破損
   - OPFが見つからない、または解析不可
   - 必須メタデータが欠落

2. **回復可能エラー** (警告を出して継続):
   - 一部の画像が見つからない
   - CSSの構文エラー
   - NCXが存在しない（OPFから生成）

3. **許容エラー** (ログのみ):
   - 未使用のリソース
   - メタデータの一部欠落

**実装方針**:
- エラーを構造化（カスタムエラータイプ）
- コンテキスト情報を含める（ファイル名、行番号など）
- ログレベル: ERROR, WARN, INFO, DEBUG
- `--strict` フラグで動作を切り替え

### 5.4 並行処理設計

**並行化できる処理**:

1. **画像最適化** (最も効果的):
   - 各画像を独立して処理
   - ワーカープールパターン
   - CPU数に応じた並行度

2. **HTMLファイルの解析**:
   - 各XHTMLファイルを並行して読み込み
   - 依存関係がないため安全

3. **テキスト圧縮**:
   - 各テキストレコードを並行圧縮
   - 最終的に順序通りに結合

**同期が必要な処理**:
- OPF解析（他の処理の前提）
- レコード番号の割り当て（順序依存）
- ファイル書き込み（競合回避）

**実装パターン**:
```
// 疑似コード
func (c *Converter) OptimizeImages() error {
    imagesChan := make(chan *Image)
    resultsChan := make(chan *OptimizedImage)
    errChan := make(chan error)
    
    // ワーカー起動
    workers := runtime.NumCPU()
    for i := 0; i < workers; i++ {
        go imageWorker(imagesChan, resultsChan, errChan)
    }
    
    // 画像を送信
    go func() {
        for _, img := range c.images {
            imagesChan <- img
        }
        close(imagesChan)
    }()
    
    // 結果を収集
    // ...
}
```

### 5.5 メモリ管理戦略

**大容量EPUB対応**:

1. **ストリーミング処理**:
   - 全ファイルをメモリに展開せず、必要に応じて読み込み
   - ZIPリーダーのストリーミングAPI使用

2. **チャンク処理**:
   - HTMLコンテンツを4096バイトのチャンクに分割して圧縮
   - レコード単位で処理

3. **一時ファイルの活用**:
   - 大きな画像は一時ディレクトリに書き出し
   - 最終的に順次読み込んでレコード化

**メモリプロファイリング**:
- `runtime/pprof` を使用
- メモリリークの検出
- ベンチマークテストで検証

---

## 6. 実装詳細

### 6.1 EPUBパース実装

#### 6.1.1 ZIPアーカイブの扱い

**手順**:
1. `archive/zip` パッケージで開く
2. `mimetype` ファイルを最初に読み、内容確認
3. `META-INF/container.xml` を読み、OPFパスを取得
4. 全ファイルリストを作成（マップで管理）

**注意点**:
- ZIPパス区切りは `/`（OS依存しない）
- ファイル名は大文字小文字を区別
- 一部のEPUBでは余分な `./` がパスに含まれる（正規化が必要）

#### 6.1.2 XMLパース

**Go標準の `encoding/xml` を使用**:

基本パターン:
```
type Element struct {
    XMLName   xml.Name
    Attribute string `xml:"attr,attr"`
    Text      string `xml:",chardata"`
    Children  []Child `xml:"child"`
}
```

**注意点**:
- 名前空間の扱い（`xmlns` 属性）
- `xml:",any"` タグで任意の子要素を受け取る
- `xml:",attr"` で属性、`xml:",chardata"` でテキスト内容

#### 6.1.3 相対パス解決

**アルゴリズム**:
1. 基準ディレクトリを特定（OPFファイルのディレクトリ）
2. 相対パスを結合: `path.Join(baseDir, relativePath)`
3. `..` を正しく処理: `path.Clean()` を使用
4. ZIPエントリ名と照合

**実装要件**:
- 各リソースの基準パスを記録
- HTMLからのCSS参照、CSS内のフォント参照など、多段階の解決
- URLデコード（%エンコードされたパス）

#### 6.1.4 HTML/XHTML解析

**`golang.org/x/net/html` または `goquery` を使用**:

手順:
1. XHTMLファイルをバイト列として読み込み
2. HTMLパーサーで解析（XMLモードでも可）
3. DOMツリーとして操作
4. タグ、属性、テキストを抽出

**goquery の利点**:
- jQueryライクなセレクター
- DOM操作が容易
- HTML出力も簡単

### 6.2 HTML/CSS変換

#### 6.2.1 HTML変換ルール

**タグ変換マップ**:
```
HTML5 → Kindle互換
article → div class="article"
section → div class="section"
aside → div class="aside"
nav → div class="nav"
header → div class="header"
footer → div class="footer"
figure → div class="figure"
figcaption → p class="figcaption"
```

**削除する属性**:
- `contenteditable`
- `draggable`
- `hidden`
- `spellcheck`
- `translate`
- `data-*` (全て)

**変換アルゴリズム**:
1. goquery でHTMLをロード
2. 各ノードを走査（`Find()`, `Each()`）
3. タグ名を確認して変換
4. 属性をフィルタリング
5. 修正されたHTMLを出力

#### 6.2.2 CSS処理

**禁止プロパティの削除**:
- `position: fixed`, `position: absolute`
- `transform: *`
- `transition: *`
- `animation: *`
- 負のマージン: `margin: -10px` など

**単位変換**:
- `px` → `em` (1em = 16px として計算)
- `pt` → `em` (1em = 12pt)
- パーセント、em、rem はそのまま

**アルゴリズム**:
1. CSS文字列を読み込み
2. 正規表現で各プロパティを検出
3. 禁止プロパティを削除
4. 単位を変換
5. 結果の文字列を生成

**インライン化の判断**:
- 小さいCSS (<10KB): `<style>` タグに埋め込み
- 大きいCSS: 各HTML要素に `style` 属性として分散
- Kindleは外部CSSをサポートするが、インライン化が安全

#### 6.2.3 リンク処理

**相対リンクの変換**:
- `href="chapter02.xhtml"` → `href="#ch02"` (同一文書内アンカー)
- 各章にユニークなID (`id="ch01"`) を付与
- アンカーのターゲットを書き換え

**外部リンク**:
- `http://`, `https://` はそのまま保持
- Kindleは外部リンクをサポート（実験的）

### 6.3 画像最適化

#### 6.3.1 最適化手順

1. **デコード**: `image.Decode()` で読み込み
2. **リサイズ**:
   - 幅が600pxを超える場合、600pxにリサイズ
   - アスペクト比を維持
   - `imaging.Lanczos` を使用（高品質）
3. **形式変換**:
   - PNG → JPEG (透明度がない場合)
   - GIF → JPEG (アニメーションでない場合)
4. **圧縮**:
   - JPEG品質: 80-85
   - ファイルサイズが127KB超の場合、品質を下げて再圧縮
5. **メタデータ記録**:
   - 最終的なサイズ（幅・高さ・バイト数）
   - 元のファイル名との対応

#### 6.3.2 特殊ケース

**透明PNG**:
- 白背景を追加してJPEGに変換
- または PNG のまま保持（Kindle対応）

**SVG**:
- ラスタライズ（PNG/JPEG変換）
- または省略（オプション）
- Kindle の SVG サポートは限定的

**カバー画像**:
- 別途処理（高解像度を保持）
- 最小1000x625px、推奨2500x1600px
- JPEG品質90以上

### 6.4 目次生成

#### 6.4.1 NCX → MOBI NCX 変換

**アルゴリズム**:
1. EPUBのNCX/NAVを解析済みのデータ構造から読み込み
2. 各エントリに対してfileposを計算:
   - HTMLファイルのバイトオフセットを記録
   - フラグメント識別子（`#section1`）を考慮
3. HTML形式の目次を生成:
   ```
   <h1>目次</h1>
   <ul>
     <li><a filepos="1234">第1章</a></li>
     ...
   </ul>
   ```
4. ネストされた項目は `<ul>` の入れ子で表現

**filepos 計算の詳細**:
- 圧縮前のHTMLバイト位置
- 各章のHTMLを順次連結したものの累積オフセット
- `id` 属性の位置を正確に記録するため、HTMLをバイト単位で走査

#### 6.4.2 ナビゲーションポイント

**`<guide>` セクション**:
```
<guide>
  <reference type="toc" title="目次" href="#toc"/>
  <reference type="text" title="本文開始" href="#start"/>
  <reference type="cover" title="表紙" href="#cover"/>
</guide>
```

**主要なtype**:
- `toc`: 目次
- `text`: 本文開始
- `cover`: 表紙

**実装**:
- HTMLの `<head>` に埋め込み
- 各typeに対応するアンカーを配置

### 6.5 メタデータマッピング

#### 6.5.1 Dublin Core → EXTH

**マッピングテーブル**:

| Dublin Core | EXTHタイプ | 変換規則 |
|------------|-----------|---------|
| dc:title | 503 | そのまま |
| dc:creator (role=aut) | 100 | 複数著者は " & " で結合 |
| dc:publisher | 101 | そのまま |
| dc:description | 103 | そのまま |
| dc:identifier (scheme=ISBN) | 104 | ISBN のみ抽出 |
| dc:subject | 105 | 複数の場合は "; " で結合 |
| dc:date | 106 | YYYY-MM-DD 形式に変換 |
| dc:language | 524 | 言語コード（例: "ja"） |
| dc:rights | 109 | そのまま |

**特殊なマッピング**:
- EPUB 3.0 の `meta` 要素（`property` 属性付き）も考慮
- Calibre固有のメタデータ（シリーズ名など）も抽出可能

#### 6.5.2 カバー画像の特定

**検出方法** (優先順位順):
1. マニフェストの `properties="cover-image"`
2. メタデータの `<meta name="cover" content="...">`
3. ガイドの `<reference type="cover" ...>`
4. ファイル名パターン（"cover.jpg", "cover.png"）

**実装**:
- 複数の方法を試行
- 最初に見つかったものを使用
- 見つからない場合は警告

### 6.6 PalmDoc圧縮の実装

#### 6.6.1 圧縮アルゴリズム

**入力**: バイト配列（非圧縮テキスト）  
**出力**: バイト配列（圧縮データ）

**手順**:
1. 入力を走査
2. 各位置で最長一致を検索（最大2047バイト後方、最大10バイト長）
3. 一致が見つかれば後方参照を出力、なければリテラルを出力
4. 特殊ケース（スペース+文字）を検出して圧縮

**詳細なロジック**:

- **バイトごとに判定**:
  1. スペース (0x20) の後に文字が続く場合:
     - 1バイト (0x80 + 文字コード) に圧縮
  2. 過去2047バイト以内に3バイト以上の一致がある場合:
     - 2バイトの後方参照 (0xC0-0xFF)
  3. それ以外:
     - リテラルバイト (0x09-0x7F) または非圧縮マーカー (0x01-0x08)

- **バッファ管理**:
  - 直近2048バイトの履歴を保持（スライディングウィンドウ）
  - 効率的な検索のためハッシュテーブル使用

#### 6.6.2 実装の最適化

**高速化のテクニック**:
- ハッシュテーブルで過去の文字列位置を記録
- 3バイトのプレフィックスでハッシュ
- 衝突はリンクリストで管理

**メモリ効率**:
- 固定サイズのリングバッファ（2KB）
- ハッシュテーブルも固定サイズ

### 6.7 AZW3ファイル生成

#### 6.7.1 レコード構築手順

**全体の流れ**:
1. レコードリストを初期化（空のスライス）
2. レコード0を構築:
   - PalmDOCヘッダー
   - MOBIヘッダー
   - 完全タイトル
   - EXTHレコード
3. テキストレコードを追加（レコード1〜N）
4. 画像レコードを追加（レコードN+1〜M）
5. NCXレコードを追加
6. FDSTレコードを追加（KF8）
7. レコード番号とオフセットを計算
8. PDBヘッダーとレコードリストを生成
9. 全データをファイルに書き込み

#### 6.7.2 オフセット計算

**基準**:
- ファイル先頭 = オフセット 0
- PDBヘッダー = 78バイト
- レコードリスト = 8バイト × レコード数 + 2バイト（パディング）

**計算式**:
```
最初のレコードオフセット = 78 + (8 × レコード数) + 2

各レコードのオフセット:
  Record[i].Offset = Record[i-1].Offset + Record[i-1].Size
```

**実装**:
- 全レコードを先に構築
- 2回目のパスでオフセットを計算
- レコードリストに記録

#### 6.7.3 バイナリ書き込み

**`encoding/binary` を使用**:

```
import "encoding/binary"

// 4バイト整数をビッグエンディアンで書き込み
binary.Write(writer, binary.BigEndian, uint32(value))

// 2バイト整数
binary.Write(writer, binary.BigEndian, uint16(value))

// バイト配列
writer.Write([]byte("MOBI"))

// ゼロ埋め
writer.Write(make([]byte, paddingSize))
```

**注意点**:
- MOBIは**ビッグエンディアン**（ネットワークバイトオーダー）
- アライメント: 一部のフィールドは4バイト境界に配置
- パディング: 不足バイトは 0x00 で埋める

#### 6.7.4 検証と後処理

**生成後の検証**:
1. ファイルサイズが期待通りか
2. PDBヘッダーのマジックナンバー（"BOOK", "MOBI"）
3. レコード数とオフセットの整合性
4. EXTHレコードの総サイズ

**オプションの最適化**:
- 未使用レコードの削除
- レコードの順序最適化
- ファイルサイズの最小化

---

## 7. 変換アルゴリズム

### 7.1 HTML統合アルゴリズム

**目的**: 複数のXHTMLファイルを1つのHTML文書に統合

**手順**:
1. スパインの順序に従ってXHTMLファイルを読み込み
2. 各ファイルの `<body>` 内容のみを抽出
3. 各章の前に `<mbp:pagebreak/>` を挿入
4. 各章にユニークなID（`id="ch01"`, `id="ch02"`, ...）を付与
5. すべてを結合して単一の `<body>` に配置
6. CSSを `<style>` タグに統合または `style` 属性に埋め込み
7. 最終的なHTML構造を構築

**例**:
```
入力:
  chapter01.xhtml: <body><h1>第1章</h1><p>...</p></body>
  chapter02.xhtml: <body><h1>第2章</h1><p>...</p></body>

出力:
  <html>
  <head>...</head>
  <body>
    <div id="ch01">
      <mbp:pagebreak/>
      <h1>第1章</h1><p>...</p>
    </div>
    <div id="ch02">
      <mbp:pagebreak/>
      <h1>第2章</h1><p>...</p>
    </div>
  </body>
  </html>
```

### 7.2 リンク解決アルゴリズム

**目的**: 章間のリンクを同一文書内のアンカーに変換

**手順**:
1. 全てのリンク（`<a href="...">`)を検出
2. hrefを解析:
   - 絶対URL (`http://...`): そのまま保持
   - 相対パス (`chapter02.xhtml`): 対応する章IDに変換
   - フラグメント (`#section1`): そのまま保持
   - ファイル+フラグメント (`chapter02.xhtml#sec1`): `#ch02-sec1` に変換
3. 元のIDとの衝突を避けるため、プレフィックスを追加
4. リンク先が存在するか検証

**例**:
```
入力: <a href="chapter02.xhtml#section1">次の章へ</a>
出力: <a href="#ch02-section1">次の章へ</a>
```

### 7.3 CSS統合アルゴリズム

**戦略1: 単一の `<style>` タグに統合**

手順:
1. 全てのCSSファイルを読み込み
2. 各CSSを処理（禁止プロパティ削除、単位変換）
3. 全てを連結
4. `<style>` タグに配置

**戦略2: インラインスタイル化**

手順:
1. CSSパーサーで各ルールを解析
2. セレクターに一致するHTML要素を検出
3. プロパティを `style` 属性として追加
4. 外部CSSは削除

**推奨**: 戦略1（実装が簡単、Kindleも対応）

### 7.4 画像参照変換アルゴリズム

**目的**: HTML内の画像参照を `kindle:embed:XXXX` 形式に変換

**手順**:
1. HTMLから全ての `<img src="...">` を検出
2. src のパスを解決
3. 対応する画像レコード番号を特定
4. src を `kindle:embed:XXXX` 形式に書き換え
   - XXXX = レコード番号（4桁16進数、ゼロ埋め）
   - 例: レコード50 → `kindle:embed:0032` (16進で0x32)

**マッピングテーブル**:
- 画像ファイル名 → レコード番号
- レコード番号 = 最初の画像レコード + 画像インデックス

### 7.5 目次のfilepos計算アルゴリズム

**目的**: 各目次エントリの正確なバイト位置を計算

**手順**:
1. 統合されたHTML文字列を生成
2. 各章のID（`id="ch01"`）の開始位置を記録:
   - 文字列検索で `id="ch01"` を探す
   - その開始バイトオフセットを記録
3. フラグメント識別子（`#section1`）がある場合:
   - 該当章内で `id="section1"` を検索
   - バイトオフセットを記録
4. NCX生成時に各エントリにfilepos属性を付与

**注意点**:
- UTF-8エンコーディングでのバイトオフセット
- 圧縮前のテキストでの位置
- HTMLタグ自体も含めたオフセット

---

## 8. テスト要件

### 8.1 ユニットテスト

**対象コンポーネント**:

1. **EPUBパーサー**:
   - container.xml 解析
   - OPF 解析（メタデータ、マニフェスト、スパイン）
   - NCX 解析
   - 相対パス解決

2. **HTML/CSS変換**:
   - タグ変換ルール
   - 属性フィルタリング
   - CSS プロパティ削除
   - 単位変換

3. **画像処理**:
   - リサイズ
   - 形式変換
   - 圧縮
   - サイズ制限

4. **PalmDoc圧縮**:
   - 基本的な圧縮
   - 後方参照
   - スペース+文字のケース
   - 解凍との一致確認

5. **MOBI生成**:
   - PDBヘッダー
   - MOBIヘッダー
   - EXTHレコード
   - レコード配置

**テストデータ**:
- 最小限のEPUB（1章のみ）
- 複雑なEPUB（多数の章、画像、CSS）
- 異常なEPUB（壊れたXML、欠落ファイル）

### 8.2 統合テスト

**シナリオ**:

1. **基本的な変換**:
   - シンプルなEPUB → AZW3
   - 出力ファイルの存在確認
   - ファイルサイズの妥当性

2. **コンテンツ検証**:
   - テキスト内容の保持
   - 画像の存在
   - 目次の正確性

3. **メタデータ検証**:
   - タイトル、著者などの保持
   - 言語情報の正確性

4. **実機テスト** (手動):
   - Kindle Previewer での表示確認
   - 実際のKindleデバイスでの動作確認
   - アプリ（iOS/Android）での確認

### 8.3 パフォーマンステスト

**ベンチマーク**:
1. 小さいEPUB（1MB）: 処理時間 < 2秒
2. 中程度のEPUB（10MB）: 処理時間 < 10秒
3. 大きいEPUB（100MB）: 処理時間 < 60秒

**メモリ使用量**:
- 入力ファイルサイズの2倍以内

**並行処理効果**:
- 画像処理でCPU使用率 > 50%

### 8.4 回帰テスト

**テストスイート**:
- 既知の問題のあるEPUBファイル
- 過去のバグ修正のテストケース
- エッジケース（空の章、巨大な画像など）

**自動化**:
- CIパイプラインで実行
- コミットごとに全テストを実行
- カバレッジ目標: 80%以上

---

## 9. 実装優先順位

### 9.1 フェーズ1: 基本機能（MVP）

**目標**: 最小限の機能で動作する変換ツール

**実装項目**:
1. プロジェクト構造のセットアップ
2. CLI フレームワーク（Cobra）の統合
3. EPUBのZIP展開
4. OPF解析（メタデータ、マニフェスト、スパイン）
5. XHTMLファイルの読み込み
6. 基本的なHTML統合（改ページなし）
7. PDBヘッダーの生成
8. MOBIヘッダーの生成
9. テキストレコードの生成（無圧縮または基本圧縮）
10. AZW3ファイルの書き込み

**成果物**: 
- テキストのみのEPUBをAZW3に変換可能
- Kindle Previewerで開ける

**期間**: 2-3週間

### 9.2 フェーズ2: コンテンツ変換

**実装項目**:
1. HTML/CSS変換（タグ、属性、プロパティ）
2. 画像の読み込みと基本的な処理
3. 画像レコードの生成
4. 画像参照の変換（`kindle:embed:`）
5. CSSのインライン化または統合
6. 改ページの挿入
7. リンク解決（章間リンク）

**成果物**:
- 画像付きEPUBの変換
- レイアウトがある程度保持される

**期間**: 3-4週間

### 9.3 フェーズ3: メタデータと目次

**実装項目**:
1. EXTHレコードの生成
2. メタデータマッピング（Dublin Core → EXTH）
3. NCX解析
4. 目次生成（filepos計算を含む）
5. NCXレコードの生成
6. ナビゲーションポイント（`<guide>`）
7. カバー画像の特定と処理

**成果物**:
- メタデータが正しく表示される
- Kindleの目次が機能する

**期間**: 2-3週間

### 9.4 フェーズ4: 最適化と品質向上

**実装項目**:
1. PalmDoc圧縮の実装
2. 画像最適化（リサイズ、圧縮）
3. 並行処理（画像処理）
4. エラーハンドリングの強化
5. ログ出力の改善
6. 進捗表示
7. 詳細なオプション（`--quality`, `--max-image-size`など）

**成果物**:
- ファイルサイズの削減
- 処理速度の向上
- ユーザーフレンドリーなCLI

**期間**: 2-3週間

### 9.5 フェーズ5: 高度な機能（オプション）

**実装項目**:
1. KF8セクション（MOBI7との分離）
2. FDSTレコード
3. INDX（索引）
4. HUFF/CDIC圧縮
5. フォント埋め込み
6. SVG対応
7. MathML対応（数式）

**成果物**:
- 完全なKF8対応
- 高度なレイアウトのサポート

**期間**: 3-4週間

### 9.6 合計開発期間

- **最小限（フェーズ1-3）**: 7-10週間（2-2.5ヶ月）
- **完全版（フェーズ1-4）**: 9-13週間（2-3ヶ月）
- **全機能（フェーズ1-5）**: 12-17週間（3-4ヶ月）

**推奨アプローチ**: 
- フェーズ1-3を先に完成させてリリース
- フィードバックを得ながらフェーズ4-5を追加

---

## 10. 参考資料

### 10.1 技術仕様ドキュメント

**EPUB仕様**:
- EPUB 3.3 仕様: https://www.w3.org/TR/epub-33/
- EPUB 2.0.1 仕様: http://idpf.org/epub/201
- Open Packaging Format (OPF): http://idpf.org/epub/20/spec/OPF_2.0.1_draft.htm
- NCX仕様: http://www.daisy.org/z3986/2005/ncx/

**MOBIフォーマット**:
- MobileRead Wiki - MOBI: https://wiki.mobileread.com/wiki/MOBI
- MobileRead Wiki - KF8: https://wiki.mobileread.com/wiki/KF8
- MobileRead Wiki - PDB: https://wiki.mobileread.com/wiki/PDB
- Kindle File Format (Wikipedia): https://en.wikipedia.org/wiki/Kindle_File_Format

**PalmDoc圧縮**:
- MobileRead Wiki - PalmDoc: https://wiki.mobileread.com/wiki/PalmDoc
- 圧縮アルゴリズム詳細: https://wiki.mobileread.com/wiki/MOBI#PalmDOC_compression

### 10.2 参照実装

**KindleUnpack**:
- GitHubリポジトリ: https://github.com/kevinhendricks/KindleUnpack
- 用途: MOBI/AZW3の構造を理解するための逆変換ツール
- 言語: Python
- 参考になる点: レコード解析、EXTH解析、KF8検出

**Calibre**:
- 公式サイト: https://calibre-ebook.com/
- ソースコード: https://github.com/kovidgoyal/calibre
- 用途: 変換アルゴリズムの参考
- 言語: Python
- 参考になる点: HTML正規化、CSS処理、画像最適化

### 10.3 Goライブラリ

**推奨ライブラリ**:

1. **CLI**: `github.com/spf13/cobra`
   - ドキュメント: https://pkg.go.dev/github.com/spf13/cobra
   - サブコマンド、フラグ、ヘルプ生成

2. **画像処理**: `github.com/disintegration/imaging`
   - ドキュメント: https://pkg.go.dev/github.com/disintegration/imaging
   - リサイズ、形式変換、品質調整

3. **HTML解析**: `github.com/PuerkitoBio/goquery`
   - ドキュメント: https://pkg.go.dev/github.com/PuerkitoBio/goquery
   - jQueryライクなセレクター

4. **XMLパース**: 標準ライブラリ `encoding/xml`
   - ドキュメント: https://pkg.go.dev/encoding/xml

5. **ZIPアーカイブ**: 標準ライブラリ `archive/zip`
   - ドキュメント: https://pkg.go.dev/archive/zip

### 10.4 検証ツール

**EPUBCheck**:
- ダウンロード: https://www.w3.org/publishing/epubcheck/
- 用途: EPUB検証
- 入力EPUBの品質確認に使用

**Kindle Previewer**:
- ダウンロード: https://www.amazon.com/Kindle-Previewer/b?node=21381691011
- 用途: AZW3の表示確認
- 必須の検証ツール

**Kindle Textbook Creator**:
- Amazon公式ツール
- 用途: KF8形式の理解

### 10.5 コミュニティリソース

**MobileRead Forums**:
- URL: https://www.mobileread.com/forums/
- トピック: E-Readers, Kindle, MOBI/AZW3形式
- 用途: 技術的な質問、トラブルシューティング

**Stack Overflow**:
- タグ: [epub], [mobi], [kindle], [golang]
- 用途: 実装の具体的な問題

### 10.6 実装時の参考コード

**疑似コード例（最小限）**:

```go
// メインの変換関数（概念的な流れ）
func ConvertEPUBToAZW3(epubPath, outputPath string) error {
    // 1. EPUBを開く
    epub := OpenEPUB(epubPath)
    
    // 2. 解析
    opf := epub.ParseOPF()
    ncx := epub.ParseNCX()
    
    // 3. コンテンツ読み込み
    html := CombineHTML(opf.Spine)
    images := LoadImages(opf.Manifest)
    
    // 4. 変換
    html = ConvertHTML(html)
    images = OptimizeImages(images)
    
    // 5. MOBI生成
    mobi := NewMOBIWriter(outputPath)
    mobi.WriteHeader(opf.Metadata)
    mobi.WriteText(html)
    mobi.WriteImages(images)
    mobi.WriteNCX(ncx)
    mobi.Close()
    
    return nil
}
```

**注意**: 実際の実装は各ステップをさらに細分化する

---

## 11. 追加の実装ガイドライン

### 11.1 コーディング規約

**Go標準に従う**:
- `gofmt` でフォーマット
- `golint` でリント
- `go vet` で静的解析

**命名規則**:
- エクスポート関数: `PascalCase`
- プライベート関数: `camelCase`
- 定数: `PascalCase` または `UPPER_SNAKE_CASE`
- パッケージ名: 小文字、短く、明確

**コメント**:
- エクスポート関数には必ず godoc コメント
- 複雑なアルゴリズムには説明コメント
- TODOコメントで未実装部分をマーク

### 11.2 エラー処理

**原則**:
- エラーは即座に返す
- ラップして文脈を追加: `fmt.Errorf("failed to parse OPF: %w", err)`
- カスタムエラー型を定義（必要に応じて）

**ログ出力**:
- 標準エラー出力に書き込み
- レベル別（ERROR, WARN, INFO, DEBUG）
- 構造化ログを検討（`log/slog`）

### 11.3 パフォーマンスの考慮

**プロファイリング**:
- `pprof` を使用して性能分析
- ボトルネックを特定
- メモリアロケーションを最小化

**ベンチマーク**:
- `testing` パッケージでベンチマークを作成
- 各変換ステップの性能測定

### 11.4 セキュリティ

**入力検証**:
- ファイルパスのサニタイズ
- ZIPボム対策（サイズ制限）
- XMLエンティティ展開攻撃の防止

**リソース制限**:
- メモリ使用量の上限設定
- タイムアウト設定

### 11.5 クロスプラットフォーム対応

**パス処理**:
- `filepath` パッケージを使用（OSに依存しない）
- ZIPアーカイブ内は `/` 固定

**改行コード**:
- 出力は LF (\n) 推奨
- Windows互換性も考慮

**バイナリ配布**:
- `go build` でクロスコンパイル
- Linux, macOS, Windows用のビルド

---

## 12. まとめ

### 12.1 重要なポイント

1. **EPUB → AZW3 変換の本質**:
   - ZIPアーカイブの展開と再構築
   - XMLメタデータの変換
   - HTMLの正規化とKindle互換化
   - バイナリフォーマット（PDB/MOBI）の生成

2. **最も複雑な部分**:
   - PalmDoc/HUFF圧縮の実装
   - filepos 計算の正確性
   - バイナリデータの正確な配置

3. **成功の鍵**:
   - MobileRead Wiki を徹底的に参照
   - KindleUnpack で生成されたファイルを解析して学習
   - 実際のKindleデバイスで頻繁にテスト
   - 小さく始めて段階的に機能追加

### 12.2 Claude Code への指示

**実装開始時**:
1. この仕様書を全体的に理解
2. フェーズ1から順番に実装
3. 各コンポーネントを独立してテスト
4. 実装中に疑問点があれば仕様書を参照
5. 不明な点があれば質問

**推奨される実装順序**:
1. プロジェクト構造とCLI
2. EPUBのZIP展開とOPF解析
3. 基本的なMOBI生成（無圧縮）
4. Kindle Previewerで動作確認
5. HTML変換と画像処理
6. 圧縮の実装
7. 最適化

**デバッグのヒント**:
- 各ステージの出力を一時ファイルに保存
- バイナリデータを16進ダンプで確認
- KindleUnpack で生成したAZW3を逆変換して比較

---

## 13. 技術的詳細の補足

### 13.1 Palmエポック時刻の変換

**定義**:
- Palmエポック: 1904年1月1日 00:00:00 UTC
- UNIXエポック: 1970年1月1日 00:00:00 UTC
- オフセット: 2082844800秒（66年分）

**変換式**:
```
PalmTime = UnixTime + 2082844800
```

**Go実装例（概念）**:
```go
palmEpochOffset := int64(2082844800)
now := time.Now().Unix()
palmTime := uint32(now + palmEpochOffset)
```

### 13.2 ビッグエンディアンの扱い

**MOBIフォーマットはビッグエンディアン**:
- 最上位バイトが先（ネットワークバイトオーダー）
- `binary.BigEndian` を使用

**例**:
```go
// 0x12345678 を書き込む場合
// バイト列: [0x12, 0x34, 0x56, 0x78]

value := uint32(0x12345678)
binary.Write(writer, binary.BigEndian, value)
```

### 13.3 文字列エンコーディング

**MOBI/AZW3はUTF-8**:
- MOBIヘッダーで UTF-8 (65001) を指定
- 全テキストをUTF-8エンコーディングで保存
- バイト長計算に注意（文字数 ≠ バイト数）

**Go の文字列**:
- デフォルトでUTF-8
- `len(str)` はバイト数を返す
- `utf8.RuneCountInString(str)` で文字数

### 13.4 NULL文字列のパディング

**PDBデータベース名**:
- 31バイトまで使用可能（32バイト目はNULL）
- 短い名前はNULLバイトで埋める

**例**:
```go
name := "My Book"
nameBytes := make([]byte, 32)
copy(nameBytes, []byte(name))
// 残りは自動的に0で初期化される
writer.Write(nameBytes)
```

### 13.5 4バイト境界アライメント

**一部のフィールドは4バイト境界に配置**:
- EXTHレコードの終了後
- 一部のヘッダーフィールド

**パディング計算**:
```go
size := len(data)
padding := (4 - (size % 4)) % 4
writer.Write(make([]byte, padding))
```

---

## 14. トラブルシューティングガイド

### 14.1 一般的な問題と解決策

**問題**: Kindle Previewerでファイルが開けない
- **原因**: PDBヘッダーまたはMOBIヘッダーの不正
- **解決**: マジックナンバー（"BOOK", "MOBI"）を確認、レコード数を検証

**問題**: テキストが文字化けする
- **原因**: エンコーディングの不一致
- **解決**: UTF-8を使用、MOBIヘッダーでエンコーディングを正しく設定

**問題**: 画像が表示されない
- **原因**: 画像参照の誤り、レコード番号の不一致
- **解決**: `kindle:embed:` の番号を検証、画像レコードの配置確認

**問題**: 目次が機能しない
- **原因**: filepos の誤り
- **解決**: バイトオフセットを再計算、圧縮前のテキストで確認

**問題**: ファイルサイズが巨大
- **原因**: 圧縮が適用されていない、画像が最適化されていない
- **解決**: PalmDoc圧縮を実装、画像をリサイズ・圧縮

### 14.2 デバッグテクニック

**16進ダンプでバイナリ確認**:
```bash
hexdump -C output.azw3 | head -n 50
```

**KindleUnpackで逆変換**:
```bash
python KindleUnpack.py output.azw3
```

**差分比較**:
- Calibreで生成したAZW3と自分のAZW3を比較
- バイナリ差分ツール（`vbindiff`, `Beyond Compare`）

**ログ出力**:
- 各ステージでの中間結果をログ
- デバッグモードでバイト数、オフセット、レコード数を出力

---

## 15. 結論

この仕様書は、EPUBからAZW3への完全独立実装をClaude Codeで実現するための包括的なガイドです。

**重要な原則**:
1. 仕様を正確に理解する
2. 段階的に実装する（MVP → 機能追加）
3. 頻繁にテストする（特に実機テスト）
4. 参照実装を活用する（KindleUnpack, Calibre）
5. コミュニティリソースを活用する（MobileRead）

**実装の成功に向けて**:
- フェーズごとに動作を確認
- 問題が発生したら仕様書の該当箇所を再確認
- 不明点は参考資料を参照
- デバッグツールを活用

この仕様書が完全な実装の羅針盤となることを期待します。
