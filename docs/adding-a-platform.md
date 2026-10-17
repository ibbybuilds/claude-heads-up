# Adding a platform

The core never imports OS code. Everything OS-specific sits behind one interface in [`internal/platform/platform.go`](../internal/platform/platform.go):

```go
type Platform interface {
	Name() string
	HostWindow(pid int) (Window, error)
	IsForeground(w Window) (bool, error)
	Focus(w Window) error
	PauseMedia(ctx context.Context) ([]MediaSession, error)
	ResumeMedia(ctx context.Context, sessions []MediaSession) error
	PlaySound(ctx context.Context, path string) error
	Toast(ctx context.Context, n Notification) error
	ClearToast(ctx context.Context, id string) error
}
```

Windows and macOS are implemented. Linux is a stub package that embeds `platform.Unsupported`, so every method returns `ErrUnsupported` and the engine skips that step. You can replace the stub methods one at a time. Each one you implement is a feature that starts working.

The macOS backend in `internal/platform/darwin` is the best template for a new port. It shells out to OS tools through one injectable `runner` function, so its tests fake every command and run on any OS.

## Steps

1. Open `internal/platform/linux`.
2. Implement a method on `Backend`. The embedded `Unsupported` method stops being used as soon as yours exists.
3. Put OS-only code in files with a build tag (`//go:build darwin`) so the package still compiles on other systems in CI.
4. Update the support table in the README.
5. Add your checks to [manual-test.md](manual-test.md).

## Rules for each method

`HostWindow(pid)` finds the window the user sees Claude in. Walk up the parent process chain from `pid` and return the first process that owns a visible window. Claude Code runs hooks through a shell, so the window belongs to the terminal app, the editor, or the Claude desktop app several levels up.

`IsForeground(w)` compares by process, not window handle. Editors have many windows.

`PauseMedia` must pause only what is playing and return exactly that. `ResumeMedia` must touch only those sessions. Never toggle play/pause blindly, or you will start media the user paused on purpose.

`PlaySound("")` plays the system's default alert sound. It may block until the sound ends, because the process exits right after the alert and would cut it off.

`Toast` should replace an earlier toast with the same `Notification.ID`, and `ClearToast(id)` should remove it. Return `ErrUnsupported` from `ClearToast` if the OS can't do that.

Every method gets a context with a timeout. Respect it.

## Hints

macOS: `osascript` for focus and notifications. For media, MediaRemote is private but widely used. [nowplaying-cli](https://github.com/kirtan-shah/nowplaying-cli) shows how. Browsers can also be controlled through AppleScript, which is what SnapBack does.

Linux: MPRIS over D-Bus covers almost every player and browser. `playerctl --list-all` shows what is available, and `github.com/godbus/dbus` talks to it from Go. `notify-send` or the `org.freedesktop.Notifications` D-Bus interface for toasts. Focus depends on the window manager, so `ErrUnsupported` on Wayland is a fine first version.
