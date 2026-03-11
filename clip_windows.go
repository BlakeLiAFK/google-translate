//go:build windows

package main

import (
	"os/exec"
	"strings"
)

// readClipboard 读取系统剪贴板文本
func readClipboard() string {
	out, err := exec.Command("powershell", "-command", "Get-Clipboard").Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(out))
}
