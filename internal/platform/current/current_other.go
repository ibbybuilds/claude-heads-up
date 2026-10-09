//go:build !windows && !darwin && !linux

package current

import "github.com/ibbybuilds/claude-heads-up/internal/platform"

type unknown struct{ platform.Unsupported }

func (unknown) Name() string { return "unsupported OS" }

func New() platform.Platform { return unknown{} }
