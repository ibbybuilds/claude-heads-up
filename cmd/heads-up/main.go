// Command heads-up pauses your media and brings Claude Code to the front when
// it needs you, then resumes the media when you reply.
package main

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"

	"github.com/ibbybuilds/claude-heads-up/internal/chime"
	"github.com/ibbybuilds/claude-heads-up/internal/config"
	"github.com/ibbybuilds/claude-heads-up/internal/engine"
	"github.com/ibbybuilds/claude-heads-up/internal/hookinput"
	"github.com/ibbybuilds/claude-heads-up/internal/logging"
	"github.com/ibbybuilds/claude-heads-up/internal/notify/ntfy"
	"github.com/ibbybuilds/claude-heads-up/internal/platform/current"
	"github.com/ibbybuilds/claude-heads-up/internal/state"
)

// Set by the release build: -ldflags "-X main.version=v1.2.3".
var version = "dev"

const usage = `heads-up: get your attention back when Claude Code needs you

Usage:
  heads-up hook <alert|resume>   run from a Claude Code hook (reads stdin)
  heads-up test                  run an alert now, ignoring triggers and focus
  heads-up doctor                check config, platform support and paths
  heads-up config init           write a config file with all defaults
  heads-up config path           print the config file path
  heads-up version
`

func main() {
	if len(os.Args) < 2 {
		fmt.Fprint(os.Stderr, usage)
		os.Exit(2)
	}
	switch os.Args[1] {
	case "hook":
		runHook(os.Args[2:])
	case "test":
		exitOn(runTest())
	case "doctor":
		exitOn(runDoctor(os.Stdout))
	case "config":
		exitOn(runConfig(os.Args[2:], os.Stdout))
	case "version", "--version", "-v":
		fmt.Println(version)
	default:
		fmt.Fprint(os.Stderr, usage)
		os.Exit(2)
	}
}

func exitOn(err error) {
	if err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
}

// runHook must never fail or print: Claude Code shows hook output to the user
// and a non-zero exit can block the turn. Everything goes to the log file.
func runHook(args []string) {
	app, closeLog, err := setup()
	if err != nil {
		return
	}
	defer closeLog()
	defer func() {
		if r := recover(); r != nil {
			app.Log.Error("hook panicked", "panic", fmt.Sprint(r))
		}
	}()

	if len(args) != 1 {
		app.Log.Error("hook needs exactly one action", "args", args)
		return
	}
	ev, err := hookinput.Parse(os.Stdin)
	if err != nil {
		app.Log.Error("bad hook input", "err", err)
		return
	}

	ctx := context.Background()
	switch args[0] {
	case "alert":
		app.Alert(ctx, ev, false)
	case "resume":
		app.Resume(ctx, ev)
	default:
		app.Log.Error("unknown hook action", "action", args[0])
	}
}

func runTest() error {
	app, closeLog, err := setup()
	if err != nil {
		return err
	}
	defer closeLog()
	cwd, _ := os.Getwd()
	app.Alert(context.Background(), hookinput.Event{
		SessionID:     "heads-up-test",
		Cwd:           cwd,
		HookEventName: "Notification",
		Message:       "This is a test alert",
	}, true)
	fmt.Println("Alert sent. Run `heads-up doctor` for the log path if something did not happen.")
	return nil
}

func runDoctor(w io.Writer) error {
	app, closeLog, err := setup()
	if err != nil {
		return err
	}
	defer closeLog()

	cfgPath, _ := config.Path()
	_, warnings := config.Load(cfgPath)
	fmt.Fprintf(w, "version:   %s\n", version)
	fmt.Fprintf(w, "platform:  %s\n", app.Platform.Name())
	fmt.Fprintf(w, "config:    %s%s\n", cfgPath, existsNote(cfgPath))
	fmt.Fprintf(w, "state/log: %s\n", app.Store.Dir())

	if host, err := app.Platform.HostWindow(app.PID); err != nil {
		fmt.Fprintf(w, "host:      not found (%v); focus will be skipped\n", err)
	} else {
		fmt.Fprintf(w, "host:      pid %d\n", host.PID)
	}

	if app.Config.Ntfy.Enabled {
		fmt.Fprintf(w, "ntfy:      %s/%s\n", app.Config.Ntfy.Server, app.Config.Ntfy.Topic)
	} else {
		fmt.Fprintln(w, "ntfy:      off")
	}

	for _, warning := range warnings {
		fmt.Fprintln(w, "warning:  ", warning)
	}
	if len(warnings) > 0 {
		return errors.New("config has warnings")
	}
	fmt.Fprintln(w, "ok")
	return nil
}

func runConfig(args []string, w io.Writer) error {
	path, err := config.Path()
	if err != nil {
		return err
	}
	if len(args) != 1 {
		return errors.New("usage: heads-up config <init|path>")
	}
	switch args[0] {
	case "path":
		fmt.Fprintln(w, path)
		return nil
	case "init":
		if err := config.WriteDefault(path); err != nil {
			if os.IsExist(err) {
				return fmt.Errorf("%s already exists", path)
			}
			return err
		}
		fmt.Fprintln(w, "wrote", path)
		return nil
	default:
		return fmt.Errorf("unknown config command %q", args[0])
	}
}

// setup wires the real dependencies. Set HEADS_UP_DEBUG=1 for verbose logs.
func setup() (*engine.Engine, func(), error) {
	stateDir, err := state.DefaultDir(config.AppName)
	if err != nil {
		return nil, nil, err
	}
	log, closer := logging.Open(stateDir, os.Getenv("HEADS_UP_DEBUG") != "")

	cfgPath, err := config.Path()
	if err != nil {
		closer.Close()
		return nil, nil, err
	}
	cfg, warnings := config.Load(cfgPath)
	for _, warning := range warnings {
		log.Warn("config", "warning", warning)
	}

	app := &engine.Engine{
		Platform: current.New(),
		Store:    state.NewStore(stateDir, time.Now),
		Config:   cfg,
		Log:      log,
		Now:      time.Now,
		PID:      os.Getpid(),
	}
	if cfg.Ntfy.Enabled {
		app.Notifier = &ntfy.Client{
			Server:   cfg.Ntfy.Server,
			Topic:    cfg.Ntfy.Topic,
			Token:    cfg.Ntfy.Token,
			Priority: cfg.Ntfy.Priority,
			HTTP:     &http.Client{},
		}
	}
	if path, err := chime.EnsureFile(stateDir); err != nil {
		log.Warn("write default chime", "err", err)
	} else {
		app.DefaultSound = path
	}
	return app, func() { closer.Close() }, nil
}

func existsNote(path string) string {
	if _, err := os.Stat(path); err != nil {
		return " (not created, using defaults)"
	}
	return ""
}
