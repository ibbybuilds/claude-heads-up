// Package linux is the Linux backend. Not implemented yet; see
// docs/adding-a-platform.md.
package linux

import "github.com/ibbybuilds/claude-heads-up/internal/platform"

type Backend struct{ platform.Unsupported }

func New() *Backend { return &Backend{} }

func (*Backend) Name() string { return "linux (not implemented)" }
