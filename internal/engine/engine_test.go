package engine

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"slices"
	"testing"
	"time"

	"github.com/ibbybuilds/claude-heads-up/internal/config"
	"github.com/ibbybuilds/claude-heads-up/internal/hookinput"
	"github.com/ibbybuilds/claude-heads-up/internal/platform"
	"github.com/ibbybuilds/claude-heads-up/internal/state"
)

type fakePlatform struct {
	calls      []string
	foreground bool
	playing    []platform.MediaSession
	resumed    []platform.MediaSession
	failSound  bool
	panicFocus bool
	noHost     bool
	soundPath  string
	toast      platform.Notification

	toastUnsupported bool
}

func (f *fakePlatform) Name() string { return "fake" }

func (f *fakePlatform) HostWindow(int) (platform.Window, error) {
	if f.noHost {
		return platform.Window{}, platform.ErrUnsupported
	}
	return platform.Window{PID: 42}, nil
}

func (f *fakePlatform) IsForeground(platform.Window) (bool, error) { return f.foreground, nil }

func (f *fakePlatform) Focus(platform.Window) error {
	f.calls = append(f.calls, "focus")
	if f.panicFocus {
		panic("boom")
	}
	return nil
}

func (f *fakePlatform) PauseMedia(context.Context) ([]platform.MediaSession, error) {
	f.calls = append(f.calls, "pause")
	paused := f.playing
	f.playing = nil
	return paused, nil
}

func (f *fakePlatform) ResumeMedia(_ context.Context, s []platform.MediaSession) error {
	f.calls = append(f.calls, "resume")
	f.resumed = append(f.resumed, s...)
	return nil
}

func (f *fakePlatform) PlaySound(_ context.Context, path string) error {
	f.calls = append(f.calls, "sound")
	f.soundPath = path
	if f.failSound {
		return errors.New("no audio device")
	}
	return nil
}

func (f *fakePlatform) ClearToast(_ context.Context, id string) error {
	f.calls = append(f.calls, "clear toast:"+id)
	return nil
}

func (f *fakePlatform) Toast(_ context.Context, n platform.Notification) error {
	f.calls = append(f.calls, "toast")
	f.toast = n
	if f.toastUnsupported {
		return platform.ErrUnsupported
	}
	return nil
}

type fakeNotifier struct{ sent []string }

func (n *fakeNotifier) Send(_ context.Context, title, body string) error {
	n.sent = append(n.sent, title+": "+body)
	return nil
}

type harness struct {
	e     *Engine
	p     *fakePlatform
	n     *fakeNotifier
	clock time.Time
}

func newHarness(t *testing.T) *harness {
	h := &harness{
		p:     &fakePlatform{playing: []platform.MediaSession{{App: "Brave", Title: "reel"}}},
		n:     &fakeNotifier{},
		clock: time.Date(2026, 10, 9, 12, 0, 0, 0, time.UTC),
	}
	now := func() time.Time { return h.clock }
	h.e = &Engine{
		Platform: h.p,
		Store:    state.NewStore(t.TempDir(), now),
		Notifier: h.n,
		Config:   config.Default(),
		Log:      slog.New(slog.NewTextHandler(io.Discard, nil)),
		Now:      now,

		DefaultSound: "chime.wav",
	}
	return h
}

func stop(session string) hookinput.Event {
	return hookinput.Event{SessionID: session, HookEventName: "Stop", Cwd: "/code/my-app"}
}

func prompt(session string) hookinput.Event {
	return hookinput.Event{SessionID: session, HookEventName: "UserPromptSubmit"}
}

func TestAlertRunsAllStepsInOrder(t *testing.T) {
	h := newHarness(t)
	h.e.Alert(context.Background(), stop("a"), false)

	want := []string{"pause", "sound", "focus", "toast"}
	if !slices.Equal(h.p.calls, want) {
		t.Errorf("calls = %v, want %v", h.p.calls, want)
	}
	if len(h.n.sent) != 1 || h.n.sent[0] != "my-app: Claude finished" {
		t.Errorf("ntfy sent = %v", h.n.sent)
	}
}

