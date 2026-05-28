//go:build windows

package scanner

import (
	"runtime"
	"syscall"
	"unsafe"
)

// GetSystemLocale 은 윈도우즈 syscall 을 통해 시스템 로케일(LocaleName)을 가져와 정규화된 언어 코드를 반환합니다.
func GetSystemLocale() string {
	if runtime.GOOS != "windows" {
		return ""
	}

	kernel32 := syscall.NewLazyDLL("kernel32.dll")
	proc := kernel32.NewProc("GetUserDefaultLocaleName")
	if proc.Find() != nil {
		return ""
	}

	// LOCALE_NAME_MAX_LENGTH 는 85 입니다.
	buf := make([]uint16, 85)
	ret, _, _ := proc.Call(
		uintptr(unsafe.Pointer(&buf[0])),
		uintptr(len(buf)),
	)
	if ret == 0 {
		return ""
	}

	return normalizeLocale(syscall.UTF16ToString(buf))
}
