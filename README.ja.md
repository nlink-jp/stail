# stail — Slack Tail

`stail` は Slack の読み取り専用コマンドラインツールです。チャンネルのメッセージを
`tail -f` の感覚でリアルタイムにストリームしたり、チャンネルの全履歴を JSON に
エクスポートしたりすることができます。

[scat](https://github.com/magifd2/scat) の姉妹ツールとして設計されています。
scat が Slack に**投稿する**ツールであるのに対し、stail は Slack を**読む**ツールです。

---

## 機能

- **リアルタイムストリーミング** — `stail tail -f` で Slack Socket Mode (WebSocket)
  に接続し、新着メッセージを随時表示します。
- **過去ログの表示** — `stail tail -n 50` で最新 N 件を表示して終了します。
- **チャンネルエクスポート** — `stail export` でチャンネルの全履歴を scat の
  export log 形式と互換性のある JSON ファイルとして保存します。
- **チャンネル一覧** — `stail channel list` でアクセス可能な全チャンネルと
  その ID を一覧表示します。
- **プロファイル管理** — トークンやデフォルトチャンネルごとに複数の名前付き
  プロファイルを管理できます。
- **サーバモード** — コンテナや CI 環境向けに、環境変数だけで設定を完結させる
  サーバモードをサポートします。
- **クロスプラットフォーム** — macOS・Linux・Windows 向けのシングルバイナリを
  提供します。

---

## インストール

[Releases](https://github.com/magifd2/stail/releases) ページから各プラットフォーム
向けのバイナリをダウンロードしてください。

ソースからビルドする場合:

```bash
make build
# バイナリ: ./bin/stail
```

---

## 初期設定

詳細な手順については **[Slack セットアップガイド](./docs/SLACK_SETUP.ja.md)** を参照してください。

### 1. Slack App の作成

stail には Slack App の 2 種類のトークンが必要です。

| トークン | 必要なコマンド | 形式 |
|---|---|---|
| **Bot Token** | 全コマンド | `xoxb-...` |
| **App-Level Token** | `tail -f`（Socket Mode）| `xapp-...` |

**Bot Token のスコープ**（OAuth & Permissions）:
- `channels:history` — パブリックチャンネルのメッセージ読み取り
- `channels:read` — パブリックチャンネルの一覧取得
- `groups:history` — プライベートチャンネルのメッセージ読み取り（必要な場合）
- `groups:read` — プライベートチャンネルの一覧取得（必要な場合）
- `users:read` — ユーザ表示名の解決

**App-Level Token**（`tail -f` 使用時のみ必要）:
1. アプリの **Basic Information** ページを開きます。
2. **App-Level Tokens** セクションで **Generate Token and Scopes** をクリックします。
3. `connections:write` スコープを追加します。
4. 生成された `xapp-...` トークンをコピーします。

アプリ設定で **Socket Mode** を有効にしてください。

### 2. 設定ファイルの初期化

```bash
stail config init
```

`~/.config/stail/config.json` をパーミッション `0600`（所有者のみ読み書き可）で
作成します。

### 3. プロファイルの追加

```bash
stail profile add my-workspace --provider slack --channel "#general"
# Bot Token と App Token の入力を求められます（入力は画面に表示されません）
```

### 4. アクティブプロファイルの設定

```bash
stail profile use my-workspace
```

---

## 使い方

### メッセージの表示 (`tail`)

```bash
# デフォルトチャンネルの最新 10 件を表示
stail tail

# 指定チャンネルの最新 50 件を表示
stail tail -c "#general" -n 50

# フォローモード: リアルタイムで新着メッセージを表示（app_token が必要）
stail tail -c "#general" -f

# フォローモード + JSON 出力（JSONL: 1行1メッセージ）
stail tail -c "#general" -f --format json

# 添付ファイルをディレクトリに保存しながら表示
stail tail -c "#general" -f --save-dir ./downloads
```

テキストモードでは添付ファイルがあるメッセージにファイル名が表示されます:

```
2026-03-19 10:44:11  #general  @alice  確認してください [添付: report.pdf]
2026-03-19 10:44:12  #general  @bob    [添付: photo.png]
```

### チャンネル履歴のエクスポート (`export`)

チャンネルの全履歴を scat の export log 形式と互換性のある JSON として出力します。

```bash
# 標準出力へ出力
stail export -c "#general"

# ファイルへ出力
stail export -c "#general" --output archive.json

# 期間を指定してエクスポート（RFC3339 形式）
stail export -c "#general" \
  --start 2025-01-01T00:00:00Z \
  --end   2025-02-01T00:00:00Z

# 全添付ファイルも保存
stail export -c "#general" --output archive.json --save-dir ./attachments
```

**エクスポート JSON スキーマ**（scat と互換）:

```json
{
  "export_timestamp": "2026-03-19T10:00:00Z",
  "channel_name": "#general",
  "messages": [
    {
      "user_id": "U12345ABC",
      "user_name": "alice",
      "post_type": "user",
      "timestamp": "2026-03-19T09:55:00Z",
      "timestamp_unix": "1742378100.000000",
      "text": "こんにちは！",
      "files": [],
      "thread_timestamp_unix": "",
      "is_reply": false
    }
  ]
}
```

`post_type` は `"user"`（ユーザ投稿）または `"bot"`（Bot 投稿）のいずれかです。

### チャンネル一覧 (`channel list`)

```bash
stail channel list
stail channel list --json
```

### プロファイル管理 (`profile`)

```bash
stail profile list                   # プロファイル一覧
stail profile use my-workspace       # アクティブプロファイルを変更
stail profile add staging            # 新規追加（トークンをプロンプトで入力）
stail profile set channel "#ops"     # デフォルトチャンネルを変更
stail profile set token              # Bot Token を更新（セキュアなプロンプト）
stail profile set app_token          # App Token を更新（セキュアなプロンプト）
stail profile remove staging         # プロファイルを削除
```

### 設定管理 (`config`)

```bash
stail config init   # デフォルト設定ファイルを作成
```

---

## グローバルフラグ

| フラグ | 説明 |
|---|---|
| `--config <path>` | 設定ファイルのパスを指定 |
| `--profile <name>` / `-p` | このコマンド実行時のみ使用するプロファイルを指定 |
| `--debug` | デバッグログを有効化 |

---

## サーバモード

コンテナや CI/CD 環境では `STAIL_MODE=server` を設定することで、設定ファイルを
使わずに環境変数のみで動作させることができます。

| 変数 | 必須 | 説明 |
|---|---|---|
| `STAIL_MODE` | yes | `server` に設定 |
| `STAIL_PROVIDER` | yes | プロバイダ: `slack` |
| `STAIL_TOKEN` | yes | Bot Token (`xoxb-...`) |
| `STAIL_APP_TOKEN` | no | App-Level Token（`tail -f` 使用時に必要） |
| `STAIL_CHANNEL` | no | デフォルトチャンネル |

```bash
export STAIL_MODE=server
export STAIL_PROVIDER=slack
export STAIL_TOKEN=xoxb-xxxxxxxxxxxx
export STAIL_CHANNEL="#alerts"

stail tail -n 20
```

### Kubernetes の例

```yaml
env:
  - name: STAIL_MODE
    value: "server"
  - name: STAIL_PROVIDER
    value: "slack"
  - name: STAIL_CHANNEL
    value: "#alerts"
  - name: STAIL_TOKEN
    valueFrom:
      secretKeyRef:
        name: slack-credentials
        key: bot-token
```

### サーバモードの制限

サーバモードでは以下のコマンドは使用できません:

- `--config` フラグ
- `--profile` フラグ
- `profile` サブコマンド群
- `config init`

---

## ビルド

```bash
# 現在の OS/Arch 向けにビルド
make build

# 全プラットフォーム向けにクロスコンパイル
# (macOS amd64/arm64, Linux amd64/arm64, Windows amd64)
make build-all

# テスト実行
make test

# 依存関係の整理
make tidy
```

> **注意（サンドボックスや制限環境）:** デフォルトの Go キャッシュパスに書き込み
> 権限がない場合は、`GOCACHE` と `GOMODCACHE` を明示的に指定してください:
>
> ```bash
> GOCACHE=/tmp/go-cache GOMODCACHE=/path/to/gopath/pkg/mod make build
> ```

---

## ライセンス

MIT License — [LICENSE](LICENSE) を参照してください。