func TestAlertThenResumeSameSession(t *testing.T) {
	h := newHarness(t)
	h.e.Alert(context.Background(), stop("a"), false)
	h.e.Resume(context.Background(), prompt("a"))

	if len(h.p.resumed) != 1 || h.p.resumed[0].Title != "reel" {
		t.Fatalf("resumed = %v", h.p.resumed)
	}
	if h.e.Store.HasPause() {
		t.Error("pause record not cleared after resume")
	}
}

func TestResumeIgnoresOtherSession(t *testing.T) {
	h := newHarness(t)
	h.e.Alert(context.Background(), stop("a"), false)
	h.e.Resume(context.Background(), prompt("b"))

	if len(h.p.resumed) != 0 {
		t.Fatalf("session b resumed session a's media: %v", h.p.resumed)
	}
	if !h.e.Store.HasPause() {
		t.Error("pause record lost")
	}
}

func TestResumeClearsOwnToastOnce(t *testing.T) {
	h := newHarness(t)
	h.e.Config.Media.Enabled = false
	h.e.Alert(context.Background(), stop("a"), false)
	if h.p.toast.ID != "a" {
		t.Fatalf("toast ID = %q, want the session ID", h.p.toast.ID)
	}

	h.p.calls = nil
	h.e.Resume(context.Background(), prompt("b"))
	if len(h.p.calls) != 0 {
		t.Errorf("reply in b touched a's toast: %v", h.p.calls)
	}
	h.e.Resume(context.Background(), prompt("a"))
	h.e.Resume(context.Background(), prompt("a"))
	if !slices.Equal(h.p.calls, []string{"clear toast:a"}) {
		t.Errorf("calls = %v, want one clear for a", h.p.calls)
	}
}

func TestFailedToastIsNotCleared(t *testing.T) {
	h := newHarness(t)
	h.p.toastUnsupported = true
	h.e.Config.Media.Enabled = false
	h.e.Alert(context.Background(), stop("a"), false)
	h.p.calls = nil
	h.e.Resume(context.Background(), prompt("a"))
	if len(h.p.calls) != 0 {
		t.Errorf("cleared a toast that was never shown: %v", h.p.calls)
	}
}

func TestResumeWithoutPauseDoesNothing(t *testing.T) {
	h := newHarness(t)
	h.e.Resume(context.Background(), prompt("a"))
	if len(h.p.calls) != 0 {
		t.Errorf("calls = %v", h.p.calls)
	}
}

func TestSecondAlertKeepsFirstOwner(t *testing.T) {
	h := newHarness(t)
	h.e.Alert(context.Background(), stop("a"), false)
	h.p.playing = []platform.MediaSession{{App: "Spotify", Title: "song"}}
	h.e.Alert(context.Background(), stop("b"), false)

	h.e.Resume(context.Background(), prompt("a"))
	if len(h.p.resumed) != 2 {
		t.Errorf("resumed = %v, want both sessions' media", h.p.resumed)
	}
}

func TestAlertThrottled(t *testing.T) {
	h := newHarness(t)
	h.e.Alert(context.Background(), stop("a"), false)
	h.p.calls = nil
	h.e.Alert(context.Background(), hookinput.Event{SessionID: "a", HookEventName: "Notification"}, false)
	if len(h.p.calls) != 0 {
		t.Errorf("throttled alert ran: %v", h.p.calls)
	}

	h.clock = h.clock.Add(3 * time.Second)
	h.e.Alert(context.Background(), stop("a"), false)
	if len(h.p.calls) == 0 {
		t.Error("alert after throttle window did not run")
	}
}

func TestAlertRunsWhenHostFocusedByDefault(t *testing.T) {
	h := newHarness(t)
	h.p.foreground = true
	h.e.Alert(context.Background(), stop("a"), false)
	if len(h.p.calls) == 0 {
		t.Error("default config skipped the alert for a focused window")
	}
}

