// Package config loads the user's settings. A missing or broken config file
// never stops the tool: bad values fall back to defaults and produce warnings.
package config

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"slices"
)

const (
	AppName       = "claude-heads-up"
	TokenEnv      = "HEADS_UP_NTFY_TOKEN"
	defaultServer = "https://ntfy.sh"
)

type Config struct {
	Triggers        Triggers `json:"triggers"`
	SkipWhenFocused bool     `json:"skipWhenFocused"`
	ThrottleSeconds int      `json:"throttleSeconds"`
	Media           Toggle   `json:"media"`
	Sound           Sound    `json:"sound"`
	Focus           Toggle   `json:"focus"`
	Toast           Toast    `json:"toast"`
	Ntfy            Ntfy     `json:"ntfy"`
}

type Triggers struct {
	Stop       bool `json:"stop"`
	Permission bool `json:"permission"`
	Question   bool `json:"question"`
}

type Toggle struct {
	Enabled bool `json:"enabled"`
}

type Toast struct {
	Enabled bool `json:"enabled"`
	Sticky  bool `json:"sticky"`
}

type Sound struct {
	Enabled bool   `json:"enabled"`
	File    string `json:"file"`
}

type Ntfy struct {
	Enabled  bool   `json:"enabled"`
	Server   string `json:"server"`
	Topic    string `json:"topic"`
	Token    string `json:"token"`
	Priority string `json:"priority"`
}

// ntfy priority names, lowest to highest.
var ntfyPriorities = []string{"min", "low", "default", "high", "urgent"}

func Default() Config {
	return Config{
		Triggers: Triggers{Stop: true, Permission: true, Question: true},
		// Off by default: a focused window says nothing about where the user is
		// looking when they have a second screen or a phone in hand.
		SkipWhenFocused: false,
		ThrottleSeconds: 2,
		Media:           Toggle{Enabled: true},
		Sound:           Sound{Enabled: true},
		Focus:           Toggle{Enabled: true},
		Toast:           Toast{Enabled: true, Sticky: true},
		Ntfy:            Ntfy{Server: defaultServer, Priority: "urgent"},
	}
}

// Path is where the config file lives: %APPDATA%\claude-heads-up\config.json
// on Windows, ~/Library/Application Support/... on macOS, ~/.config/... on Linux.
func Path() (string, error) {
	dir, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, AppName, "config.json"), nil
}

// Load reads path. It always returns a usable Config; warnings describe
// anything that was ignored or replaced by a default.
func Load(path string) (Config, []string) {
	cfg := Default()
	data, err := os.ReadFile(path)
	if errors.Is(err, fs.ErrNotExist) {
		return finalize(cfg, nil)
	}
	if err != nil {
		return finalize(cfg, []string{fmt.Sprintf("read %s: %v (using defaults)", path, err)})
	}
	return parse(data, path)
}

func parse(data []byte, path string) (Config, []string) {
	cfg := Default()
	if err := json.Unmarshal(data, &cfg); err != nil {
		return finalize(Default(), []string{fmt.Sprintf("%s is not valid JSON: %v (using defaults)", path, err)})
	}

	var warnings []string
	strict := json.NewDecoder(bytes.NewReader(data))
	strict.DisallowUnknownFields()
	if err := strict.Decode(new(Config)); err != nil {
		warnings = append(warnings, fmt.Sprintf("%s: %v (ignored)", path, err))
	}
	return finalize(cfg, warnings)
}

func finalize(cfg Config, warnings []string) (Config, []string) {
	if token := os.Getenv(TokenEnv); token != "" {
		cfg.Ntfy.Token = token
	}
	if cfg.ThrottleSeconds < 0 {
		warnings = append(warnings, "throttleSeconds cannot be negative (using 2)")
		cfg.ThrottleSeconds = Default().ThrottleSeconds
	}
	if cfg.Ntfy.Server == "" {
		cfg.Ntfy.Server = defaultServer
	}
	if !slices.Contains(ntfyPriorities, cfg.Ntfy.Priority) {
		warnings = append(warnings, fmt.Sprintf("ntfy.priority %q is not one of %v (using urgent)", cfg.Ntfy.Priority, ntfyPriorities))
		cfg.Ntfy.Priority = Default().Ntfy.Priority
	}
	if cfg.Ntfy.Enabled && cfg.Ntfy.Topic == "" {
		warnings = append(warnings, "ntfy is enabled but topic is empty (phone push disabled)")
		cfg.Ntfy.Enabled = false
	}
	return cfg, warnings
}

// WriteDefault creates a config file with every option set to its default.
// It refuses to overwrite an existing file.
func WriteDefault(path string) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(Default(), "", "  ")
	if err != nil {
		return err
	}
	f, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o600)
	if err != nil {
		return err
	}
	if _, err := f.Write(append(data, '\n')); err != nil {
		f.Close()
		return err
	}
	return f.Close()
}
