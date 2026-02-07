# Task 12: NCX/NAVパース

## 概要
EPUB内のNCX（Navigation Control file for XML）とEPUB 3 NAVドキュメントをパースし、目次データ構造を生成する。NCXを優先し、NCXが存在しない場合のみNAVを使用する。

## 関連
- **spec.md参照**: §3.2.4, §3.2.5
- **依存タスク**: なし（既存のepubパッケージを拡張）
- **GitHub Issue**: —

## 背景
NCXはEPUB 2.0の目次ファイルで、Kindle互換性のために重要。EPUB 3.0ではNAVドキュメント（HTMLベース）が導入されたが、本プロジェクトではNCXを優先する方針。NCXが存在しない場合のみNAVからNCX相当の構造を生成する。

## 実装場所
- 新規ファイル: `internal/epub/ncx.go`
- テストファイル: `internal/epub/ncx_test.go`

## 要件

### NCXパース

#### NCX XML構造
```
<ncx>
  <head>
    <meta name="dtb:uid" content="..."/>
    <meta name="dtb:depth" content="2"/>
  </head>
  <docTitle><text>タイトル</text></docTitle>
  <navMap>
    <navPoint id="..." playOrder="1">
      <navLabel><text>第1章</text></navLabel>
      <content src="chapter01.xhtml"/>
      <navPoint>（ネスト可能）</navPoint>
    </navPoint>
  </navMap>
</ncx>
```

#### パース対象
- `head` > `meta`: `dtb:uid`, `dtb:depth` の値
- `docTitle` > `text`: 文書タイトル
- `navMap` > `navPoint`: 再帰的にネスト構造をパース
  - `id`: ナビゲーションポイントID
  - `playOrder`: グローバル順序番号
  - `navLabel` > `text`: エントリの表示テキスト
  - `content` の `src`: コンテンツファイルパス（フラグメント識別子を含む場合あり）
  - 子 `navPoint`: ネストされた目次エントリ

#### パス正規化
- `content` の `src` はNCXファイルからの相対パス
- 手順: まずNCXファイルの所在ディレクトリを基準に相対パスを解決し、その結果をOPFディレクトリ基準のパスに正規化
- フラグメント識別子（`#section1`）はパスとは別に保持

### NAVパース

#### NAV HTML構造
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

#### パース対象
- `epub:type="toc"` の `<nav>` 要素を検出
- `ol` > `li` の階層構造を再帰的に解析
- `a` 要素の `href` とテキスト内容を抽出

#### NAV → NCX相当構造への変換
- NAVの `ol/li` 構造を `NavPoint` のツリーに変換
- `playOrder` は出現順に自動採番
- `id` は `nav-N` 形式で自動生成

### プロジェクト方針
- NCXが存在する場合: NCXを使用
- NCXが存在しない場合: NAVから生成
- 両方存在する場合: NCXを優先

## データ構造

### NCX 構造体
- UID: string — dtb:uid
- Depth: int — dtb:depth
- DocTitle: string — 文書タイトル
- NavPoints: []NavPoint — トップレベルのナビゲーションポイント

### NavPoint 構造体
- ID: string — ナビゲーションポイントID
- PlayOrder: int — グローバル順序番号
- Label: string — 表示テキスト
- ContentPath: string — コンテンツファイルパス（フラグメントなし）
- Fragment: string — フラグメント識別子（あれば）
- Children: []NavPoint — ネストされた子ポイント

## 実装ガイドライン
- NCXは `encoding/xml` でパース
- NAVは `goquery` でHTMLとしてパース
- `epub.Reader` のファイル読み込み機能を使用してNCX/NAVファイルを取得
- パスの正規化は既存の `filepath.Join` + `filepath.ToSlash` パターンを使用
- フラグメント識別子の分離: `strings.SplitN(src, "#", 2)` で分割

## AozoraEpub3互換の注意点
- AozoraEpub3は `toc.ncx` と `nav.xhtml` の両方を含むことが多い → NCX優先
- `nav.xhtml` はデザイン用として扱い、目次データとしてはNCXを使用

## テスト方針
- 基本的なNCXパース（フラットな目次）
- ネストされたNCXパース（2-3階層）
- NAVパース（NCXが無い場合）
- パスの正規化が正しいこと
- フラグメント識別子の分離が正しいこと
- playOrderの順序が保持されること
- 空のNCXの処理
- NCXとNAVの両方が存在する場合、NCXが優先されること

## 完了条件
- [ ] NCXパース関数
- [ ] NAVパース関数（フォールバック用）
- [ ] NCX/NAV共通のデータ構造
- [ ] パス正規化
- [ ] NCX優先のフォールバックロジック
- [ ] 全テストがパス