func TestAlertSkippedWhenHostFocused(t *testing.T) {
	h := newHarness(t)
	h.e.Config.SkipWhenFocused = true
	h.p.foreground = true
	h.e.Alert(context.Background(), stop("a"), false)
	if len(h.p.calls) != 0 || len(h.n.sent) != 0 {
		t.Errorf("alert ran while focused: calls=%v sent=%v", h.p.calls, h.n.sent)
	}
}

func TestForceIgnoresFocusAndTriggers(t *testing.T) {
	h := newHarness(t)
	h.e.Config.SkipWhenFocused = true
	h.p.foreground = true
	h.e.Config.Triggers = config.Triggers{}
	h.e.Alert(context.Background(), stop("a"), true)
	if len(h.p.calls) == 0 {
		t.Error("forced alert did not run")
	}
}

func TestDisabledTrigger(t *testing.T) {
	h := newHarness(t)
	h.e.Config.Triggers.Stop = false
	h.e.Alert(context.Background(), stop("a"), false)
	if len(h.p.calls) != 0 {
		t.Errorf("disabled trigger ran: %v", h.p.calls)
	}
}

func TestTriggerMapping(t *testing.T) {
	e := &Engine{Config: config.Default()}
	tests := []struct {
		ev   hookinput.Event
		want bool
	}{
		{hookinput.Event{HookEventName: "Stop"}, true},
		{hookinput.Event{HookEventName: "Notification"}, true},
		{hookinput.Event{HookEventName: "PreToolUse", ToolName: "AskUserQuestion"}, true},
		{hookinput.Event{HookEventName: "PreToolUse", ToolName: "Bash"}, false},
		{hookinput.Event{HookEventName: "UserPromptSubmit"}, false},
	}
	for _, tt := range tests {
		if got := e.triggerEnabled(tt.ev); got != tt.want {
			t.Errorf("triggerEnabled(%s/%s) = %v", tt.ev.HookEventName, tt.ev.ToolName, got)
		}
	}
}

func TestFailingStepsDoNotStopOthers(t *testing.T) {
	h := newHarness(t)
	h.p.failSound = true
	h.p.panicFocus = true
	h.e.Alert(context.Background(), stop("a"), false)

	want := []string{"pause", "sound", "focus", "toast"}
	if !slices.Equal(h.p.calls, want) {
		t.Errorf("calls = %v, want %v", h.p.calls, want)
	}
	if len(h.n.sent) != 1 {
		t.Error("ntfy skipped after earlier failures")
	}
}

func TestNoHostWindowSkipsFocusOnly(t *testing.T) {
	h := newHarness(t)
	h.p.noHost = true
	h.e.Alert(context.Background(), stop("a"), false)
	want := []string{"pause", "sound", "toast"}
	if !slices.Equal(h.p.calls, want) {
		t.Errorf("calls = %v, want %v", h.p.calls, want)
	}
}

func TestSoundAndToastSettings(t *testing.T) {
	h := newHarness(t)
	h.e.Alert(context.Background(), stop("a"), false)
	if h.p.soundPath != "chime.wav" {
		t.Errorf("default sound = %q, want the built-in chime", h.p.soundPath)
	}
	if !h.p.toast.Sticky || h.p.toast.Title != "my-app" {
		t.Errorf("toast = %+v", h.p.toast)
	}

	h = newHarness(t)
	h.e.Config.Sound.File = "custom.wav"
	h.e.Config.Toast.Sticky = false
	h.e.Alert(context.Background(), stop("a"), false)
	if h.p.soundPath != "custom.wav" || h.p.toast.Sticky {
		t.Errorf("sound=%q toast=%+v", h.p.soundPath, h.p.toast)
	}
}

func TestMediaDisabled(t *testing.T) {
	h := newHarness(t)
	h.e.Config.Media.Enabled = false
	h.e.Alert(context.Background(), stop("a"), false)
	if slices.Contains(h.p.calls, "pause") || h.e.Store.HasPause() {
		t.Errorf("media paused while disabled: %v", h.p.calls)
	}
}
