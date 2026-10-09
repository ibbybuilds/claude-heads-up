//go:build windows

package windows

import (
	"errors"
	"fmt"
	"strings"
	"sync"
	"syscall"
	"unsafe"

	"github.com/ibbybuilds/claude-heads-up/internal/platform"
	"golang.org/x/sys/windows"
)

var (
	user32                 = windows.NewLazySystemDLL("user32.dll")
	procGetWindow          = user32.NewProc("GetWindow")
	procShowWindow         = user32.NewProc("ShowWindow")
	procGetWindowTextLen   = user32.NewProc("GetWindowTextLengthW")
	procIsIconic           = user32.NewProc("IsIconic")
	procSetForegroundWin   = user32.NewProc("SetForegroundWindow")
	procBringWindowToTop   = user32.NewProc("BringWindowToTop")
	procAttachThreadInput  = user32.NewProc("AttachThreadInput")
	procKeybdEvent         = user32.NewProc("keybd_event")
	kernel32               = windows.NewLazySystemDLL("kernel32.dll")
	procGetConsoleWindow   = kernel32.NewProc("GetConsoleWindow")
	procGetCurrentThreadID = kernel32.NewProc("GetCurrentThreadId")
)

const (
	gwOwner        = 4
	swRestore      = 9
	vkMenu         = 0x12
	keyeventfKeyUp = 0x2
	maxDepth       = 64
)

// Reaching the desktop shell means no real host was found; its windows are
// the taskbar and desktop, which must never be "focused".
var stopAt = map[string]bool{"explorer.exe": true, "services.exe": true, "wininit.exe": true}

var errNoHost = errors.New("no ancestor process owns a visible window")

type process struct {
	parent uint32
	exe    string
}

// HostWindow finds the window the user sees Claude in. Under Windows
// Terminal, VS Code, Cursor or the Claude desktop app that window belongs to
// an ancestor process. Under the legacy console host it belongs to conhost,
// which is not an ancestor, so the attached console window is checked first.
func (*Backend) HostWindow(pid int) (platform.Window, error) {
	if hwnd, _, _ := procGetConsoleWindow.Call(); hwnd != 0 && windows.IsWindowVisible(windows.HWND(hwnd)) {
		var owner uint32
		windows.GetWindowThreadProcessId(windows.HWND(hwnd), &owner)
		return platform.Window{PID: int(owner), Handle: hwnd}, nil
	}

	procs, err := snapshotProcesses()
	if err != nil {
		return platform.Window{}, err
	}
	windowsByPID := topLevelWindows()

	cur := uint32(pid)
	seen := map[uint32]bool{}
	for range maxDepth {
		if seen[cur] {
			break
		}
		seen[cur] = true
		p, ok := procs[cur]
		if !ok || stopAt[strings.ToLower(p.exe)] {
			break
		}
		if hwnd, ok := windowsByPID[cur]; ok {
			return platform.Window{PID: int(cur), Handle: uintptr(hwnd)}, nil
		}
		cur = p.parent
	}
	return platform.Window{}, errNoHost
}

// IsForeground compares processes, not handles, because editors and the
// Claude app often have several top-level windows.
func (*Backend) IsForeground(w platform.Window) (bool, error) {
	fg := windows.GetForegroundWindow()
	if fg == 0 {
		return false, nil
	}
	var pid uint32
	if _, err := windows.GetWindowThreadProcessId(fg, &pid); err != nil {
		return false, err
	}
	return int(pid) == w.PID, nil
}

// Focus works around the foreground lock: Windows ignores
// SetForegroundWindow from a background process unless it shares input state
// with the current foreground thread. The Alt-key fallback is a last resort
// because some apps toggle their menu bar on a lone Alt.
func (b *Backend) Focus(w platform.Window) error {
	hwnd := windows.HWND(w.Handle)
	if iconic, _, _ := procIsIconic.Call(w.Handle); iconic != 0 {
		procShowWindow.Call(w.Handle, swRestore)
	}

	fgThread, _ := windows.GetWindowThreadProcessId(windows.GetForegroundWindow(), nil)
	ownThread, _, _ := procGetCurrentThreadID.Call()
	if fgThread != 0 && uintptr(fgThread) != ownThread {
		procAttachThreadInput.Call(ownThread, uintptr(fgThread), 1)
		defer procAttachThreadInput.Call(ownThread, uintptr(fgThread), 0)
	}
	procBringWindowToTop.Call(w.Handle)
	procSetForegroundWin.Call(w.Handle)
	if windows.GetForegroundWindow() == hwnd {
		return nil
	}

	procKeybdEvent.Call(vkMenu, 0, 0, 0)
	procSetForegroundWin.Call(w.Handle)
	procKeybdEvent.Call(vkMenu, 0, keyeventfKeyUp, 0)
	if windows.GetForegroundWindow() != hwnd {
		return fmt.Errorf("windows refused to focus pid %d", w.PID)
	}
	return nil
}

func snapshotProcesses() (map[uint32]process, error) {
	snap, err := windows.CreateToolhelp32Snapshot(windows.TH32CS_SNAPPROCESS, 0)
	if err != nil {
		return nil, fmt.Errorf("process snapshot: %w", err)
	}
	defer windows.CloseHandle(snap)

	procs := map[uint32]process{}
	var entry windows.ProcessEntry32
	entry.Size = uint32(unsafe.Sizeof(entry))
	for err = windows.Process32First(snap, &entry); err == nil; err = windows.Process32Next(snap, &entry) {
		procs[entry.ProcessID] = process{
			parent: entry.ParentProcessID,
			exe:    windows.UTF16ToString(entry.ExeFile[:]),
		}
	}
	return procs, nil
}

// syscall.NewCallback slots are never freed, so the callback is created once.
var (
	enumOnce     sync.Once
	enumCallback uintptr
	enumResult   map[uint32]windows.HWND
	enumMu       sync.Mutex
)

// topLevelWindows maps each PID to its first visible, unowned, titled
// window: the kind that shows up in Alt+Tab.
func topLevelWindows() map[uint32]windows.HWND {
	enumOnce.Do(func() {
		enumCallback = syscall.NewCallback(func(hwnd windows.HWND, _ uintptr) uintptr {
			if !windows.IsWindowVisible(hwnd) {
				return 1
			}
			if owner, _, _ := procGetWindow.Call(uintptr(hwnd), gwOwner); owner != 0 {
				return 1
			}
			if n, _, _ := procGetWindowTextLen.Call(uintptr(hwnd)); n == 0 {
				return 1
			}
			var pid uint32
			windows.GetWindowThreadProcessId(hwnd, &pid)
			if _, ok := enumResult[pid]; !ok {
				enumResult[pid] = hwnd
			}
			return 1
		})
	})

	enumMu.Lock()
	defer enumMu.Unlock()
	enumResult = map[uint32]windows.HWND{}
	windows.EnumWindows(enumCallback, nil)
	return enumResult
}
