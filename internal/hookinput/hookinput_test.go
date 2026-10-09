package hookinput

import (
	"errors"
	"strings"
	"testing"
)

func TestParse(t *testing.T) {
	ev, err := Parse(strings.NewReader(`{
		"session_id": "abc",
		"cwd": "C:\\code\\my-app",
		"hook_event_name": "Notification",
		"message": "Claude needs your permission to use Bash",
		"notification_type": "permission_prompt",
		"extra_field": 1
	}`))
	if err != nil {
		t.Fatal(err)
	}
	if ev.SessionID != "abc" || ev.NotificationType != "permission_prompt" {
		t.Fatalf("unexpected event: %+v", ev)
	}
	if got := ev.Reason(); got != "Claude needs your permission to use Bash" {
		t.Errorf("Reason() = %q", got)
	}
}

func TestParseErrors(t *testing.T) {
	if _, err := Parse(strings.NewReader(`{"cwd":"/x"}`)); !errors.Is(err, ErrNoSession) {
		t.Errorf("missing session: got %v", err)
	}
	if _, err := Parse(strings.NewReader(`not json`)); err == nil {
		t.Error("invalid JSON: want error")
	}
}

func TestReason(t *testing.T) {
	tests := []struct {
		ev   Event
		want string
	}{
		{Event{HookEventName: "Stop"}, "Claude finished"},
		{Event{HookEventName: "PreToolUse", ToolName: "AskUserQuestion"}, "Claude has a question"},
		{Event{HookEventName: "Notification"}, "Claude needs your input"},
	}
	for _, tt := range tests {
		if got := tt.ev.Reason(); got != tt.want {
			t.Errorf("Reason(%+v) = %q, want %q", tt.ev, got, tt.want)
		}
	}
}

func TestProject(t *testing.T) {
	if got := (Event{Cwd: "/home/me/my-app/"}).Project(); got != "my-app" {
		t.Errorf("Project() = %q", got)
	}
	if got := (Event{}).Project(); got != "Claude Code" {
		t.Errorf("empty cwd Project() = %q", got)
	}
}
