// Package platform defines what an OS backend must provide. The engine only
// talks to this interface, so adding an OS never touches the core.
package platform

import (
	"context"
	"errors"
)

// ErrUnsupported means the backend cannot do this on the current OS. The
// engine skips the step silently instead of logging it as a failure.
var ErrUnsupported = errors.New("not supported on this platform")

// Window identifies the top-level window that hosts a Claude session.
type Window struct {
	PID    int
	Handle uintptr
}

// Notification is a desktop toast. ID groups toasts: a new toast with the
// same ID replaces the old one, and ClearToast removes it. Sticky asks the OS
// to keep it on screen until dismissed, where the OS supports that.
type Notification struct {
	ID     string
	Title  string
	Body   string
	Sticky bool
}

// MediaSession is one media player the backend paused. Title is part of the
// identity because browsers report every tab under the same app ID.
type MediaSession struct {
	App   string `json:"app"`
	Title string `json:"title"`
}

type Platform interface {
	Name() string

	// HostWindow walks up the process tree from pid and returns the first
	// ancestor that owns a visible top-level window.
	HostWindow(pid int) (Window, error)
	IsForeground(w Window) (bool, error)
	Focus(w Window) error

	// PauseMedia pauses only what is playing and returns exactly that, so
	// ResumeMedia never restarts something the user paused themselves.
	PauseMedia(ctx context.Context) ([]MediaSession, error)
	ResumeMedia(ctx context.Context, sessions []MediaSession) error

	// PlaySound plays path, or the platform's default alert sound if empty.
	PlaySound(ctx context.Context, path string) error
	Toast(ctx context.Context, n Notification) error
	ClearToast(ctx context.Context, id string) error
}

// Unsupported is embedded by backends that are not implemented yet, so a new
// port can fill in one method at a time.
type Unsupported struct{}

func (Unsupported) HostWindow(int) (Window, error)    { return Window{}, ErrUnsupported }
func (Unsupported) IsForeground(Window) (bool, error) { return false, ErrUnsupported }
func (Unsupported) Focus(Window) error                { return ErrUnsupported }
func (Unsupported) PauseMedia(context.Context) ([]MediaSession, error) {
	return nil, ErrUnsupported
}
func (Unsupported) ResumeMedia(context.Context, []MediaSession) error { return ErrUnsupported }
func (Unsupported) PlaySound(context.Context, string) error           { return ErrUnsupported }
func (Unsupported) Toast(context.Context, Notification) error         { return ErrUnsupported }
func (Unsupported) ClearToast(context.Context, string) error          { return ErrUnsupported }
