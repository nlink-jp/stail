# 設定ファイルのフォーマット

## ファイルの場所

| プラットフォーム | デフォルトパス |
|---|---|
| macOS / Linux | `~/.config/stail/config.json` |
| Windows | `%USERPROFILE%\.config\stail\config.json` |

`--config` フラグで別のパスを指定できます:

```bash
stail --config /path/to/my-config.json tail -c "#general"
```

このファイルはトークンの秘匿情報を保護するため、**パーミッション `0600`**
（所有者のみ読み書き可）で作成されます。

---

## トップレベル構造

```json
{
  "current_profile": "<profile-name>",
  "profiles": {
    "<profile-name>": { ... },
    "<profile-name>": { ... }
  }
}
```

| フィールド | 型 | 説明 |
|---|---|---|
| `current_profile` | string | デフォルトで使用されるプロファイルの名前 |
| `profiles` | object | プロファイル名 → プロファイルオブジェクトのマップ |

---

## プロファイルのフィールド

```json
{
  "provider":  "slack",
  "token":     "xoxb-...",
  "app_token": "xapp-...",
  "channel":   "#general",
  "username":  "mybot"
}
```

| フィールド | 型 | 必須 | 説明 |
|---|---|---|---|
| `provider` | string | yes | プロバイダの種類。現在は `slack` のみサポートされています。 |
| `token` | string | yes | Slack Bot Token (`xoxb-...`)。全コマンドで必要です。 |
| `app_token` | string | no | Slack App-Level Token (`xapp-...`)。`tail -f`（Socket Mode）で必要です。 |
| `channel` | string | no | `-c` が指定されなかったときに使用されるデフォルトチャンネル。`#channel-name` またはチャンネル ID（`C...`）を指定できます。 |
| `username` | string | no | 将来の使用のために予約されています。 |

---

## 完全な例

本番用とステージング用の 2 つのプロファイルを持つ設定ファイル:

```json
{
  "current_profile": "production",
  "profiles": {
    "production": {
      "provider":  "slack",
      "token":     "xoxb-YOUR-BOT-TOKEN",
      "app_token": "xapp-YOUR-APP-TOKEN",
      "channel":   "#alerts"
    },
    "staging": {
      "provider": "slack",
      "token":    "xoxb-YOUR-STAGING-BOT-TOKEN",
      "channel":  "#staging-logs"
    }
  }
}
```

---

## 設定の管理

### ファイルの作成

```bash
stail config init
```

空の `default` プロファイルを持つファイルをデフォルトパスに作成します。

### プロファイルの追加（対話式 — トークンは画面に表示されません）

```bash
stail profile add production --provider slack --channel "#alerts"
# Prompts:
#   Bot Token (xoxb-...):
#   App Token (xapp-..., leave empty to skip):
```

### アクティブプロファイルの切り替え

```bash
stail profile use staging
```

### 個別のフィールドの更新

```bash
stail profile set channel "#ops"
stail profile set token        # secure prompt
stail profile set app_token    # secure prompt
```

### 全プロファイルの一覧表示

```bash
stail profile list
# * production (provider: slack)    ← active
#   staging    (provider: slack)
```

### プロファイルの削除

```bash
stail profile remove staging
```

---

## セキュリティに関する注意

- 設定ファイルはパーミッション `0600` で保存されます。これを変更しないでください。
- トークンがログやデバッグ出力に書き出されることはありません。
- `config.json` をバージョン管理にコミットしないでください。
- CI/CD 環境では、設定ファイルの代わりに環境変数を使う
  [サーバモード](../../README.ja.md#サーバモード) を利用してください。

---

## サーバモード（設定ファイルなし）

`STAIL_MODE=server` が設定されている場合、設定ファイルは完全に無視され、すべての
設定が環境変数から読み込まれます。詳細は
[README](../../README.ja.md#サーバモード) を参照してください。
