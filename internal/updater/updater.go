package updater

import (
	"archive/tar"
	"archive/zip"
	"compress/gzip"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"time"
)

const (
	repoOwner = "BlakeLiAFK"
	repoName  = "google-translate"
	apiURL    = "https://api.github.com/repos/" + repoOwner + "/" + repoName + "/releases/latest"
)

// ReleaseAsset GitHub release 附件
type ReleaseAsset struct {
	Name               string `json:"name"`
	BrowserDownloadURL string `json:"browser_download_url"`
	Size               int64  `json:"size"`
}

// ReleaseInfo 最新版本信息
type ReleaseInfo struct {
	TagName      string        `json:"tag_name"`
	Name         string        `json:"name"`
	Body         string        `json:"body"`
	HTMLURL      string        `json:"html_url"`
	HasUpdate    bool          `json:"has_update"`
	CurrentVer   string        `json:"current_ver"`
	DownloadURL  string        `json:"download_url"`
	DownloadSize int64         `json:"download_size"`
	Assets       []ReleaseAsset `json:"assets"`
}

// CheckUpdate 检查是否有新版本
func CheckUpdate(currentVersion, proxyURL string) (*ReleaseInfo, error) {
	client := buildHTTPClient(proxyURL)

	req, err := http.NewRequest("GET", apiURL, nil)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Accept", "application/vnd.github.v3+json")
	req.Header.Set("User-Agent", "google-translate-desktop")

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request github api: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("github api returned %d: %s", resp.StatusCode, string(body))
	}

	var release struct {
		TagName string         `json:"tag_name"`
		Name    string         `json:"name"`
		Body    string         `json:"body"`
		HTMLURL string         `json:"html_url"`
		Assets  []ReleaseAsset `json:"assets"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&release); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}

	info := &ReleaseInfo{
		TagName:    release.TagName,
		Name:       release.Name,
		Body:       release.Body,
		HTMLURL:    release.HTMLURL,
		CurrentVer: currentVersion,
		HasUpdate:  compareVersions(release.TagName, currentVersion) > 0,
		Assets:     release.Assets,
	}

	// 找到当前平台对应的下载链接
	assetName := platformAssetName()
	for _, a := range release.Assets {
		if a.Name == assetName {
			info.DownloadURL = a.BrowserDownloadURL
			info.DownloadSize = a.Size
			break
		}
	}

	return info, nil
}

// DownloadAndReplace 下载新版本并替换当前可执行文件
func DownloadAndReplace(downloadURL, proxyURL string) error {
	if downloadURL == "" {
		return fmt.Errorf("no download url")
	}

	client := &http.Client{Timeout: 10 * time.Minute}
	if proxyURL != "" {
		if u, err := url.Parse(proxyURL); err == nil {
			client.Transport = &http.Transport{Proxy: http.ProxyURL(u)}
		}
	}

	// 下载到临时文件
	resp, err := client.Get(downloadURL)
	if err != nil {
		return fmt.Errorf("download: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("download returned %d", resp.StatusCode)
	}

	tmpFile, err := os.CreateTemp("", "gt-update-*")
	if err != nil {
		return fmt.Errorf("create temp file: %w", err)
	}
	tmpPath := tmpFile.Name()
	defer os.Remove(tmpPath)

	if _, err := io.Copy(tmpFile, resp.Body); err != nil {
		tmpFile.Close()
		return fmt.Errorf("save download: %w", err)
	}
	tmpFile.Close()

	// 根据平台解压并替换
	switch runtime.GOOS {
	case "darwin":
		return replaceDarwin(tmpPath)
	case "windows":
		return replaceWindows(tmpPath)
	default:
		return replaceLinux(tmpPath)
	}
}

// replaceDarwin 替换 macOS .app bundle
func replaceDarwin(zipPath string) error {
	exe, err := os.Executable()
	if err != nil {
		return fmt.Errorf("get executable path: %w", err)
	}
	exe, _ = filepath.EvalSymlinks(exe)

	// 可执行文件在 .app/Contents/MacOS/google-translate
	// 需要定位到 .app 根目录
	appDir := exe
	for i := 0; i < 3; i++ {
		appDir = filepath.Dir(appDir)
	}
	if !strings.HasSuffix(appDir, ".app") {
		// 非 .app 模式运行，直接替换二进制
		return replaceBinaryFromZip(zipPath, exe)
	}

	// 解压到临时目录
	tmpDir, err := os.MkdirTemp("", "gt-update-app-*")
	if err != nil {
		return fmt.Errorf("create temp dir: %w", err)
	}
	defer os.RemoveAll(tmpDir)

	if err := unzip(zipPath, tmpDir); err != nil {
		return fmt.Errorf("unzip: %w", err)
	}

	// 找到解压后的 .app 目录
	newAppDir := filepath.Join(tmpDir, "google-translate.app")
	if _, err := os.Stat(newAppDir); err != nil {
		return fmt.Errorf("extracted .app not found: %w", err)
	}

	// 备份旧的 .app
	backupDir := appDir + ".bak"
	os.RemoveAll(backupDir)
	if err := os.Rename(appDir, backupDir); err != nil {
		return fmt.Errorf("backup old app: %w", err)
	}

	// 移动新的 .app 到原位
	if err := os.Rename(newAppDir, appDir); err != nil {
		// 还原备份
		os.Rename(backupDir, appDir)
		return fmt.Errorf("install new app: %w", err)
	}

	os.RemoveAll(backupDir)
	return nil
}

// replaceWindows 替换 Windows exe
func replaceWindows(zipPath string) error {
	exe, err := os.Executable()
	if err != nil {
		return fmt.Errorf("get executable path: %w", err)
	}
	exe, _ = filepath.EvalSymlinks(exe)

	// 解压到临时目录
	tmpDir, err := os.MkdirTemp("", "gt-update-win-*")
	if err != nil {
		return fmt.Errorf("create temp dir: %w", err)
	}
	defer os.RemoveAll(tmpDir)

	if err := unzip(zipPath, tmpDir); err != nil {
		return fmt.Errorf("unzip: %w", err)
	}

	newExe := filepath.Join(tmpDir, "google-translate.exe")
	if _, err := os.Stat(newExe); err != nil {
		return fmt.Errorf("extracted exe not found: %w", err)
	}

	// Windows 不能覆盖运行中的 exe，使用重命名
	backupExe := exe + ".bak"
	os.Remove(backupExe)
	if err := os.Rename(exe, backupExe); err != nil {
		return fmt.Errorf("backup old exe: %w", err)
	}

	if err := copyFile(newExe, exe); err != nil {
		os.Rename(backupExe, exe)
		return fmt.Errorf("install new exe: %w", err)
	}

	// 备份文件在下次启动时清理
	return nil
}

// replaceLinux 替换 Linux 二进制
func replaceLinux(tarPath string) error {
	exe, err := os.Executable()
	if err != nil {
		return fmt.Errorf("get executable path: %w", err)
	}
	exe, _ = filepath.EvalSymlinks(exe)

	// 解压到临时目录
	tmpDir, err := os.MkdirTemp("", "gt-update-linux-*")
	if err != nil {
		return fmt.Errorf("create temp dir: %w", err)
	}
	defer os.RemoveAll(tmpDir)

	if err := untarGz(tarPath, tmpDir); err != nil {
		return fmt.Errorf("untar: %w", err)
	}

	newExe := filepath.Join(tmpDir, "google-translate")
	if _, err := os.Stat(newExe); err != nil {
		return fmt.Errorf("extracted binary not found: %w", err)
	}

	// Linux 允许替换运行中的文件（通过删除+复制）
	backupExe := exe + ".bak"
	os.Remove(backupExe)
	if err := os.Rename(exe, backupExe); err != nil {
		return fmt.Errorf("backup old binary: %w", err)
	}

	if err := copyFile(newExe, exe); err != nil {
		os.Rename(backupExe, exe)
		return fmt.Errorf("install new binary: %w", err)
	}

	os.Chmod(exe, 0755)
	os.Remove(backupExe)
	return nil
}

// replaceBinaryFromZip 从 zip 中提取二进制直接替换（开发模式 macOS）
func replaceBinaryFromZip(zipPath, exePath string) error {
	tmpDir, err := os.MkdirTemp("", "gt-update-bin-*")
	if err != nil {
		return fmt.Errorf("create temp dir: %w", err)
	}
	defer os.RemoveAll(tmpDir)

	if err := unzip(zipPath, tmpDir); err != nil {
		return fmt.Errorf("unzip: %w", err)
	}

	// 在解压目录中递归查找可执行文件
	newExe := ""
	filepath.Walk(tmpDir, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() {
			return nil
		}
		if info.Name() == "google-translate" {
			newExe = path
			return filepath.SkipAll
		}
		return nil
	})
	if newExe == "" {
		return fmt.Errorf("binary not found in archive")
	}

	backupExe := exePath + ".bak"
	os.Remove(backupExe)
	if err := os.Rename(exePath, backupExe); err != nil {
		return fmt.Errorf("backup: %w", err)
	}

	if err := copyFile(newExe, exePath); err != nil {
		os.Rename(backupExe, exePath)
		return fmt.Errorf("install: %w", err)
	}

	os.Chmod(exePath, 0755)
	os.Remove(backupExe)
	return nil
}

// platformAssetName 返回当前平台对应的 release asset 文件名
func platformAssetName() string {
	switch runtime.GOOS {
	case "darwin":
		return "google-translate-macos-arm64.zip"
	case "windows":
		return "google-translate-windows-amd64.zip"
	default:
		return "google-translate-linux-amd64.tar.gz"
	}
}

// unzip 解压 zip 到目标目录
func unzip(src, dest string) error {
	r, err := zip.OpenReader(src)
	if err != nil {
		return err
	}
	defer r.Close()

	for _, f := range r.File {
		target := filepath.Join(dest, f.Name)
		// 防止 zip slip 攻击
		if !strings.HasPrefix(filepath.Clean(target), filepath.Clean(dest)+string(os.PathSeparator)) {
			continue
		}

		if f.FileInfo().IsDir() {
			os.MkdirAll(target, 0755)
			continue
		}

		os.MkdirAll(filepath.Dir(target), 0755)
		outFile, err := os.OpenFile(target, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, f.Mode())
		if err != nil {
			return err
		}

		rc, err := f.Open()
		if err != nil {
			outFile.Close()
			return err
		}

		_, err = io.Copy(outFile, rc)
		rc.Close()
		outFile.Close()
		if err != nil {
			return err
		}
	}
	return nil
}

// untarGz 解压 tar.gz 到目标目录
func untarGz(src, dest string) error {
	f, err := os.Open(src)
	if err != nil {
		return err
	}
	defer f.Close()

	gz, err := gzip.NewReader(f)
	if err != nil {
		return err
	}
	defer gz.Close()

	tr := tar.NewReader(gz)
	for {
		header, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}

		target := filepath.Join(dest, header.Name)
		if !strings.HasPrefix(filepath.Clean(target), filepath.Clean(dest)+string(os.PathSeparator)) {
			continue
		}

		switch header.Typeflag {
		case tar.TypeDir:
			os.MkdirAll(target, 0755)
		case tar.TypeReg:
			os.MkdirAll(filepath.Dir(target), 0755)
			outFile, err := os.OpenFile(target, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, os.FileMode(header.Mode))
			if err != nil {
				return err
			}
			_, err = io.Copy(outFile, tr)
			outFile.Close()
			if err != nil {
				return err
			}
		}
	}
	return nil
}

// copyFile 复制文件
func copyFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()

	out, err := os.OpenFile(dst, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0755)
	if err != nil {
		return err
	}
	defer out.Close()

	_, err = io.Copy(out, in)
	return err
}

// buildHTTPClient 构建支持代理的 HTTP 客户端
func buildHTTPClient(proxyURL string) *http.Client {
	client := &http.Client{Timeout: 15 * time.Second}
	if proxyURL != "" {
		if u, err := url.Parse(proxyURL); err == nil {
			client.Transport = &http.Transport{Proxy: http.ProxyURL(u)}
		}
	}
	return client
}

// compareVersions 比较两个版本号，返回 1(a>b), -1(a<b), 0(a==b)
func compareVersions(a, b string) int {
	pa := parseVersion(a)
	pb := parseVersion(b)
	for i := range 3 {
		if pa[i] > pb[i] {
			return 1
		}
		if pa[i] < pb[i] {
			return -1
		}
	}
	return 0
}

// parseVersion 解析版本号为 [major, minor, patch]
func parseVersion(v string) [3]int {
	v = strings.TrimPrefix(v, "v")
	parts := strings.SplitN(v, ".", 3)
	var result [3]int
	for i, p := range parts {
		if i >= 3 {
			break
		}
		if idx := strings.IndexAny(p, "-+"); idx >= 0 {
			p = p[:idx]
		}
		n, _ := strconv.Atoi(p)
		result[i] = n
	}
	return result
}
