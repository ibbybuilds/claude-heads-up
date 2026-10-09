//go:build windows

package windows

import (
	"context"
	"fmt"
	"unsafe"

	"golang.org/x/sys/windows"
)

var (
	winmm         = windows.NewLazySystemDLL("winmm.dll")
	procPlaySound = winmm.NewProc("PlaySoundW")
)

const (
	sndSync      = 0x0000
	sndNoDefault = 0x0002
	sndAlias     = 0x00010000
	sndFilename  = 0x00020000

	// The user's own "Notification" sound from Control Panel > Sound.
	defaultAlias = "Notification.Default"
)

// PlaySound blocks until the sound ends (an async sound would be cut off when
// the hook process exits), but gives up waiting when ctx expires.
func (*Backend) PlaySound(ctx context.Context, path string) error {
	name, flags := defaultAlias, uint32(sndAlias)
	if path != "" {
		name, flags = path, sndFilename
	}
	ptr, err := windows.UTF16PtrFromString(name)
	if err != nil {
		return err
	}

	done := make(chan error, 1)
	go func() {
		ok, _, callErr := procPlaySound.Call(uintptr(unsafe.Pointer(ptr)), 0, uintptr(flags|sndSync|sndNoDefault))
		if ok == 0 {
			done <- fmt.Errorf("play %q: %v", name, callErr)
			return
		}
		done <- nil
	}()
	select {
	case err := <-done:
		return err
	case <-ctx.Done():
		return nil
	}
}
