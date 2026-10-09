package state

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/ibbybuilds/claude-heads-up/internal/platform"
)

type clock struct{ t time.Time }

func (c *clock) now() time.Time { return c.t }

func newTestStore(t *testing.T) (*Store, *clock) {
	c := &clock{t: time.Date(2026, 10, 9, 12, 0, 0, 0, time.UTC)}
	return NewStore(t.TempDir(), c.now), c
}

func TestPauseRoundTrip(t *testing.T) {
	s, c := newTestStore(t)
	if s.HasPause() {
		t.Fatal("new store has pause")
	}
	want := Pause{
		OwnerSession: "a",
		PausedAt:     c.t,
		Media:        []platform.MediaSession{{App: "Brave", Title: "video"}},
	}
	if err := s.SavePause(want); err != nil {
		t.Fatal(err)
	}
	if !s.HasPause() {
		t.Fatal("HasPause after save = false")
	}
	got, err := s.LoadPause()
	if err != nil || got == nil || got.OwnerSession != "a" || len(got.Media) != 1 {
		t.Fatalf("LoadPause = %+v, %v", got, err)
	}
	if err := s.ClearPause(); err != nil {
		t.Fatal(err)
	}
	if err := s.ClearPause(); err != nil {
		t.Errorf("second clear: %v", err)
	}
}

func TestPauseExpires(t *testing.T) {
	s, c := newTestStore(t)
	if err := s.SavePause(Pause{OwnerSession: "a", PausedAt: c.t}); err != nil {
		t.Fatal(err)
	}
	c.t = c.t.Add(PauseTTL + time.Minute)
	got, err := s.LoadPause()
	if err != nil || got != nil {
		t.Fatalf("expired pause: got %+v, %v", got, err)
	}
	if s.HasPause() {
		t.Error("expired pause file was not removed")
	}
}

func TestAllowAlertThrottles(t *testing.T) {
	s, c := newTestStore(t)
	window := 2 * time.Second
	if ok, _ := s.AllowAlert("a", window); !ok {
		t.Fatal("first alert blocked")
	}
	if ok, _ := s.AllowAlert("a", window); ok {
		t.Error("second alert within window allowed")
	}
	if ok, _ := s.AllowAlert("b", window); !ok {
		t.Error("other session blocked")
	}
	c.t = c.t.Add(3 * time.Second)
	if ok, _ := s.AllowAlert("a", window); !ok {
		t.Error("alert after window blocked")
	}
}

func TestWithLock(t *testing.T) {
	s, _ := newTestStore(t)
	err := s.WithLock(func() error {
		if inner := s.WithLock(func() error { return nil }); !errors.Is(inner, ErrLocked) {
			t.Errorf("nested lock: got %v, want ErrLocked", inner)
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(s.Dir(), lockFile)); !os.IsNotExist(err) {
		t.Error("lock file left behind")
	}
}

func TestWithLockBreaksStaleLock(t *testing.T) {
	s, _ := newTestStore(t)
	path := filepath.Join(s.Dir(), lockFile)
	os.WriteFile(path, nil, 0o600)
	old := time.Now().Add(-time.Minute)
	os.Chtimes(path, old, old)
	ran := false
	if err := s.WithLock(func() error { ran = true; return nil }); err != nil || !ran {
		t.Fatalf("stale lock not broken: ran=%v err=%v", ran, err)
	}
}
