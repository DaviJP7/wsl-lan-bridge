//go:build windows

package netwin

import (
	"fmt"
	"os"
	"strings"
	"syscall"
	"unsafe"

	"golang.org/x/sys/windows"
)

func IsAdmin() bool {
	var sid *windows.SID
	err := windows.AllocateAndInitializeSid(
		&windows.SECURITY_NT_AUTHORITY,
		2,
		windows.SECURITY_BUILTIN_DOMAIN_RID,
		windows.DOMAIN_ALIAS_RID_ADMINS,
		0, 0, 0, 0, 0, 0,
		&sid,
	)
	if err != nil {
		return false
	}
	defer windows.FreeSid(sid)
	token := windows.Token(0)
	member, err := token.IsMember(sid)
	return err == nil && member
}

// RelaunchElevated restarts the current process with UAC elevation and waits.
func RelaunchElevated(extraArgs ...string) (int, error) {
	exe, err := os.Executable()
	if err != nil {
		return 1, err
	}
	args := append([]string{}, os.Args[1:]...)
	args = append(args, extraArgs...)
	// Avoid duplicating the elevated marker.
	filtered := make([]string, 0, len(args))
	for _, a := range args {
		if a == "--elevated" {
			continue
		}
		filtered = append(filtered, a)
	}
	filtered = append(filtered, "--elevated")
	param := strings.Join(quoteArgs(filtered), " ")

	verb, _ := syscall.UTF16PtrFromString("runas")
	file, _ := syscall.UTF16PtrFromString(exe)
	params, _ := syscall.UTF16PtrFromString(param)
	dir, _ := syscall.UTF16PtrFromString("")
	var show int32 = 1

	err = shellExecuteEx(verb, file, params, dir, show)
	if err != nil {
		return 1, fmt.Errorf("não foi possível elevar privilégios (UAC cancelado?): %w", err)
	}
	return 0, nil
}

func quoteArgs(args []string) []string {
	out := make([]string, len(args))
	for i, a := range args {
		if strings.ContainsAny(a, " \t\"") {
			out[i] = `"` + strings.ReplaceAll(a, `"`, `\"`) + `"`
		} else {
			out[i] = a
		}
	}
	return out
}

type shellExecuteInfo struct {
	cbSize       uint32
	fMask        uint32
	hwnd         uintptr
	lpVerb       *uint16
	lpFile       *uint16
	lpParameters *uint16
	lpDirectory  *uint16
	nShow        int32
	hInstApp     uintptr
	lpIDList     uintptr
	lpClass      *uint16
	hKeyClass    uintptr
	dwHotKey     uint32
	hIcon        uintptr
	hProcess     windows.Handle
}

const (
	seeMaskNoCloseProcess = 0x00000040
	seeMaskFlagNoUI       = 0x00000400
)

func shellExecuteEx(verb, file, params, dir *uint16, show int32) error {
	var info shellExecuteInfo
	info.cbSize = uint32(unsafe.Sizeof(info))
	info.fMask = seeMaskNoCloseProcess
	info.lpVerb = verb
	info.lpFile = file
	info.lpParameters = params
	info.lpDirectory = dir
	info.nShow = show

	shell32 := windows.NewLazySystemDLL("shell32.dll")
	proc := shell32.NewProc("ShellExecuteExW")
	r, _, err := proc.Call(uintptr(unsafe.Pointer(&info)))
	if r == 0 {
		if err != nil {
			return err
		}
		return fmt.Errorf("ShellExecuteEx falhou")
	}
	if info.hProcess != 0 {
		defer windows.CloseHandle(info.hProcess)
		_, waitErr := windows.WaitForSingleObject(info.hProcess, windows.INFINITE)
		if waitErr != nil {
			return waitErr
		}
		var code uint32
		if gerr := windows.GetExitCodeProcess(info.hProcess, &code); gerr != nil {
			return gerr
		}
		if code != 0 {
			return fmt.Errorf("processo elevado saiu com código %d", code)
		}
	}
	return nil
}
