# void-cutter

複数人でオンライン収録されたポッドキャストの音声素材（WAV）に対し、以下の編集処理を自動で行うコマンドラインツールです。

## 機能

- **ラウドネス正規化**: Apple Podcastの推奨値（-16 LUFS）に準拠したラウドネス正規化
- **無音部分の自動カット**: 全音声トラックで共通する無音部分の自動検出・短縮
- **音声互換性チェック**: サンプルレート、チャンネル数などの互換性を自動検証
- **詳細なデバッグ情報**: 音声ファイルの詳細な分析情報の表示
- **テストモード**: 処理なしでファイルをコピーするテストモード

## 使用方法

```bash
void-cutter [OPTIONS] <input1.wav> <input2.wav> ...
```

## オプション

| オプション                | 短縮形 | デフォルト値 | 説明                                         |
| ------------------------- | ------ | ------------ | -------------------------------------------- |
| `--output-suffix`         | `-s`   | `_edited`    | 出力ファイル名に付与する接尾辞               |
| `--target-loudness`       | `-l`   | `-16.0`      | ターゲットとするラウドネス値 LUFS            |
| `--silence-threshold`     | `-t`   | `-50.0`      | 無音と判定する音量の閾値 dBFS（-120〜0）     |
| `--min-silence-duration`  | `-m`   | `500`        | 無音と判定する最小の連続時間（ミリ秒）       |
| `--keep-silence-duration` | `-k`   | `250`        | カット後に残す無音の長さ（ミリ秒）           |
| `--debug-info`            |        | `false`      | 音声ファイルの詳細なデバッグ情報を表示       |
| `--test-copy`             |        | `false`      | テストモード：処理せずにファイルをコピーのみ |

## 使用例

### 基本的な使用例

```bash
void-cutter a.wav b.wav c.wav
```

### パラメータをカスタマイズした使用例

```bash
void-cutter --silence-threshold -45 --min-silence-duration 700 a.wav b.wav c.wav
```

### デバッグ情報を表示

```bash
void-cutter --debug-info a.wav b.wav c.wav
```

### テストモード（処理なしでコピーのみ）

```bash
void-cutter --test-copy a.wav b.wav c.wav
```

## 技術仕様

- **言語**: Go 1.23.0
- **主要ライブラリ**:
  - `github.com/go-audio/audio`: 音声データ処理
  - `github.com/go-audio/wav`: WAVファイル読み書き
  - `github.com/spf13/cobra`: CLI フレームワーク
- **対応フォーマット**: WAV（16bit/24bit PCM）

## ビルド方法

```bash
go build -o void-cutter .
```

## 動作要件

- Go 1.23.0 以上
- WAV形式の音声ファイル（16bit または 24bit PCM）
