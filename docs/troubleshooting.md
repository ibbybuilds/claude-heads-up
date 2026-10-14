# Troubleshooting

Start with these two commands. The binary is in the plugin's `dist` folder, or run it through the launcher at `bin/heads-up`.

```bash
heads-up doctor   # what it detected, where the config and log are
heads-up test     # fire an alert now, ignoring triggers and focus
```

Logs go to `%LOCALAPPDATA%\claude-heads-up\heads-up.log` on Windows. Set `HEADS_UP_DEBUG=1` before starting Claude Code to log every step.

## Nothing happens at all

Check that Claude Code loaded the hooks. Run `/hooks` in Claude Code and look for `heads-up` entries. If they're missing, reinstall the plugin and restart Claude Code.

On Windows, check that Git for Windows is installed. The hooks run through Git Bash.

If you turned on `skipWhenFocused`, nothing happens while the Claude window is in front. That is on purpose.

## Video doesn't pause

The media has to show up in the Windows media flyout (press a volume key and look for it). If it isn't there, Windows can't control it.

Muted video never shows up there. Instagram starts reels muted, so a reel only becomes pausable after you turn the sound on.

Chromium browsers (Chrome, Brave, Edge) report one active media session per browser to Windows. If an old paused YouTube tab holds it, a reel playing in another tab can be invisible. Closing the old tab fixes it.

## Video doesn't resume

Only a reply in the same Claude session resumes the media. If you replied in a different session, the video stays paused on purpose.

Declining a permission request fires no hook, so the media stays paused until your next message.

## Window doesn't come to the front

Windows blocks background programs from stealing focus in some situations. The log will say `windows refused to focus`. The toast and sound still work. Click the toast to switch.

## The toast says "Windows PowerShell"

Windows only shows toasts from registered apps. claude-heads-up borrows PowerShell's registration so it doesn't need an installer.
