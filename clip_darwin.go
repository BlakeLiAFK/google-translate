//go:build darwin

package main

import "os/exec"

// readClipboard 读取系统剪贴板文本
func readClipboard() string {
	out, err := exec.Command("pbpaste").Output()
	if err != nil {
		return ""
	}
	return string(out)
}
