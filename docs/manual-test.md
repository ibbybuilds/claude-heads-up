# Manual test checklist

Unit tests cover the rules. These checks cover the parts that need a real desktop. Run them before a release that touches a platform backend.

Build and install locally first:

```bash
sh scripts/build-dist.sh --local
```

Then point a Claude Code session at the repo as a plugin (see CONTRIBUTING.md).

## Windows

- [ ] YouTube in Chrome: alert pauses it, reply resumes it
- [ ] YouTube in Brave or Edge: same
- [ ] Instagram reels in the browser: same
- [ ] Spotify desktop: same
- [ ] Pause a song yourself, then trigger an alert and reply: the song stays paused
- [ ] Claude window in front when the alert fires: nothing happens
- [ ] Focus works from Windows Terminal, VS Code terminal and the Claude desktop app
- [ ] Minimized window gets restored
- [ ] Two sessions: alert in A, reply in B, media stays paused. Reply in A resumes it
- [ ] Approve a permission prompt: media resumes
- [ ] Answer an AskUserQuestion prompt: media resumes
- [ ] Toast shows project name and reason
- [ ] ntfy push arrives on the phone
- [ ] `heads-up doctor` shows the right host process
- [ ] Delete the config, put invalid JSON in it: Claude Code still works, doctor reports it
