// Package darwin is the macOS backend. Everything goes through command-line
// tools that ship with macOS (ps, afplay, osascript), so the package needs no
// cgo and cross-compiles like the rest of the binary.
//
// It has no build tag on purpose: the logic compiles and is unit-tested on
// every OS, with the commands replaced by a fake runner.
package darwin

import (
	"bytes"
	"context"
	"fmt"
	"os/exec"
	"strings"
)

// runner executes a command and returns its stdout.
type runner func(ctx context.Context, name string, args ...string) ([]byte, error)

type Backend struct {
	run runner
}

func New() *Backend { return &Backend{run: execRunner} }

func (*Backend) Name() string { return "darwin (beta)" }

func execRunner(ctx context.Context, name string, args ...string) ([]byte, error) {
	var stderr bytes.Buffer
	cmd := exec.CommandContext(ctx, name, args...)
	cmd.Stderr = &stderr
	out, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("%s: %w: %s", name, err, strings.TrimSpace(stderr.String()))
	}
	return out, nil
}

// jxa runs a JavaScript for Automation script. Inputs travel as argv, never
// spliced into the source, so titles and paths cannot inject code.
func (b *Backend) jxa(ctx context.Context, script string, args ...string) (string, error) {
	out, err := b.run(ctx, "osascript", append([]string{"-l", "JavaScript", "-e", script}, args...)...)
	return strings.TrimSpace(string(out)), err
}
