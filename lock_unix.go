//go:build !windows

package main

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"syscall"
)

// acquireLock 通过 flock 文件锁实现单实例
// 返回释放函数，进程退出时调用
func acquireLock() func() {
	home, _ := os.UserHomeDir()
	lockDir := filepath.Join(home, ".google-translate")
	os.MkdirAll(lockDir, 0o755)
	lockPath := filepath.Join(lockDir, "app.lock")

	f, err := os.OpenFile(lockPath, os.O_CREATE|os.O_RDWR, 0o644)
	if err != nil {
		log.Fatal("open lock file:", err)
	}

	// 非阻塞排他锁
	err = syscall.Flock(int(f.Fd()), syscall.LOCK_EX|syscall.LOCK_NB)
	if err != nil {
		fmt.Fprintln(os.Stderr, "Google Translate is already running.")
		f.Close()
		os.Exit(0)
	}

	// 写入 PID
	f.Truncate(0)
	f.Seek(0, 0)
	fmt.Fprintf(f, "%d", os.Getpid())
	f.Sync()

	return func() { f.Close() }
}
