# stail — Slack Tail

`stail` is a read-only command-line tool for Slack. It lets you stream channel
messages in real time — like `tail -f` — or export full channel history to JSON.

It is designed as a companion to [scat](https://github.com/magifd2/scat): scat
**posts** to Slack, stail **reads** from it.

---

## Features

- **Real-time streaming** — `stail tail -f` connects via Slack Socket Mode
  (WebSocket) and prints new messages as they arrive.
- **Historical tail** — `stail tail -n 50` shows the last N messages and exits.
- **Timestamp-based tail** — `stail tail --since 2024-01-15T10:00:00Z` streams all messages from an absolute point in time (Slack ts or RFC3339).
- **Channel export** — `stail export` downloads full channel history as a
  structured JSON file compatible with scat's export log format.
- **Channel listing** — `stail channel list` shows all accessible channels with
  their IDs.
- **Profile management** — Multiple named profiles, each with its own token and
  default channel.
- **Server mode** — Configure entirely via environment variables for container /
  CI deployments.
- **Cross-platform** — Single static binary for macOS, Linux, and Windows.

---

## Installation

Download the latest binary for your platform from the
[Releases](https://github.com/magifd2/stail/releases) page.

Or build from source:

```bash
make build
# Binary: ./bin/stail
```

---

## Initial Setup

For a detailed step-by-step walkthrough, see the **[Slack Setup Guide](./docs/SLACK_SETUP.md)**.

### 1. Create a Slack App

stail requires two tokens from your Slack App:

| Token | Required for | Format |
|---|---|---|
| **Bot Token** | All commands | `xoxb-...` |
| **App-Level Token** | `tail -f` (Socket Mode) | `xapp-...` |

**Bot Token scopes** (OAuth & Permissions):
- `channels:history` — read public channel messages
- `channels:read` — list public channels
- `groups:history` — read private channel messages (if needed)
- `groups:read` — list private channels (if needed)
- `users:read` — resolve user display names

**App-Level Token** (required for `tail -f` only):
1. Go to your app's **Basic Information** page.
2. Under **App-Level Tokens**, click **Generate Token and Scopes**.
3. Add the `connections:write` scope.
4. Copy the `xapp-...` token.

Enable **Socket Mode** in your app settings.

### 2. Initialize the config

```bash
stail config init
```

Creates `~/.config/stail/config.json` with `0600` permissions.

### 3. Add a profile

```bash
stail profile add my-workspace --provider slack --channel "#general"
# You will be prompted to enter the Bot Token and App Token securely.
```

### 4. Set the active profile

```bash
stail profile use my-workspace
```

---

## Usage

### Tail messages (`tail`)

```bash
# Show last 10 messages from the default channel
stail tail

# Show last 50 messages from a specific channel
stail tail -c "#general" -n 50

# Show all messages since an absolute timestamp (RFC3339 or Slack ts)
stail tail -c "#general" --since 2024-01-15T10:00:00Z
stail tail -c "#general" --since 1742378100.123456

# Show the newest 5 messages since a given timestamp
stail tail -c "#general" --since 2024-01-15T10:00:00Z -n 5

# Follow mode: stream new messages in real time (requires app_token)
stail tail -c "#general" -f

# Follow mode with JSON output (JSONL, one object per line)
stail tail -c "#general" -f --format json

# Save attached files to a directory while tailing
stail tail -c "#general" -f --save-dir ./downloads
```

Attached files are shown inline in text mode:

```
2026-03-19 10:44:11  #general  @alice  See attached [添付: report.pdf]
2026-03-19 10:44:12  #general  @bob    [添付: photo.png]
```

### Export channel history (`export`)

Exports the full history as a JSON document matching scat's export log schema:

```bash
# Export to stdout
stail export -c "#general"

# Export to a file
stail export -c "#general" --output archive.json

# Export a specific time range (RFC3339 or Slack ts)
stail export -c "#general" \
  --start 2025-01-01T00:00:00Z \
  --end   2025-02-01T00:00:00Z

# Export and save all attached files
stail export -c "#general" --output archive.json --save-dir ./attachments
```

> **Note:** `export` fetches the full channel history into memory before writing.
> For very large channels, use `--start` / `--end` to export in smaller time ranges.
> Both flags accept RFC3339 (e.g. `2025-01-01T00:00:00Z`) or Slack ts format (e.g. `1742378100.000000`).

**Export JSON schema** (compatible with scat):

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
      "text": "Hello!",
      "files": [],
      "thread_timestamp_unix": "",
      "is_reply": false
    }
  ]
}
```

`post_type` is either `"user"` or `"bot"`.

### List channels (`channel list`)

```bash
stail channel list
stail channel list --json
```

### Profile management (`profile`)

```bash
stail profile list              # list all profiles
stail profile use my-workspace  # switch active profile
stail profile add staging       # add a new profile (prompts for tokens)
stail profile set channel "#ops"
stail profile set token         # update token (secure prompt)
stail profile set app_token     # update app token (secure prompt)
stail profile remove staging
```

### Config (`config`)

```bash
stail config init   # create default config file
```

---

## Global Flags

| Flag | Description |
|---|---|
| `--config <path>` | Alternative config file path |
| `--profile <name>` / `-p` | Override the active profile for this invocation |
| `--debug` | Enable verbose debug logging |
| `--quiet` / `-q` | Suppress informational stderr messages (warnings and errors are still shown) |

---

## Server Mode

For container and CI deployments, set `STAIL_MODE=server` to read all
configuration from environment variables — no config file needed.

| Variable | Required | Description |
|---|---|---|
| `STAIL_MODE` | yes | Set to `server` |
| `STAIL_PROVIDER` | yes | Provider: `slack` |
| `STAIL_TOKEN` | yes | Bot Token (`xoxb-...`) |
| `STAIL_APP_TOKEN` | no | App-Level Token (required for `tail -f`) |
| `STAIL_CHANNEL` | no | Default channel |

```bash
export STAIL_MODE=server
export STAIL_PROVIDER=slack
export STAIL_TOKEN=xoxb-xxxxxxxxxxxx
export STAIL_CHANNEL="#alerts"

stail tail -n 20
```

### Kubernetes example

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

### Server mode restrictions

The following commands are not available in server mode:

- `--config` flag
- `--profile` flag
- All `profile` subcommands
- `config init`

---

## Build

```bash
# Current OS/Arch
make build

# All platforms (macOS amd64/arm64, Linux amd64/arm64, Windows amd64)
make build-all

# Run tests
make test

# Tidy dependencies
make tidy
```

> **Note (sandbox / restricted environments):** If the default Go cache paths
> are not writable, set `GOCACHE` and `GOMODCACHE` explicitly:
>
> ```bash
> GOCACHE=/tmp/go-cache GOMODCACHE=/path/to/gopath/pkg/mod make build
> ```

---

## License

MIT License — see [LICENSE](LICENSE).
