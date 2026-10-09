//go:build windows

// Package windows is the Windows 10/11 backend. Window and sound work uses
// Win32 directly; media and toasts go through WinRT via Windows PowerShell 5.1,
// which ships with every Windows install and can load WinRT types natively.
package windows

type Backend struct{}

func New() *Backend { return &Backend{} }

func (*Backend) Name() string { return "windows" }
