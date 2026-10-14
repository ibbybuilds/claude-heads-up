// Package hookinput parses the JSON that Claude Code writes to a hook's stdin.
package hookinput

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"path/filepath"
)

const maxInput = 1 << 20

// Event is the subset of hook input the engine needs. Field names follow
// https://docs.claude.com/en/docs/claude-code/hooks.
type Event struct {
	SessionID        string `json:"session_id"`
	Cwd              string `json:"cwd"`
	HookEventName    string `json:"hook_event_name"`
	Message          string `json:"message"`
	NotificationType string `json:"notification_type"`
	ToolName         string `json:"tool_name"`
}

var ErrNoSession = errors.New("hook input has no session_id")

func Parse(r io.Reader) (Event, error) {
	var ev Event
	data, err := io.ReadAll(io.LimitReader(r, maxInput))
	if err != nil {
		return ev, fmt.Errorf("read hook input: %w", err)
	}
	// Windows PowerShell 5.1 prefixes piped text with a UTF-8 byte order mark.
	data = bytes.TrimPrefix(data, []byte("\xef\xbb\xbf"))
	if err := json.Unmarshal(data, &ev); err != nil {
		return ev, fmt.Errorf("decode hook input: %w", err)
	}
	if ev.SessionID == "" {
		return ev, ErrNoSession
	}
	return ev, nil
}

// Project is the folder name of the session, used to tell sessions apart in
// notifications.
func (e Event) Project() string {
	if e.Cwd == "" {
		return "Claude Code"
	}
	return filepath.Base(filepath.Clean(e.Cwd))
}

// Reason is the human-readable cause of an alert.
func (e Event) Reason() string {
	switch {
	case e.HookEventName == "Stop":
		return "Claude finished"
	case e.ToolName == "AskUserQuestion":
		return "Claude has a question"
	case e.Message != "":
		return e.Message
	default:
		return "Claude needs your input"
	}
}
