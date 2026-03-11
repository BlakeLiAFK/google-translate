//go:build linux

package main

import "os/exec"

// readClipboard 读取系统剪贴板文本（依赖 xclip）
func readClipboard() string {
	out, err := exec.Command("xclip", "-selection", "clipboard", "-o").Output()
	if err != nil {
		// 备选 xsel
		out, err = exec.Command("xsel", "--clipboard", "--output").Output()
		if err != nil {
			return ""
		}
	}
	return string(out)
}
