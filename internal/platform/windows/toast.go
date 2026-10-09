//go:build windows

package windows

import (
	"context"
	_ "embed"

	"github.com/ibbybuilds/claude-heads-up/internal/platform"
)

//go:embed toast.ps1
var toastScript string

func (*Backend) Toast(ctx context.Context, n platform.Notification) error {
	sticky := "0"
	if n.Sticky {
		sticky = "1"
	}
	_, err := runPowerShell(ctx, toastScript,
		"HEADS_UP_TITLE="+n.Title,
		"HEADS_UP_BODY="+n.Body,
		"HEADS_UP_STICKY="+sticky)
	return err
}
