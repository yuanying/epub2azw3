# Task 01: MOBIヘッダー生成

## 概要
PDBレコード0に配置するMOBIヘッダーを生成する。PalmDOCヘッダー（16バイト）、MOBIヘッダー本体（248バイト、KF8対応）、および完全タイトル（Full Name）を含むレコード0全体を構築する。

## 関連
- **spec.md参照**: §4.3（PalmDOCヘッダー, MOBIヘッダー本体, 完全タイトル）
- **依存タスク**: なし（Task 02 EXTHと組み合わせてレコード0を完成）
- **GitHub Issue**: #6

## 背景
MOBIヘッダーはAZW3ファイルの最初のレコード（レコード0）に配置され、ファイル全体の構造を定義する。PalmDOCヘッダー、MOBI識別子で始まるヘッダー本体、EXTHレコード（Task 02）、Full Nameで構成される。KF8-only形式ではMOBIタイプ248、ファイルバージョン8を設定する。

## 実装場所
- 新規ファイル: `internal/mobi/mobi_header.go`
- テストファイル: `internal/mobi/mobi_header_test.go`

## 要件

### PalmDOCヘッダー（16バイト）

| オフセット | サイズ | 内容 | 値 |
|-----------|-------|------|---|
| 0 | 2 | 圧縮タイプ | 1=無圧縮, 2=PalmDoc |
| 2 | 2 | 未使用 | 0x0000 |
| 4 | 4 | テキスト長 | 解凍後のテキストバイト数 |
| 8 | 2 | テキストレコード数 | ceil(テキスト長 / 4096) |
| 10 | 2 | 最大レコードサイズ | 4096 |
| 12 | 2 | 暗号化タイプ | 0（なし） |
| 14 | 2 | 未使用 | 0x0000 |

### MOBIヘッダー本体（248バイト、KF8対応）

| オフセット | サイズ | 内容 | 説明 |
|-----------|-------|------|------|
| 0 | 4 | 識別子 | "MOBI" (0x4D4F4249) |
| 4 | 4 | ヘッダー長 | 248（KF8対応） |
| 8 | 4 | MOBIタイプ | 248（KF8） |
| 12 | 4 | テキストエンコーディング | 65001（UTF-8） |
| 16 | 4 | 一意ID | ランダム値 |
| 20 | 4 | ファイルバージョン | 8（KF8） |
| 24-63 | 40 | 各種インデックス | 0xFFFFFFFF（未使用） |
| 64 | 4 | 最初のNon-Bookインデックス | 0xFFFFFFFF |
| 68 | 4 | Full Name Offset | MOBIヘッダー先頭からの相対オフセット |
| 72 | 4 | Full Name Length | タイトルのUTF-8バイト長 |
| 76 | 4 | Language コード | 言語コード（日本語=0x0411、英語=0x0409） |
| 80 | 4 | 最初の画像インデックス | 最初の画像レコード番号 |
| 84-99 | 16 | HUFF関連 | 0xFFFFFFFF / 0x00000000 |
| 100 | 4 | EXTHフラグ | 0x40 |
| 104-135 | 32 | 未使用/DRM | 0x00000000 / 0xFFFFFFFF |
| 136-159 | 24 | DRM/未使用 | 0xFFFFFFFF / 0x00000000 |
| 160 | 2 | 最初のコンテンツレコード | 通常1 |
| 162 | 2 | 最後のコンテンツレコード | テキストレコード数 |
| 164 | 4 | 未使用 | 0x00000001 |
| 168 | 4 | FCIS レコード番号 | FCISレコードの番号 |
| 172 | 4 | FCIS レコード数 | 0x00000001 |
| 176 | 4 | FLIS レコード番号 | FLISレコードの番号 |
| 180 | 4 | FLIS レコード数 | 0x00000001 |
| 184-207 | 24 | 未使用 | 0x00000000 / 0xFFFFFFFF |
| 208 | 4 | Extra record data flags | KF8フラグ |
| 212 | 4 | INDXレコードオフセット | 0xFFFFFFFF |
| 216-235 | 20 | KF8未使用 | 0xFFFFFFFF |
| 236 | 4 | FDST flow count | FDSTフローの数 |
| 240 | 4 | FDST開始オフセット | FDSTレコードのオフセット |
| 244 | 4 | 未使用 | 0 |

