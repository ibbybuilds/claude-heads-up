// Package engine decides what happens on an alert or a resume. It holds all
// the rules (triggers, throttle, focus skip, session ownership) and talks to
// the OS only through platform.Platform.
package engine

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/ibbybuilds/claude-heads-up/internal/config"
	"github.com/ibbybuilds/claude-heads-up/internal/hookinput"
	"github.com/ibbybuilds/claude-heads-up/internal/platform"
	"github.com/ibbybuilds/claude-heads-up/internal/state"
)

const (
	mediaTimeout = 5 * time.Second
	ntfyTimeout  = 3 * time.Second
	stepTimeout  = 2 * time.Second
)

type Notifier interface {
	Send(ctx context.Context, title, body string) error
}

type Engine struct {
	Platform platform.Platform
	Store    *state.Store
	Notifier Notifier // nil when phone push is off
	Config   config.Config
	Log      *slog.Logger
	Now      func() time.Time
	// DefaultSound plays when the config names no sound file.
	DefaultSound string
	// PID is where the host window search starts: the hook process itself.
	PID int
}

// Alert gets the user's attention. With force set (the `test` command) it
// ignores triggers, throttle and the focus skip.
func (e *Engine) Alert(ctx context.Context, ev hookinput.Event, force bool) {
	log := e.Log.With("session", ev.SessionID, "event", ev.HookEventName)

	if !force && !e.triggerEnabled(ev) {
		return
	}
	if !force {
		window := time.Duration(e.Config.ThrottleSeconds) * time.Second
		allowed, err := e.Store.AllowAlert(ev.SessionID, window)
		if err != nil {
			log.Warn("throttle state", "err", err)
		}
		if !allowed {
			log.Debug("throttled")
			return
		}
	}

	host, hostErr := e.Platform.HostWindow(e.PID)
	if hostErr != nil && !errors.Is(hostErr, platform.ErrUnsupported) {
		log.Warn("find host window", "err", hostErr)
	}
	if !force && e.Config.SkipWhenFocused && hostErr == nil {
		if focused, err := e.Platform.IsForeground(host); err == nil && focused {
			log.Debug("host window already focused, skipping")
			return
		}
	}

	title, body := ev.Project(), ev.Reason()

	if e.Config.Media.Enabled {
		e.step(ctx, log, "pause media", mediaTimeout, func(ctx context.Context) error {
			return e.pauseMedia(ctx, ev.SessionID)
		})
	}
	if e.Config.Sound.Enabled {
		e.step(ctx, log, "play sound", stepTimeout, func(ctx context.Context) error {
			path := e.Config.Sound.File
			if path == "" {
				path = e.DefaultSound
			}
			return e.Platform.PlaySound(ctx, path)
		})
	}
	if e.Config.Focus.Enabled && hostErr == nil {
		e.step(ctx, log, "focus", stepTimeout, func(context.Context) error {
			return e.Platform.Focus(host)
		})
	}
	if e.Config.Toast.Enabled {
		e.step(ctx, log, "toast", mediaTimeout, func(ctx context.Context) error {
			return e.Platform.Toast(ctx, platform.Notification{
				Title: title, Body: body, Sticky: e.Config.Toast.Sticky,
			})
		})
	}
	if e.Notifier != nil {
		e.step(ctx, log, "ntfy", ntfyTimeout, func(ctx context.Context) error {
			return e.Notifier.Send(ctx, title, body)
		})
	}
}

// Resume plays the media an alert paused, but only for a reply in the session
// that paused it. A reply in another session must not restart the video.
func (e *Engine) Resume(ctx context.Context, ev hookinput.Event) {
	if !e.Config.Media.Enabled || !e.Store.HasPause() {
		return
	}
	log := e.Log.With("session", ev.SessionID, "event", ev.HookEventName)

	var media []platform.MediaSession
	err := e.Store.WithLock(func() error {
		p, err := e.Store.LoadPause()
		if err != nil || p == nil || p.OwnerSession != ev.SessionID {
			return err
		}
		media = p.Media
		return e.Store.ClearPause()
	})
	if err != nil {
		log.Warn("read pause state", "err", err)
		return
	}
	if len(media) == 0 {
		return
	}
	e.step(ctx, log, "resume media", mediaTimeout, func(ctx context.Context) error {
		return e.Platform.ResumeMedia(ctx, media)
	})
}

func (e *Engine) triggerEnabled(ev hookinput.Event) bool {
	t := e.Config.Triggers
	switch {
	case ev.HookEventName == "Stop":
		return t.Stop
	case ev.HookEventName == "Notification":
		return t.Permission
	case ev.ToolName == "AskUserQuestion":
		return t.Question
	default:
		return false
	}
}

// pauseMedia runs the slow platform call outside the lock so a concurrent
// resume is not starved, then merges the result into the record. An existing
// record keeps its owner: that session alerted first.
func (e *Engine) pauseMedia(ctx context.Context, session string) error {
	media, err := e.Platform.PauseMedia(ctx)
	if err != nil || len(media) == 0 {
		return err
	}
	return e.Store.WithLock(func() error {
		p, err := e.Store.LoadPause()
		if err != nil {
			e.Log.Warn("replacing unreadable pause state", "err", err)
			p = nil
		}
		if p == nil {
			p = &state.Pause{OwnerSession: session, PausedAt: e.Now()}
		}
		p.Media = append(p.Media, media...)
		return e.Store.SavePause(*p)
	})
}

// step runs one action in isolation: a failure, timeout or panic is logged
// and the next step still runs.
func (e *Engine) step(ctx context.Context, log *slog.Logger, name string, timeout time.Duration, fn func(context.Context) error) {
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	defer func() {
		if r := recover(); r != nil {
			log.Error("step panicked", "step", name, "panic", fmt.Sprint(r))
		}
	}()
	err := fn(ctx)
	switch {
	case err == nil:
		log.Debug("step done", "step", name)
	case errors.Is(err, platform.ErrUnsupported):
		log.Debug("step unsupported", "step", name)
	default:
		log.Warn("step failed", "step", name, "err", err)
	}
}
