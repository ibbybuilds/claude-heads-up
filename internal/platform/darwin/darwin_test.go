package darwin

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/ibbybuilds/claude-heads-up/internal/platform"
)

var _ platform.Platform = (*Backend)(nil)

type call struct {
	name string
	args []string
}

// fake answers commands by name and records every call.
type fake struct {
	calls   []call
	answers map[string]string // key: command name, or "jxa:<first script line>"
	err     error
}

func (f *fake) run(_ context.Context, name string, args ...string) ([]byte, error) {
	f.calls = append(f.calls, call{name, args})
	if f.err != nil {
		return nil, f.err
	}
	key := name
	if name == "osascript" && len(args) >= 4 {
		key = "jxa:" + firstLine(args[3])
	}
	return []byte(f.answers[key]), nil
}

func firstLine(s string) string { return strings.SplitN(s, "\n", 2)[0] }

func newFake(answers map[string]string) (*Backend, *fake) {
	f := &fake{answers: answers}
	return &Backend{run: f.run}, f
}

func TestParseParents(t *testing.T) {
	got := parseParents("  1     0\n 400   1\n 500 400\ngarbage line here\n\n")
	if got[500] != 400 || got[400] != 1 || len(got) != 3 {
		t.Errorf("parseParents = %v", got)
	}
}

func TestHostWindowWalksToFirstGUIAncestor(t *testing.T) {
	b, _ := newFake(map[string]string{
		"ps":                              "1 0\n300 1\n400 300\n500 400\n600 500",
		"jxa:" + firstLine(guiPIDsScript): "[300, 999]",
	})
	w, err := b.HostWindow(600)
	if err != nil || w.PID != 300 {
		t.Fatalf("HostWindow = %+v, %v; want pid 300", w, err)
	}
}

func TestHostWindowNoGUIAncestor(t *testing.T) {
	b, _ := newFake(map[string]string{
		"ps":                              "1 0\n400 1\n500 400",
		"jxa:" + firstLine(guiPIDsScript): "[999]",
	})
	if _, err := b.HostWindow(500); !errors.Is(err, errNoHost) {
		t.Errorf("err = %v, want errNoHost", err)
	}
}

func TestHostWindowSurvivesCycle(t *testing.T) {
	b, _ := newFake(map[string]string{
		"ps":                              "500 600\n600 500",
		"jxa:" + firstLine(guiPIDsScript): "[]",
	})
	if _, err := b.HostWindow(500); !errors.Is(err, errNoHost) {
		t.Errorf("err = %v, want errNoHost", err)
	}
}

func TestIsForeground(t *testing.T) {
	b, _ := newFake(map[string]string{"jxa:" + firstLine(frontPIDScript): "300\n"})
	if ok, err := b.IsForeground(platform.Window{PID: 300}); err != nil || !ok {
		t.Errorf("IsForeground(300) = %v, %v", ok, err)
	}
	if ok, _ := b.IsForeground(platform.Window{PID: 301}); ok {
		t.Error("IsForeground(301) = true")
	}
}

func TestFocusPassesPIDAsArgument(t *testing.T) {
	b, f := newFake(nil)
	if err := b.Focus(platform.Window{PID: 300}); err != nil {
		t.Fatal(err)
	}
	args := f.calls[0].args
	if args[len(args)-1] != "300" {
		t.Errorf("args = %v", args)
	}
}

func TestToastNeverSplicesTextIntoScript(t *testing.T) {
	b, f := newFake(nil)
	title := `my-app"); doShellScript("rm -rf ~`
	if err := b.Toast(context.Background(), platform.Notification{Title: title, Body: "Claude finished"}); err != nil {
		t.Fatal(err)
	}
	args := f.calls[0].args
	if args[3] != toastScript {
		t.Error("script text was modified")
	}
	if args[4] != title || args[5] != "Claude finished" {
		t.Errorf("argv = %q", args[4:])
	}
}

func TestClearToastUnsupported(t *testing.T) {
	b, _ := newFake(nil)
	if err := b.ClearToast(context.Background(), "a"); !errors.Is(err, platform.ErrUnsupported) {
		t.Errorf("err = %v", err)
	}
}

func TestPlaySound(t *testing.T) {
	b, f := newFake(nil)
	for _, path := range []string{"", "/tmp/chime.wav"} {
		if err := b.PlaySound(context.Background(), path); err != nil {
			t.Fatal(err)
		}
	}
	if f.calls[0].name != "afplay" || f.calls[0].args[0] != defaultSound || f.calls[1].args[0] != "/tmp/chime.wav" {
		t.Errorf("calls = %+v", f.calls)
	}
}

func TestPauseMediaParsesResult(t *testing.T) {
	b, f := newFake(map[string]string{
		"jxa:" + firstLine(mediaScript): `[{"app":"Spotify","title":""},{"app":"Brave Browser","title":""}]`,
	})
	got, err := b.PauseMedia(context.Background())
	if err != nil || len(got) != 2 || got[1].App != "Brave Browser" {
		t.Fatalf("PauseMedia = %+v, %v", got, err)
	}
	args := f.calls[0].args
	if args[4] != "pause" || args[5] != "[]" {
		t.Errorf("argv = %q", args[4:])
	}
}

func TestResumeMediaPassesTargets(t *testing.T) {
	b, f := newFake(map[string]string{"jxa:" + firstLine(mediaScript): "[]"})
	err := b.ResumeMedia(context.Background(), []platform.MediaSession{{App: "Spotify"}})
	if err != nil {
		t.Fatal(err)
	}
	args := f.calls[0].args
	if args[4] != "resume" || !strings.Contains(args[5], `"app":"Spotify"`) {
		t.Errorf("argv = %q", args[4:])
	}
}

func TestMediaScriptBadOutput(t *testing.T) {
	b, _ := newFake(map[string]string{"jxa:" + firstLine(mediaScript): "execution error: not allowed"})
	if _, err := b.PauseMedia(context.Background()); err == nil {
		t.Error("want error for non-JSON output")
	}
}
