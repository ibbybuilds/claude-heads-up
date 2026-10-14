# Changelog

## v0.1.2

- Alert even when the Claude window is focused. `skipWhenFocused` is now off by default, because a focused window does not mean you are looking at it on a multi-monitor setup or with a phone in hand

## v0.1.1

- Install over HTTPS, so users without a GitHub SSH key can install
- Log one line per alert and resume, so the log shows that hooks run

## v0.1.0

- Windows backend: pause and resume media through the system media API, play the notification sound, focus the Claude window, show a toast
- Built-in attention chime as the default sound
- Sticky toasts that stay until dismissed
- Phone push through ntfy on every OS, `urgent` priority by default
- Alerts on Stop, permission prompts and AskUserQuestion
- Resume only for a reply in the session that paused the media
- `heads-up doctor`, `test` and `config` commands
- macOS and Linux stubs
