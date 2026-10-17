package darwin

import (
	"context"

	"github.com/ibbybuilds/claude-heads-up/internal/platform"
)

// Played when the config names no sound file and the built-in chime is
// unavailable.
const defaultSound = "/System/Library/Sounds/Glass.aiff"

const toastScript = `function run(argv) {
	const app = Application.currentApplication();
	app.includeStandardAdditions = true;
	app.displayNotification(argv[1], {withTitle: argv[0]});
}`

// PlaySound blocks until the sound ends, because the hook process exits right
// after the alert and would cut an async sound off.
func (b *Backend) PlaySound(ctx context.Context, path string) error {
	if path == "" {
		path = defaultSound
	}
	_, err := b.run(ctx, "afplay", path)
	return err
}

// Toast ignores ID and Sticky: notifications posted by osascript can be
// neither replaced nor removed, and their style is a per-user System Settings
// choice.
func (b *Backend) Toast(ctx context.Context, n platform.Notification) error {
	_, err := b.jxa(ctx, toastScript, n.Title, n.Body)
	return err
}

func (*Backend) ClearToast(context.Context, string) error { return platform.ErrUnsupported }
