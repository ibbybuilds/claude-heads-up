package config

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func TestLoadMissingFileUsesDefaults(t *testing.T) {
	cfg, warnings := Load(filepath.Join(t.TempDir(), "nope.json"))
	if len(warnings) != 0 {
		t.Errorf("warnings = %v", warnings)
	}
	if !reflect.DeepEqual(cfg, Default()) {
		t.Errorf("cfg = %+v", cfg)
	}
}

func TestParsePartialKeepsDefaults(t *testing.T) {
	cfg, warnings := parse([]byte(`{"sound": {"enabled": false}}`), "c.json")
	if len(warnings) != 0 {
		t.Errorf("warnings = %v", warnings)
	}
	if cfg.Sound.Enabled {
		t.Error("sound should be disabled")
	}
	if !cfg.Media.Enabled || !cfg.Triggers.Stop {
		t.Error("unset fields should keep defaults")
	}
}

func TestParseWarnings(t *testing.T) {
	tests := []struct {
		name, json, wantWarning string
	}{
		{"invalid json", `{`, "not valid JSON"},
		{"unknown field", `{"volume": 3}`, "unknown field"},
		{"negative throttle", `{"throttleSeconds": -1}`, "negative"},
		{"ntfy without topic", `{"ntfy": {"enabled": true}}`, "topic is empty"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg, warnings := parse([]byte(tt.json), "c.json")
			if len(warnings) != 1 || !strings.Contains(warnings[0], tt.wantWarning) {
				t.Fatalf("warnings = %v, want one containing %q", warnings, tt.wantWarning)
			}
			if cfg.ThrottleSeconds < 0 || cfg.Ntfy.Enabled && cfg.Ntfy.Topic == "" {
				t.Errorf("invalid value survived: %+v", cfg)
			}
		})
	}
}

func TestTokenFromEnv(t *testing.T) {
	t.Setenv(TokenEnv, "secret")
	cfg, _ := parse([]byte(`{"ntfy": {"token": "from-file"}}`), "c.json")
	if cfg.Ntfy.Token != "secret" {
		t.Errorf("token = %q, env should win", cfg.Ntfy.Token)
	}
}

func TestWriteDefaultRoundTrip(t *testing.T) {
	path := filepath.Join(t.TempDir(), "sub", "config.json")
	if err := WriteDefault(path); err != nil {
		t.Fatal(err)
	}
	if err := WriteDefault(path); !os.IsExist(err) {
		t.Errorf("second write: got %v, want exists error", err)
	}
	cfg, warnings := Load(path)
	if len(warnings) != 0 || !reflect.DeepEqual(cfg, Default()) {
		t.Errorf("round trip: cfg=%+v warnings=%v", cfg, warnings)
	}
}
