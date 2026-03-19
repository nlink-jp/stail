# Configuration File Format

## File Location

| Platform | Default Path |
|---|---|
| macOS / Linux | `~/.config/stail/config.json` |
| Windows | `%USERPROFILE%\.config\stail\config.json` |

An alternative path can be specified with the `--config` flag:

```bash
stail --config /path/to/my-config.json tail -c "#general"
```

The file is created with **`0600` permissions** (owner read/write only) to protect
token secrets.

---

## Top-Level Structure

```json
{
  "current_profile": "<profile-name>",
  "profiles": {
    "<profile-name>": { ... },
    "<profile-name>": { ... }
  }
}
```

| Field | Type | Description |
|---|---|---|
| `current_profile` | string | Name of the profile used by default |
| `profiles` | object | Map of profile name → profile object |

---

## Profile Fields

```json
{
  "provider":  "slack",
  "token":     "xoxb-...",
  "app_token": "xapp-...",
  "channel":   "#general",
  "username":  "mybot"
}
```

| Field | Type | Required | Description |
|---|---|---|---|
| `provider` | string | yes | Provider type. Currently only `slack` is supported. |
| `token` | string | yes | Slack Bot Token (`xoxb-...`). Required for all commands. |
| `app_token` | string | no | Slack App-Level Token (`xapp-...`). Required for `tail -f` (Socket Mode). |
| `channel` | string | no | Default channel used when `-c` is not specified. Accepts `#channel-name` or a channel ID (`C...`). |
| `username` | string | no | Reserved for future use. |

---

## Complete Example

A config file with two profiles — one for production, one for staging:

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

## Managing the Config

### Create the file

```bash
stail config init
```

Creates the file at the default path with an empty `default` profile.

### Add a profile (interactive — tokens are not echoed)

```bash
stail profile add production --provider slack --channel "#alerts"
# Prompts:
#   Bot Token (xoxb-...):
#   App Token (xapp-..., leave empty to skip):
```

### Switch the active profile

```bash
stail profile use staging
```

### Update individual fields

```bash
stail profile set channel "#ops"
stail profile set token        # secure prompt
stail profile set app_token    # secure prompt
```

### List all profiles

```bash
stail profile list
# * production (provider: slack)    ← active
#   staging    (provider: slack)
```

### Remove a profile

```bash
stail profile remove staging
```

---

## Security Notes

- The config file is stored with `0600` permissions. Do not change this.
- Tokens are never written to logs or debug output.
- Do not commit `config.json` to version control.
- For CI/CD environments, use [Server Mode](../README.md#server-mode) with
  environment variables instead of a config file.

---

## Server Mode (no config file)

When `STAIL_MODE=server` is set, the config file is ignored entirely and all
settings are read from environment variables. See the
[README](../README.md#server-mode) for details.
