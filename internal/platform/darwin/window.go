package darwin

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/ibbybuilds/claude-heads-up/internal/platform"
)

const (
	maxDepth      = 64
	windowTimeout = 3 * time.Second
)

var errNoHost = errors.New("no ancestor process is a GUI app")

// System Events asks for Automation permission the first time, for whichever
// app runs Claude Code (Terminal, iTerm, VS Code, Claude).
const guiPIDsScript = `function run() {
	const se = Application('System Events');
	return JSON.stringify(se.processes.whose({backgroundOnly: false}).unixId());
}`

const frontPIDScript = `function run() {
	const se = Application('System Events');
	const front = se.processes.whose({frontmost: true});
	return front.length ? String(front[0].unixId()) : '0';
}`

const focusScript = `function run(argv) {
	const se = Application('System Events');
	const procs = se.processes.whose({unixId: Number(argv[0])});
	if (!procs.length) throw new Error('process ' + argv[0] + ' not found');
	procs[0].frontmost = true;
}`

// HostWindow finds the GUI app the user sees Claude in. Claude Code runs
// hooks through a shell, so that app is several levels up the process tree.
func (b *Backend) HostWindow(pid int) (platform.Window, error) {
	ctx, cancel := context.WithTimeout(context.Background(), windowTimeout)
	defer cancel()

	psOut, err := b.run(ctx, "ps", "-axo", "pid=,ppid=")
	if err != nil {
		return platform.Window{}, err
	}
	parents := parseParents(string(psOut))

	guiOut, err := b.jxa(ctx, guiPIDsScript)
	if err != nil {
		return platform.Window{}, err
	}
	var guiPIDs []int
	if err := json.Unmarshal([]byte(guiOut), &guiPIDs); err != nil {
		return platform.Window{}, fmt.Errorf("parse GUI process list %q: %w", guiOut, err)
	}
	gui := map[int]bool{}
	for _, p := range guiPIDs {
		gui[p] = true
	}

	cur := pid
	seen := map[int]bool{}
	for range maxDepth {
		if cur <= 1 || seen[cur] {
			break
		}
		seen[cur] = true
		if gui[cur] {
			return platform.Window{PID: cur}, nil
		}
		cur = parents[cur]
	}
	return platform.Window{}, errNoHost
}

func (b *Backend) IsForeground(w platform.Window) (bool, error) {
	ctx, cancel := context.WithTimeout(context.Background(), windowTimeout)
	defer cancel()
	out, err := b.jxa(ctx, frontPIDScript)
	if err != nil {
		return false, err
	}
	front, err := strconv.Atoi(out)
	if err != nil {
		return false, fmt.Errorf("parse frontmost pid %q: %w", out, err)
	}
	return front == w.PID, nil
}

func (b *Backend) Focus(w platform.Window) error {
	ctx, cancel := context.WithTimeout(context.Background(), windowTimeout)
	defer cancel()
	_, err := b.jxa(ctx, focusScript, strconv.Itoa(w.PID))
	return err
}

// parseParents reads `ps -axo pid=,ppid=` output into pid -> parent pid.
func parseParents(out string) map[int]int {
	parents := map[int]int{}
	for _, line := range strings.Split(out, "\n") {
		fields := strings.Fields(line)
		if len(fields) != 2 {
			continue
		}
		pid, err1 := strconv.Atoi(fields[0])
		ppid, err2 := strconv.Atoi(fields[1])
		if err1 == nil && err2 == nil {
			parents[pid] = ppid
		}
	}
	return parents
}
