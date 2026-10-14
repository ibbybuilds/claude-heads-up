// Package state stores what survives between hook invocations: which media
// was paused and by which Claude session, and when each session last alerted.
// Every hook is a separate short-lived process, so this lives on disk.
package state

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"time"

	"github.com/ibbybuilds/claude-heads-up/internal/platform"
)

const (
	pauseFile    = "paused.json"
	throttleFile = "throttle.json"
	lockFile     = "state.lock"
	alertsDir    = "alerts"

	// A pause older than this is from a crashed or closed session.
	PauseTTL      = 6 * time.Hour
	throttleTTL   = time.Hour
	lockWait      = time.Second
	lockStaleAge  = 10 * time.Second
	lockRetryStep = 25 * time.Millisecond
)

var ErrLocked = errors.New("state is locked by another hook")

// Pause records media that an alert paused. There is at most one, because
// media is global while Claude sessions are not.
type Pause struct {
	OwnerSession string                  `json:"ownerSession"`
	PausedAt     time.Time               `json:"pausedAt"`
	Media        []platform.MediaSession `json:"media"`
}

type Store struct {
	dir string
	now func() time.Time
}

func NewStore(dir string, now func() time.Time) *Store {
	return &Store{dir: dir, now: now}
}

// DefaultDir is the per-user cache dir, e.g. %LOCALAPPDATA%\claude-heads-up.
func DefaultDir(app string) (string, error) {
	dir, err := os.UserCacheDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, app), nil
}

func (s *Store) Dir() string { return s.dir }

// HasPause is a lock-free check for the hot path: PostToolUse fires on every
// tool call and almost never has anything to resume.
func (s *Store) HasPause() bool {
	_, err := os.Stat(filepath.Join(s.dir, pauseFile))
	return err == nil
}

// WithLock runs fn while holding an exclusive lock on the state dir.
func (s *Store) WithLock(fn func() error) error {
	if err := os.MkdirAll(s.dir, 0o755); err != nil {
		return err
	}
	path := filepath.Join(s.dir, lockFile)
	deadline := time.Now().Add(lockWait)
	for {
		f, err := os.OpenFile(path, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o600)
		if err == nil {
			f.Close()
			defer os.Remove(path)
			return fn()
		}
		if !errors.Is(err, fs.ErrExist) {
			return err
		}
		if info, statErr := os.Stat(path); statErr == nil && time.Since(info.ModTime()) > lockStaleAge {
			os.Remove(path)
			continue
		}
		if time.Now().After(deadline) {
			return ErrLocked
		}
		time.Sleep(lockRetryStep)
	}
}

// LoadPause returns nil when there is no pause or it has expired.
func (s *Store) LoadPause() (*Pause, error) {
	var p Pause
	ok, err := s.read(pauseFile, &p)
	if err != nil || !ok {
		return nil, err
	}
	if s.now().Sub(p.PausedAt) > PauseTTL {
		return nil, s.ClearPause()
	}
	return &p, nil
}

func (s *Store) SavePause(p Pause) error { return s.write(pauseFile, p) }

func (s *Store) ClearPause() error {
	err := os.Remove(filepath.Join(s.dir, pauseFile))
	if errors.Is(err, fs.ErrNotExist) {
		return nil
	}
	return err
}

// MarkAlert records that session has an unanswered alert, so the next resume
// knows there is a toast to clear. One empty file per session keeps the check
// a single stat, cheap enough for PostToolUse.
func (s *Store) MarkAlert(session string) error {
	dir := filepath.Join(s.dir, alertsDir)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(dir, safeName(session)), nil, 0o600)
}

// TakeAlert reports whether session had an unanswered alert and clears it.
func (s *Store) TakeAlert(session string) bool {
	return os.Remove(filepath.Join(s.dir, alertsDir, safeName(session))) == nil
}

// safeName keeps a session ID from escaping the alerts dir. Claude session
// IDs are UUIDs, so this only matters for malformed input.
func safeName(session string) string {
	out := []rune(session)
	for i, r := range out {
		ok := r >= 'a' && r <= 'z' || r >= 'A' && r <= 'Z' || r >= '0' && r <= '9' || r == '-' || r == '_'
		if !ok {
			out[i] = '_'
		}
	}
	return string(out)
}

// AllowAlert reports whether session may alert now, and records the attempt
// if so. Claude often fires Stop and a notification within a second.
func (s *Store) AllowAlert(session string, window time.Duration) (bool, error) {
	last := map[string]time.Time{}
	if _, err := s.read(throttleFile, &last); err != nil {
		last = map[string]time.Time{}
	}
	now := s.now()
	if t, ok := last[session]; ok && now.Sub(t) < window {
		return false, nil
	}
	for id, t := range last {
		if now.Sub(t) > throttleTTL {
			delete(last, id)
		}
	}
	last[session] = now
	return true, s.write(throttleFile, last)
}

func (s *Store) read(name string, v any) (bool, error) {
	data, err := os.ReadFile(filepath.Join(s.dir, name))
	if errors.Is(err, fs.ErrNotExist) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	if err := json.Unmarshal(data, v); err != nil {
		return false, fmt.Errorf("corrupt %s: %w", name, err)
	}
	return true, nil
}

// write goes through a temp file and rename so a crash never leaves half a
// JSON document behind.
func (s *Store) write(name string, v any) error {
	if err := os.MkdirAll(s.dir, 0o755); err != nil {
		return err
	}
	data, err := json.Marshal(v)
	if err != nil {
		return err
	}
	tmp, err := os.CreateTemp(s.dir, name+".*.tmp")
	if err != nil {
		return err
	}
	if _, err := tmp.Write(data); err != nil {
		tmp.Close()
		os.Remove(tmp.Name())
		return err
	}
	if err := tmp.Close(); err != nil {
		os.Remove(tmp.Name())
		return err
	}
	return os.Rename(tmp.Name(), filepath.Join(s.dir, name))
}
