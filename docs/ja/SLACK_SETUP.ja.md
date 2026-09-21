# stail Slack セットアップガイド

`stail` を Slack で使用するには、Slack App を作成して 2 種類のトークンを取得する
必要があります。**Bot Token**（全コマンド共通）と **App-Level Token**（`tail -f`
によるリアルタイムストリーミング用）です。このガイドでは手順を順を追って説明します。

---

## ステップ 1: Slack App の作成

1. [Slack API サイト](https://api.slack.com/apps) にアクセスし、ログインします。

2. **「Create New App」** ボタンをクリックします。

3. 表示されるダイアログで **「From scratch」** を選択します。

4. アプリ名（例: `stail`）を入力し、インストールするワークスペースを選択して
   **「Create App」** をクリックします。

### Display Information（任意）

以下の説明文はあくまで参考です。ご自身の用途に合わせて自由に変更してください。

#### Short description

```
Read Slack messages from the command line. Stream channels like tail -f.
```

#### Long description

```
stail is a read-only command-line tool for Slack. It lets you stream channel
messages in real time (like tail -f) using Slack Socket Mode, or export full
channel history to structured JSON. Designed for developers, operators, and
anyone who wants to integrate Slack message data into scripts or pipelines.
```

---

## ステップ 2: Bot Token スコープの追加

1. 左サイドバーから **「OAuth & Permissions」** をクリックします。

2. **「Scopes」** セクションまでスクロールします。

3. **「Bot Token Scopes」** の **「Add an OAuth Scope」** をクリックし、
   以下のスコープを追加します。

   **必須スコープ:**

   | スコープ | 用途 |
   |---|---|
   | `channels:history` | パブリックチャンネルのメッセージ履歴を読み取る |
   | `channels:read` | チャンネルの一覧取得・名前解決 |
   | `users:read` | ユーザー ID から表示名を解決する |

   **オプションスコープ（プライベートチャンネルを使う場合）:**

   | スコープ | 用途 |
   |---|---|
   | `groups:history` | プライベートチャンネルのメッセージ履歴を読み取る |
   | `groups:read` | プライベートチャンネルの一覧取得 |

> **注意:** stail は読み取り専用ツールです。`chat:write` や `files:write` などの
> 書き込み権限スコープは**不要**です。

---

## ステップ 3: Socket Mode の有効化（`tail -f` を使う場合のみ）

Socket Mode は WebSocket 経由でメッセージをリアルタイム受信する `--follow` / `-f`
フラグに必要です。`stail tail`（`-f` なし）や `stail export` だけ使う場合は
このステップをスキップできます。

1. 左サイドバーから **「Socket Mode」** をクリックします。

2. **「Enable Socket Mode」** を **オン** にします。

---

## ステップ 4: App-Level Token の作成（`tail -f` を使う場合のみ）

1. 左サイドバーから **「Basic Information」** をクリックします。

2. **「App-Level Tokens」** セクションまでスクロールし、
   **「Generate Token and Scopes」** をクリックします。

3. トークン名を入力します（例: `stail-socket`）。

4. **「Add Scope」** をクリックし、**`connections:write`** スコープを追加します。

5. **「Generate」** をクリックし、生成されたトークンをコピーします。
   トークンは `xapp-` で始まります。

   このトークンは安全な場所に保管してください。ステップ 8 で使用します。

---

## ステップ 5: メッセージイベントの購読（`tail -f` を使う場合のみ）

1. 左サイドバーから **「Event Subscriptions」** をクリックします。

2. **「Enable Events」** を **オン** にします。

3. **「Subscribe to bot events」** の **「Add Bot User Event」** をクリックし、
   以下のイベントを追加します。

   | イベント | 用途 |
   |---|---|
   | `message.channels` | パブリックチャンネルのメッセージを受信 |
   | `message.groups` | プライベートチャンネルのメッセージを受信（任意） |

4. **「Save Changes」** をクリックします。

---

## ステップ 6: ワークスペースへのアプリのインストール

1. 左サイドバーから **「OAuth & Permissions」** をクリックします。

2. ページ上部の **「Install to Workspace」** をクリックします。

3. **「Allow」** をクリックしてアプリを承認します。

---

## ステップ 7: Bot Token のコピー

インストール完了後、**「OAuth & Permissions」** ページに
**「Bot User OAuth Token」** が表示されます。`xoxb-` で始まるトークンをコピーします。

---

## ステップ 8: stail の設定

### 設定ファイルの初期化

```bash
stail config init
```

`~/.config/stail/config.json` がパーミッション `0600`（所有者のみ読み書き可）で
作成されます。

### プロファイルの追加

```bash
stail profile add my-workspace --provider slack --channel "#general"
```

2 つのトークンの入力を求められます:

```
Bot Token (xoxb-...): [xoxb- トークンを貼り付け]
App Token (xapp-..., leave empty to skip): [xapp- トークンを貼り付け、不要なら Enter]
```

### アクティブプロファイルの設定

```bash
stail profile use my-workspace
```

これでセットアップは完了です。

---

## ステップ 9: チャンネルへのボットの招待

ボットが読み取るチャンネルのメンバーである必要があります（特にプライベートチャンネル）。

読み取りたい各 Slack チャンネルで以下のコマンドを実行してください:

```
/invite @<あなたのアプリ名>
```

---

## 動作確認

セットアップが正しく完了しているか確認します:

```bash
# アクセス可能なチャンネルの一覧を表示
stail channel list

# チャンネルの最新 5 件を表示
stail tail -c "#general" -n 5

# リアルタイムストリーミング（app_token が必要）
stail tail -c "#general" -f
```

---

## トークンのまとめ

| トークン | 取得場所 | 用途 |
|---|---|---|
| `xoxb-...`（Bot Token） | OAuth & Permissions → Bot User OAuth Token | 全コマンド |
| `xapp-...`（App-Level Token） | Basic Information → App-Level Tokens | `tail -f` のみ |
