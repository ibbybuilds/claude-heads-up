// Package current picks the backend for the OS the binary was built for.
package current

import (
	"github.com/ibbybuilds/claude-heads-up/internal/platform"
	"github.com/ibbybuilds/claude-heads-up/internal/platform/windows"
)

func New() platform.Platform { return windows.New() }
