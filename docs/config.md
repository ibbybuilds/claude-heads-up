# Configuration

The config file is optional. Without it, everything except phone push is on.

| OS | Path |
|---|---|
| Windows | `%APPDATA%\claude-heads-up\config.json` |
| macOS | `~/Library/Application Support/claude-heads-up/config.json` |
| Linux | `~/.config/claude-heads-up/config.json` |

`heads-up config init` writes the defaults. `heads-up config path` prints the path.

You only need the keys you want to change. Anything you leave out keeps its default.

## Full example with defaults

```json
{
  "triggers": {
    "stop": true,
    "permission": true,
    "question": true
  },
  "skipWhenFocused": true,
  "throttleSeconds": 2,
  "media": { "enabled": true },
  "sound": { "enabled": true, "file": "" },
  "focus": { "enabled": true },
  "toast": { "enabled": true, "sticky": true },
  "ntfy": {
    "enabled": false,
    "server": "https://ntfy.sh",
    "topic": "",
    "token": "",
    "priority": "urgent"
  }
}
```

## Options

`triggers.stop` alerts when Claude finishes its turn.

`triggers.permission` alerts when Claude asks for permission to use a tool, or an MCP server asks for input.

`triggers.question` alerts when Claude asks you a question with the AskUserQuestion tool.

`skipWhenFocused` does nothing when the window running this Claude session is already in front. Turn it off if you want the sound every time.

`throttleSeconds` ignores a second alert from the same session within this many seconds. Claude often fires two events back to back.

`media.enabled` pauses playing media on alert and resumes it when you reply.

`sound.file` is a path to a `.wav` file. Empty means the built-in chime: two fast rising arpeggios at full volume. Point it at `C:\Windows\Media\Windows Notify System Generic.wav` if you want the quiet Windows sound instead.

`focus.enabled` brings the Claude window to the front.

`toast.enabled` shows a desktop notification. The Claude desktop app has its own notifications, so you may want this off there.

`toast.sticky` keeps the toast on screen until you close it. Set it to `false` for a normal toast that slides away.

`ntfy.enabled` sends a push to your phone. Needs `topic`.

`ntfy.server` is the ntfy server. Change it if you self-host.

`ntfy.topic` is the topic your phone subscribes to. Pick a long random name.

`ntfy.priority` is one of `min`, `low`, `default`, `high`, `urgent`. `urgent` shows a banner over the current app on Android and is time-sensitive on iOS.

`ntfy.token` is an access token for servers with access control. The `HEADS_UP_NTFY_TOKEN` environment variable overrides it, so you can keep the token out of the file.

## Bad values

A broken config never breaks Claude Code. If the file isn't valid JSON, every option uses its default. Unknown keys are ignored. `heads-up doctor` lists every problem it found.
