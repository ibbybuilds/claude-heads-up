# Contributing

Thanks for helping. The most useful contribution right now is a macOS or Linux backend. [docs/adding-a-platform.md](docs/adding-a-platform.md) explains how.

## Layout

```
cmd/heads-up/          CLI entry point
internal/engine/       all alert and resume rules, OS-independent
internal/platform/     the Platform interface and one package per OS
internal/config/       config file loading and validation
internal/state/        what survives between hook calls (paused media, throttle)
internal/hookinput/    parsing Claude Code's hook JSON
internal/notify/ntfy/  phone push
hooks/hooks.json       which Claude Code events call the binary
bin/heads-up           launcher that picks the right binary from dist/
```

The rule that keeps this maintainable is that `engine` only talks to interfaces. If you need OS behavior in the engine, add a method to `platform.Platform` instead.

## Development

You need Go 1.24 or newer.

```bash
go test ./...
go vet ./...
GOOS=darwin go vet ./...   # the stubs must compile everywhere
GOOS=linux go vet ./...
```

## Testing the plugin locally

Build a binary for your machine into `dist/`:

```bash
sh scripts/build-dist.sh --local
```

Start Claude Code with the repo loaded as a plugin:

```bash
claude --plugin-dir /path/to/claude-heads-up
```

Then go through [docs/manual-test.md](docs/manual-test.md) for your platform.

## Hook rules

A hook must never print to stdout or stderr and must always exit 0. Claude Code shows hook output to the user, and a broken alert tool should never get in the way of Claude. Log through `slog` instead.

## Releases

Maintainers tag `vX.Y.Z` on `main`. The release workflow builds every binary, publishes a GitHub release and updates the `release` branch, which is what the marketplace installs from. `main` never contains binaries.

## Commits

Conventional commits: `feat:`, `fix:`, `docs:`, `refactor:`, `test:`, `chore:`.
