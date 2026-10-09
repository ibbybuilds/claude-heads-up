//go:build windows

package windows

import (
	"bytes"
	"context"
	"encoding/base64"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"
	"unicode/utf16"

	"golang.org/x/sys/windows"
)

// runPowerShell runs script in Windows PowerShell 5.1 (not pwsh 7, which
// cannot load WinRT types). The script travels as -EncodedCommand so no file
// is written and execution policy does not apply. Inputs go through env
// vars, never string interpolation, so titles cannot inject code.
func runPowerShell(ctx context.Context, script string, env ...string) ([]byte, error) {
	exe := filepath.Join(os.Getenv("SystemRoot"), "System32", "WindowsPowerShell", "v1.0", "powershell.exe")
	cmd := exec.CommandContext(ctx, exe,
		"-NoProfile", "-NonInteractive", "-ExecutionPolicy", "Bypass",
		"-EncodedCommand", encodeCommand(script))
	cmd.Env = append(os.Environ(), env...)
	cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true, CreationFlags: windows.CREATE_NO_WINDOW}

	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	out, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("powershell: %w: %s", err, strings.TrimSpace(stderr.String()))
	}
	return out, nil
}

// encodeCommand produces what -EncodedCommand expects: base64 of UTF-16LE.
func encodeCommand(script string) string {
	units := utf16.Encode([]rune(script))
	buf := make([]byte, len(units)*2)
	for i, u := range units {
		buf[2*i] = byte(u)
		buf[2*i+1] = byte(u >> 8)
	}
	return base64.StdEncoding.EncodeToString(buf)
}
