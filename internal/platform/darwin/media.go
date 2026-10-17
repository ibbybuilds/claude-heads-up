package darwin

import (
	"context"
	_ "embed"
	"encoding/json"
	"fmt"

	"github.com/ibbybuilds/claude-heads-up/internal/platform"
)

//go:embed media.js
var mediaScript string

func (b *Backend) PauseMedia(ctx context.Context) ([]platform.MediaSession, error) {
	return b.runMedia(ctx, "pause", nil)
}

func (b *Backend) ResumeMedia(ctx context.Context, sessions []platform.MediaSession) error {
	_, err := b.runMedia(ctx, "resume", sessions)
	return err
}

// runMedia returns the apps the script acted on.
func (b *Backend) runMedia(ctx context.Context, action string, targets []platform.MediaSession) ([]platform.MediaSession, error) {
	if targets == nil {
		targets = []platform.MediaSession{}
	}
	targetJSON, err := json.Marshal(targets)
	if err != nil {
		return nil, err
	}
	out, err := b.jxa(ctx, mediaScript, action, string(targetJSON))
	if err != nil {
		return nil, err
	}
	var sessions []platform.MediaSession
	if err := json.Unmarshal([]byte(out), &sessions); err != nil {
		return nil, fmt.Errorf("media script output %q: %w", out, err)
	}
	return sessions, nil
}
