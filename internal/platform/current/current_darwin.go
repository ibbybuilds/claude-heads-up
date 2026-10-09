package current

import (
	"github.com/ibbybuilds/claude-heads-up/internal/platform"
	"github.com/ibbybuilds/claude-heads-up/internal/platform/darwin"
)

func New() platform.Platform { return darwin.New() }
