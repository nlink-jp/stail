# AGENTS.md — stail 開発ルール

このファイルは AI エージェントおよびこのプロジェクトに新しく参加する開発者向けのルール集です。
実際の開発中に判明した問題や決定事項を反映しています。

---

## プロジェクト構成

```
stail/
├── main.go                      # エントリポイント（最小限）
├── go.mod                       # Go 1.23, module: github.com/magifd2/stail
├── Makefile
├── internal/
│   ├── config/   # 設定ファイル管理・環境変数解決
│   ├── slack/    # Slack REST クライアント・Socket Mode クライアント・データモデル
│   ├── format/   # テキスト／JSON フォーマッタ
│   └── cmd/      # Cobra サブコマンド群
└── docs/
```

**パッケージの責務を越えない。** `cmd` パッケージのみが `os.Stdout` / `os.Stderr` に書く。
`slack` と `format` は `io.Writer` を受け取る純粋な関数・メソッドとして実装する。

---

## テスト設計ルール

### ネットワーク接続禁止

`httptest.NewServer` はテスト環境でポートバインドができないため**使用禁止**。
HTTP テストは `http.RoundTripper` を差し替えるモックで行う。

```go
// client_test.go のパターン
type mockTransport struct {
    handler func(path string, query map[string]string) (interface{}, error)
}
func (m *mockTransport) RoundTrip(req *http.Request) (*http.Response, error) { ... }
```

**注意:** `HTTPClient` がビルドする URL は `https://slack.com/api/<method>` 形式のため、
`RoundTrip` が受け取るパスは `/api/<method>` になる。
テストハンドラに渡す前に `/api` プレフィックスを除去すること。

```go
wrapped := func(path string, ...) {
    return handler(strings.TrimPrefix(path, "/api"), ...)
}
```

### WebSocket テスト

Socket Mode テストでは `gorilla/websocket` をインポートしない（本番バイナリのみ使用）。
`websocket.TextMessage` の代わりにローカル定数を使う。

```go
const wsTextMessage = 1  // websocket.TextMessage と同値
```

WebSocket 接続は `WsConn` インターフェース経由でモックする（`testWsConn` 参照）。

### テストパッケージ

`internal/slack/` のテストは `package slack_test`（外部テスト）で書く。
これにより未公開フィールドへのアクセスを禁じ、公開 API のみを検証できる。

型・関数をテストから参照する必要が出た場合は、`unexported` → `Exported` に変更する
（テストのためだけに内部構造を公開しない）。

### テスタビリティのための設計

依存を注入できる設計を最初から組み込む。後付けは困難。

| コンポーネント | 注入ポイント |
|---|---|
| `HTTPClient` | `WithTransport(rt)`, `WithBaseURL(u)` |
| `SlackSocketClient` | `WithDialFunc(fn)`, `WithOpenFunc(fn)`, `WithSocketBaseURL(u)` |

---

## エラーハンドリングルール

### `errors.Is` vs `os.IsNotExist`

`fmt.Errorf("%w", err)` でラップされたエラーには `os.IsNotExist()` が機能しない。
**必ず `errors.Is(err, os.ErrNotExist)` を使うこと。**

```go
// 誤り — ラップされたエラーをアンラップできない
if os.IsNotExist(err) { ... }

// 正しい
if errors.Is(err, os.ErrNotExist) { ... }
```

### エラーラップ

エラーは必ず `fmt.Errorf("context: %w", err)` でコンテキストを付けてラップする。
生のエラーをそのまま返さない。

---

## セキュリティルール

- 設定ファイルは **0600 パーミッション** で作成する（`config.Save` 参照）。
- トークンをログ・デバッグ出力・エラーメッセージに含めない。
- トークン入力には `golang.org/x/term` を使い、エコーを抑制する。
- ダウンロード URL (`url_private_download`) は時限 URL のため JSON エクスポートに含めない（`File.URLPrivateDownload` は `json:"-"`）。

---

## フラグ・設定の配線ルール

- **宣言したフラグは必ずコードに接続する。** 宣言だけして使われていないフラグはバグと見なす。
- グローバルフラグ（`--debug` 等）は `root.go` で宣言し、サブコマンドはパッケージ変数として参照する。
- フラグを機能コンポーネントに渡す場合は明示的なメソッド（例: `WithDebug(bool)`）で渡す。

---

## ドキュメント更新ルール

機能を追加・変更したら**必ず同時に**以下を更新する:

| ファイル | 更新内容 |
|---|---|
| `README.md` | 使用例・フラグ一覧 |
| `README.ja.md` | 日本語版（README.md と同期） |
| `CHANGELOG.md` | `[Unreleased]` セクションに追記 |