### Full Name

- EXTHレコードの後に配置
- 可変長UTF-8文字列
- 4バイト境界にパディング
- Full Name OffsetはMOBIヘッダー先頭（"MOBI"識別子）からの相対位置

### Record 0 内部レイアウト
```
[0..15]      PalmDOCヘッダー (16 bytes)
[16..N]      MOBIヘッダー ("MOBI"から始まる248バイト)
[N+1..M]     EXTHレコード (Task 02で生成、4バイトアライメント含む)
[M+1..M+L]   Full Name (可変長UTF-8文字列)
[M+L+1..]    パディング（4バイト境界）
```

## データ構造

### MOBIHeader 構造体
- Compression: uint16 — 圧縮タイプ
- TextLength: uint32 — 解凍後テキストバイト数
- TextRecordCount: uint16 — テキストレコード数
- MaxRecordSize: uint16 — 最大レコードサイズ（4096固定）
- Encoding: uint32 — テキストエンコーディング（65001=UTF-8）
- UniqueID: uint32 — ランダム一意ID
- FileVersion: uint32 — ファイルバージョン（8=KF8）
- MOBIType: uint32 — MOBIタイプ（248=KF8）
- FullNameOffset: uint32 — タイトル文字列オフセット
- FullNameLength: uint32 — タイトルバイト長
- LanguageCode: uint32 — 言語コード
- FirstImageIndex: uint32 — 最初の画像レコード番号
- EXTHFlags: uint32 — EXTHフラグ（0x40）
- FirstContentRecord: uint16 — 最初のテキストレコード番号
- LastContentRecord: uint16 — 最後のテキストレコード番号
- FCISRecordNumber: uint32 — FCISレコード番号
- FLISRecordNumber: uint32 — FLISレコード番号
- FDSTFlowCount: uint32 — FDSTフロー数
- FDSTOffset: uint32 — FDSTレコードオフセット

### LanguageCode マッピング
言語コード文字列からMOBI言語コードへの変換マップが必要:
- "ja" → 0x0411
- "en" → 0x0409
- "de" → 0x0407
- "fr" → 0x040C
- その他: デフォルト 0x0409

## 実装ガイドライン
- `encoding/binary` + `binary.BigEndian` で全フィールドをビッグエンディアンで書き込む
- `crypto/rand` または `math/rand` で一意IDを生成
- Full Name Offsetは **MOBIヘッダー先頭（"MOBI"識別子）からの相対位置**。計算式は `MOBIヘッダー(248) + EXTHレコードサイズ`（PalmDOCヘッダー16バイトは含めない）
- 既存の `mobi.PDB` と連携してレコード0のデータを構築

## テスト方針
- PalmDOCヘッダーのバイト列が16バイトであること
- MOBIヘッダーの識別子が "MOBI" であること
- ヘッダー長が248バイトであること
- MOBIタイプが248（KF8）であること
- テキストエンコーディングが65001（UTF-8）であること
- Full Name Offset/Length が正しく計算されること
- 言語コード変換が正しいこと
- レコード0全体のバイナリ出力が期待通りであること

## 完了条件
- [x] PalmDOCヘッダー構造体と生成関数
- [x] MOBIヘッダー構造体と生成関数（248バイト、KF8フィールド含む）
- [x] Full Name の配置とパディング
- [x] 言語コード変換マップ
- [x] レコード0全体の構築関数（EXTH挿入ポイントを含む）
- [x] 全テストがパス
