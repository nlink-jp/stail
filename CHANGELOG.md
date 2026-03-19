# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

## [0.1.0] - 2026-03-19

### Added

- **`stail tail`** — Show the last N messages from a channel (`-n`, default 10).
- **`stail tail -f`** — Follow mode: stream new messages in real time via Slack Socket Mode (requires `app_token`).
- **`stail export`** — Export full channel history as JSON, matching scat's export log schema (`export_timestamp`, `channel_name`, `messages`). Supports `--start` / `--end` (RFC3339) for time-range filtering and `--output` for file output.
- **`stail channel list`** — List all accessible channels with ID, name, privacy, and membership. Supports `--json`.
- **`stail config init`** — Create a default config file at `~/.config/stail/config.json` with `0600` permissions.
- **`stail profile add/use/list/set/remove`** — Full profile management. Tokens are entered via a secure prompt (no echo).
- **Server mode** — Set `STAIL_MODE=server` to read all configuration from environment variables (`STAIL_PROVIDER`, `STAIL_TOKEN`, `STAIL_APP_TOKEN`, `STAIL_CHANNEL`). Mirrors scat's server mode design.
- **Output formats** — `--format text` (human-readable) and `--format json` (JSONL for streaming, JSON array for export).
- **Attachment display** — Messages with attached files show filenames inline in text mode (e.g. `[添付: report.pdf]`).
- **`--save-dir`** — Download attached files to a local directory (`tail` and `export`). Directory is created automatically; files saved as `<fileID>_<filename>` to avoid collisions.
- **`--debug`** — Print received WebSocket envelopes and events to stderr for Socket Mode diagnostics.
- **Cross-platform binaries** — `make build-all` produces binaries for macOS (amd64/arm64), Linux (amd64/arm64), and Windows (amd64).
- **Unit tests** — Full test coverage for `config`, `slack`, and `format` packages using mock transports (no live Slack connection required).