`docs/` 配下のファイル（`CONFIG_FORMAT.md` 等）は関連する場合のみ更新する。

---

## リリースワークフロー

CI/CD はなく、手動リリース。バージョンは [Semantic Versioning](https://semver.org/) に従う。

### バージョニング方針

| 変更種別 | バンプ例 |
|---|---|
| 新機能追加 | `0.1.2` → `0.1.3` |
| バグ修正 | `0.1.2` → `0.1.3` |
| 破壊的変更 | `0.1.x` → `0.2.0` |

### リリース手順

```bash
# 1. CHANGELOG.md の [Unreleased] セクションを新バージョンに確定
#    [Unreleased] → [0.1.3] - YYYY-MM-DD

# 2. 変更をコミット
git add <files>
git commit -m "chore: release vX.Y.Z"

# 3. タグを打つ
git tag vX.Y.Z

# 4. プッシュ
git push origin main
git push origin vX.Y.Z

# 5. 全プラットフォーム向けバイナリをビルド
make build-all
# → bin/stail-darwin-amd64, bin/stail-darwin-arm64,
#    bin/stail-linux-amd64, bin/stail-linux-arm64,
#    bin/stail-windows-amd64.exe

# 6. GitHub Release を作成してバイナリをアップロード
gh release create vX.Y.Z --title "vX.Y.Z" --notes "..."
gh release upload vX.Y.Z bin/stail-darwin-amd64 bin/stail-darwin-arm64 \
  bin/stail-linux-amd64 bin/stail-linux-arm64 bin/stail-windows-amd64.exe
```

### チェックリスト

- [ ] `CHANGELOG.md` の `[Unreleased]` → `[X.Y.Z] - YYYY-MM-DD` に確定
- [ ] `README.md` / `README.ja.md` に新機能・フラグを追記済み
- [ ] `make test` がすべて PASS
- [ ] タグ・コミット・バイナリ・GitHub Release を確認

---

## ビルドルール

```bash
go build ./...   # コンパイルチェックのみ。バイナリは更新されない
make build       # ./bin/stail を生成（動作確認はこちら）
make build-all   # 全プラットフォーム向けクロスコンパイル
make test        # 全テスト実行
```

**サンドボックス・制限環境では環境変数を指定する:**

```bash
GOCACHE=$TMPDIR/go-cache GOMODCACHE=$(go env GOPATH)/pkg/mod make build
```

Makefile に `GOCACHE` / `GOMODCACHE` のデフォルト値が設定されているため、
通常は `make build` だけで動作する。

### 依存ライブラリのバージョン

ローカルキャッシュに存在するバージョンを使う。
バージョン変更時は `go mod tidy` 前にキャッシュの存在を確認する。

| ライブラリ | バージョン |
|---|---|
| `gorilla/websocket` | v1.5.3 |
| `spf13/cobra` | v1.9.1 |
| `golang.org/x/term` | v0.34.0 |

---

## Slack API 実装上の注意

### Socket Mode

- イベント受信後は **3秒以内** に `envelope_id` で ACK を返す（処理前に送る）。
- `message` イベントのうち `subtype` が空のものだけを処理する（`message_deleted`, `message_changed` 等は除外）。
- `disconnect` イベント受信時は再接続する（`Run` メソッドがループで処理）。

### ファイルダウンロード

- `url_private_download` のダウンロードには `Authorization: Bearer <bot_token>` ヘッダーが必要。
- 保存ファイル名は `<fileID>_<name>` 形式で衝突を防ぐ。
- ダウンロード失敗は警告を stderr に出して処理を継続する（致命的エラーとしない）。

### よくある設定ミス（デバッグ時の確認ポイント）

1. Slack App の **Event Subscriptions** で `message.channels`（または `message.groups`）を購読していない
2. ボットがチャンネルに参加していない
3. Socket Mode が App 設定で有効になっていない

`--debug` フラグで受信 WebSocket イベントを stderr に出力できる。

---

## `cmd` パッケージ内の共通処理

複数のサブコマンドから使われるヘルパーは専用ファイルに分離する。

| ファイル | 内容 |
|---|---|
| `root.go` | グローバルフラグ、`persistentPreRunE`、`loadConfig` |
| `files.go` | `saveMessageFiles`、`downloadToFile`、`sanitizeFilename` |

新たに複数コマンドで共有するロジックが生まれた場合は同様に分離する。

---

## デバッグ出力のルール

- 通常出力（メッセージ本文等）→ `os.Stdout`
- 進捗・警告・デバッグ → `os.Stderr`
- `--debug` が有効な場合のみデバッグ行を出力する（`[debug]` プレフィックスを付ける）
