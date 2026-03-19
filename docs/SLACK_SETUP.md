# stail Slack Setup Guide

To use `stail` with Slack, you need to create a Slack App and obtain two tokens:
a **Bot Token** (for all commands) and an **App-Level Token** (for `tail -f` real-time
streaming). This guide walks you through the complete process.

---

## Step 1: Create a Slack App

1. Go to the [Slack API site](https://api.slack.com/apps) and log in.

2. Click the **"Create New App"** button.

3. In the dialog, select **"From scratch"**.

4. Enter an app name (e.g., `stail`), select your workspace, and click **"Create App"**.

### Suggested Display Information

The following descriptions are optional — feel free to use your own.

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

## Step 2: Add Bot Token Scopes

1. From the left sidebar, click **"OAuth & Permissions"**.

2. Scroll down to the **"Scopes"** section.

3. Under **"Bot Token Scopes"**, click **"Add an OAuth Scope"** and add the
   following scopes:

   **Required scopes:**

   | Scope | Purpose |
   |---|---|
   | `channels:history` | Read message history from public channels |
   | `channels:read` | List public channels and resolve channel names |
   | `users:read` | Resolve user IDs to display names |

   **Optional scopes (for private channels):**

   | Scope | Purpose |
   |---|---|
   | `groups:history` | Read message history from private channels |
   | `groups:read` | List private channels |

> **Note:** stail is a read-only tool. It does **not** require `chat:write`,
> `files:write`, or any other write-permission scopes.

---

## Step 3: Enable Socket Mode (required for `tail -f`)

Socket Mode is needed for the `--follow` / `-f` flag, which streams new messages
in real time via WebSocket. If you only need `stail tail` (without `-f`) and
`stail export`, you can skip this step.

1. From the left sidebar, click **"Socket Mode"**.

2. Toggle **"Enable Socket Mode"** to **on**.

---

## Step 4: Create an App-Level Token (required for `tail -f`)

1. From the left sidebar, click **"Basic Information"**.

2. Scroll down to **"App-Level Tokens"** and click **"Generate Token and Scopes"**.

3. Give the token a name (e.g., `stail-socket`).

4. Click **"Add Scope"** and add the **`connections:write`** scope.

5. Click **"Generate"** and copy the token — it starts with `xapp-`.

   Keep this token secure; you will need it in Step 7.

---

## Step 5: Subscribe to Message Events (required for `tail -f`)

1. From the left sidebar, click **"Event Subscriptions"**.

2. Toggle **"Enable Events"** to **on**.

3. Under **"Subscribe to bot events"**, click **"Add Bot User Event"** and add:

   | Event | Purpose |
   |---|---|
   | `message.channels` | Receive messages from public channels |
   | `message.groups` | Receive messages from private channels (optional) |

4. Click **"Save Changes"**.

---

## Step 6: Install the App to Your Workspace

1. From the left sidebar, click **"OAuth & Permissions"**.

2. Scroll to the top and click **"Install to Workspace"**.

3. Click **"Allow"** to authorize the app.

---

## Step 7: Copy Your Bot Token

After installation, the **"OAuth & Permissions"** page shows a
**"Bot User OAuth Token"** starting with `xoxb-`. Copy it.

---

## Step 8: Configure stail

### Initialize the config file

```bash
stail config init
```

This creates `~/.config/stail/config.json` with `0600` permissions.

### Add a profile

```bash
stail profile add my-workspace --provider slack --channel "#general"
```

You will be prompted for two tokens:

```
Bot Token (xoxb-...): [paste your xoxb- token]
App Token (xapp-..., leave empty to skip): [paste your xapp- token, or press Enter to skip]
```

### Set the active profile

```bash
stail profile use my-workspace
```

Your setup is now complete.

---

## Step 9: Invite the Bot to Channels

The bot must be a member of any channel it needs to read from (this applies
especially to private channels).

In each Slack channel you want to monitor, type:

```
/invite @<your-app-name>
```

---

## Verification

Test that everything is working:

```bash
# List accessible channels
stail channel list

# Show the last 5 messages from a channel
stail tail -c "#general" -n 5

# Stream messages in real time (requires app_token)
stail tail -c "#general" -f
```

---

## Token Summary

| Token | Where to find it | Used for |
|---|---|---|
| `xoxb-...` (Bot Token) | OAuth & Permissions → Bot User OAuth Token | All commands |
| `xapp-...` (App-Level Token) | Basic Information → App-Level Tokens | `tail -f` only |
