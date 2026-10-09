//go:build windows

package windows

import (
	"context"
	_ "embed"
)

//go:embed toast.ps1
var toastScript string

func (*Backend) Toast(ctx context.Context, title, body string) error {
	_, err := runPowerShell(ctx, toastScript,
		"HEADS_UP_TITLE="+title,
		"HEADS_UP_BODY="+body)
	return err
}
