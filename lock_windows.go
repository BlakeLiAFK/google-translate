//go:build windows

package main

import (
	"fmt"
	"os"
	"syscall"
	"unsafe"
)

var (
	kernel32        = syscall.NewLazyDLL("kernel32.dll")
	procCreateMutex = kernel32.NewProc("CreateMutexW")
	procGetLastErr  = kernel32.NewProc("GetLastError")
)

const errorAlreadyExists = 183

// acquireLock 通过 Windows 命名互斥体实现单实例
// 返回释放函数，进程退出时调用
func acquireLock() func() {
	name, _ := syscall.UTF16PtrFromString("Global\\GoogleTranslateDesktopApp")

	handle, _, _ := procCreateMutex.Call(
		0,
		0,
		uintptr(unsafe.Pointer(name)),
	)

	if handle == 0 {
		fmt.Fprintln(os.Stderr, "Google Translate is already running.")
		os.Exit(0)
	}

	// 检查是否已存在
	lastErr, _, _ := procGetLastErr.Call()
	if lastErr == errorAlreadyExists {
		fmt.Fprintln(os.Stderr, "Google Translate is already running.")
		syscall.CloseHandle(syscall.Handle(handle))
		os.Exit(0)
	}

	return func() {
		syscall.CloseHandle(syscall.Handle(handle))
	}
}
