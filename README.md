# claude-heads-up

You ask Claude Code to do something, switch to YouTube or Instagram while it works, and miss the moment it finishes. Claude sits there waiting for a reply you didn't know it needed.

claude-heads-up is a Claude Code plugin that fixes this. When Claude finishes or needs permission, it:

1. pauses whatever video or music is playing
2. plays a loud two-burst chime, built to be hard to tune out
3. brings the terminal or Claude window to the front
4. shows a toast that stays on screen until you close it
5. optionally sends an urgent push to your phone through [ntfy](https://ntfy.sh)

When you reply, it resumes the media it paused. Only that media. If you paused a song yourself, it stays paused.

Inspired by [SnapBack](https://github.com/teoobarca/snapback), which does this on macOS.

## Platform support

| | Windows 10/11 | macOS | Linux |
|---|---|---|---|
| Pause and resume media | yes | planned | planned |
| Sound | yes | planned | planned |
| Focus Claude window | yes | planned | planned |
| Toast | yes | planned | planned |
| Phone push (ntfy) | yes | yes | yes |

The core is OS-independent. Each OS is one folder under `internal/platform/`. See [adding a platform](docs/adding-a-platform.md) if you want to help with macOS or Linux.

## Install

In Claude Code:

```
/plugin marketplace add ibbybuilds/claude-heads-up
/plugin install claude-heads-up
```

Restart Claude Code. That's it. No config needed.

Windows needs [Git for Windows](https://git-scm.com/download/win), because Claude Code runs plugin hooks through Git Bash. Claude Code itself recommends it, so you probably have it already.

## What triggers an alert

| Claude Code event | Alert |
|---|---|
| Claude finishes its turn (`Stop`) | yes |
| Claude asks for permission (`Notification`) | yes |
| Claude asks you a question (`AskUserQuestion`) | yes |
| You send a message | resumes media |
| You approve a permission and the tool runs | resumes media |

If you decline a permission, Claude Code fires no hook, so your media stays paused until you send your next message.

## Phone push for reels on your phone

Desktop alerts can't reach you if you're scrolling on your phone. ntfy can:

1. Install the ntfy app ([Android](https://play.google.com/store/apps/details?id=io.heckel.ntfy), [iOS](https://apps.apple.com/app/ntfy/id1625396347)).
2. Subscribe to a topic with a long random name, like `heads-up-k3v9q2xw7m`. Anyone who knows a public topic name can read it, so don't use `claude`.
3. Turn it on in the config:

```json
{
  "ntfy": { "enabled": true, "topic": "heads-up-k3v9q2xw7m" }
}
```

Pushes go out at ntfy's `urgent` priority. On Android that means a banner over whatever app you're in and a long vibration. To make it louder on Android, open ntfy, go to Settings, then Channel settings, and open the max priority channel. Give it a long ringtone as its sound, turn on pop on screen, and allow it to override Do Not Disturb if you want. Also turn off battery optimization for ntfy so pushes arrive instantly.

Self-hosted ntfy works too. Set `server`, and put an access token in `token` or the `HEADS_UP_NTFY_TOKEN` environment variable.

## Configuration

Optional. Create the file with every default filled in:

```bash
heads-up config init
```

It lives at `%APPDATA%\claude-heads-up\config.json` on Windows. See [docs/config.md](docs/config.md) for every option.

The `heads-up` binary is inside the plugin folder. `heads-up config path` prints the config location, and `heads-up doctor` checks your setup.

## Troubleshooting

Run `heads-up test` to fire an alert right now, and `heads-up doctor` to see what it detected. Logs go to `%LOCALAPPDATA%\claude-heads-up\heads-up.log`. Set `HEADS_UP_DEBUG=1` for verbose logs.

More in [docs/troubleshooting.md](docs/troubleshooting.md).

## How it works

Claude Code runs the plugin's hooks in the background, so they never slow Claude down. Each hook calls a small Go binary. The binary reads the event, decides whether to alert, and calls the OS backend.

On Windows, media control goes through the same system media API that powers the volume flyout. Every browser tab and app that shows up there can be paused, with no per-browser code. The backend talks to that API through Windows PowerShell 5.1, which ships with Windows.

The [design doc](docs/specs/2026-10-09-design.md) has the full picture.

## License

MIT
