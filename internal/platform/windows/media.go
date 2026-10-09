//go:build windows

package windows

import (
	"bytes"
	"context"
	_ "embed"
	"encoding/json"
	"fmt"

	"github.com/ibbybuilds/claude-heads-up/internal/platform"
)

//go:embed media.ps1
var mediaScript string

func (*Backend) PauseMedia(ctx context.Context) ([]platform.MediaSession, error) {
	return runMedia(ctx, "pause", nil)
}

func (*Backend) ResumeMedia(ctx context.Context, sessions []platform.MediaSession) error {
	_, err := runMedia(ctx, "resume", sessions)
	return err
}

// runMedia returns the sessions the script acted on.
func runMedia(ctx context.Context, action string, targets []platform.MediaSession) ([]platform.MediaSession, error) {
	targetJSON, err := json.Marshal(targets)
	if err != nil {
		return nil, err
	}
	out, err := runPowerShell(ctx, mediaScript,
		"HEADS_UP_ACTION="+action,
		"HEADS_UP_MEDIA="+string(targetJSON))
	if err != nil {
		return nil, err
	}

	var sessions []platform.MediaSession
	out = bytes.TrimPrefix(bytes.TrimSpace(out), []byte("\xef\xbb\xbf"))
	if err := json.Unmarshal(out, &sessions); err != nil {
		return nil, fmt.Errorf("media script output %q: %w", out, err)
	}
	return sessions, nil
}
