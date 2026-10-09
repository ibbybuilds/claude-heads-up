// Package logging writes to a small rotating file. Hooks must never write to
// stdout or stderr, because Claude Code shows that output to the user.
package logging

import (
	"io"
	"log/slog"
	"os"
	"path/filepath"
)

const (
	FileName = "heads-up.log"
	maxSize  = 1 << 20
)

// Open returns a logger writing to dir/heads-up.log, rotating to .1 once the
// file passes 1 MB. If the file cannot be opened, logs are discarded rather
// than failing the hook.
func Open(dir string, debug bool) (*slog.Logger, io.Closer) {
	level := slog.LevelInfo
	if debug {
		level = slog.LevelDebug
	}
	path := filepath.Join(dir, FileName)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return discard(), io.NopCloser(nil)
	}
	if info, err := os.Stat(path); err == nil && info.Size() > maxSize {
		os.Rename(path, path+".1")
	}
	f, err := os.OpenFile(path, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o600)
	if err != nil {
		return discard(), io.NopCloser(nil)
	}
	return slog.New(slog.NewTextHandler(f, &slog.HandlerOptions{Level: level})), f
}

func discard() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}
