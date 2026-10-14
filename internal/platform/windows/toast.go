//go:build windows

package windows

import (
	"context"
	_ "embed"

	"github.com/ibbybuilds/claude-heads-up/internal/platform"
)

//go:embed toast.ps1
var toastScript string

// Windows caps toast tags at 64 characters.
const maxTagLen = 64

func (*Backend) Toast(ctx context.Context, n platform.Notification) error {
	sticky := "0"
	if n.Sticky {
		sticky = "1"
	}
	_, err := runPowerShell(ctx, toastScript,
		"HEADS_UP_ACTION=show",
		"HEADS_UP_TAG="+toastTag(n.ID),
		"HEADS_UP_TITLE="+n.Title,
		"HEADS_UP_BODY="+n.Body,
		"HEADS_UP_STICKY="+sticky)
	return err
}

func (*Backend) ClearToast(ctx context.Context, id string) error {
	_, err := runPowerShell(ctx, toastScript,
		"HEADS_UP_ACTION=clear",
		"HEADS_UP_TAG="+toastTag(id))
	return err
}

func toastTag(id string) string {
	if len(id) > maxTagLen {
		return id[:maxTagLen]
	}
	return id
}
