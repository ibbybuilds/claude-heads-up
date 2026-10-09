package current

import (
	"github.com/ibbybuilds/claude-heads-up/internal/platform"
	"github.com/ibbybuilds/claude-heads-up/internal/platform/linux"
)

func New() platform.Platform { return linux.New() }
