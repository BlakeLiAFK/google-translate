package main

import (
	"context"
	"embed"
	_ "embed"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"log"
	"log/slog"
	"runtime"
	"strings"
	"time"

	"google-translate/internal/api"
	"google-translate/internal/cache"
	"google-translate/internal/config"
	"google-translate/internal/db"
	"google-translate/internal/engine"
	"google-translate/internal/history"
	"google-translate/internal/i18n"
	mcpserver "google-translate/internal/mcp"
	"google-translate/internal/service"
	"google-translate/internal/tts"

	"github.com/wailsapp/wails/v3/pkg/application"
	"github.com/wailsapp/wails/v3/pkg/events"
)

//go:embed all:frontend/dist
var assets embed.FS

//go:embed assets/tray.png
var trayIconBytes []byte

// AppService 桌面应用绑定服务
type AppService struct {
	svc       *service.TranslateService
	apiServer *api.Server
	mcpServer *mcpserver.Server
	cfg       *config.Config
	hist      *history.History
	cch       *cache.Cache
	miniWin   *application.WebviewWindow
}

// Translate 翻译文本（skipHistory: 自动翻译时跳过历史记录）
func (a *AppService) Translate(text, target, source string, skipHistory bool) (*service.TranslateResult, error) {
	return a.svc.Translate(text, target, source, skipHistory)
}

// GetLanguages 获取支持的语言列表
func (a *AppService) GetLanguages() []service.Language {
	return a.svc.SupportedLanguages()
}

// GetHistory 获取翻译历史
func (a *AppService) GetHistory(offset, limit int, keyword string) ([]*history.Entry, error) {
	return a.hist.List(offset, limit, keyword)
}

// GetHistoryCount 获取历史总数
func (a *AppService) GetHistoryCount(keyword string) (int64, error) {
	return a.hist.Count(keyword)
}

// ToggleFavorite 切换收藏
func (a *AppService) ToggleFavorite(id int64) error {
	return a.hist.ToggleFavorite(id)
}

// DeleteHistory 删除历史记录
func (a *AppService) DeleteHistory(id int64) error {
	return a.hist.Delete(id)
}

// ClearHistory 清空历史
func (a *AppService) ClearHistory() error {
	return a.hist.Clear()
}

// GetSettings 获取所有设置
func (a *AppService) GetSettings() map[string]string {
	return a.cfg.GetAll()
}

// SetSetting 设置配置
func (a *AppService) SetSetting(key, value string) error {
	return a.cfg.Set(key, value)
}

// GetCacheStats 获取缓存统计
func (a *AppService) GetCacheStats() (int64, error) {
	return a.cch.Stats()
}

// ClearCache 清空缓存
func (a *AppService) ClearCache() error {
	return a.cch.Clear()
}

// SetProxy 设置代理并保存配置
func (a *AppService) SetProxy(proxyURL string) error {
	if err := a.svc.Engine.SetProxy(proxyURL); err != nil {
		return err
	}
	return a.cfg.Set("proxy_url", proxyURL)
}

// Correct 通过同语言翻译获取拼写纠正建议
func (a *AppService) Correct(text, lang string) (string, error) {
	if text == "" {
		return "", nil
	}
	if lang == "" || lang == "auto" {
		lang = "en"
	}
	r, err := a.svc.Engine.Translate(context.Background(), text, lang, lang)
	if err != nil {
		return "", err
	}
	return r.Translated, nil
}

// PlayTTS 获取 TTS 语音（返回 base64 编码）
func (a *AppService) PlayTTS(text, lang string) (string, error) {
	if lang == "" || lang == "auto" {
		lang = "en"
	}
	proxy := a.cfg.Get("proxy_url")
	data, err := tts.Fetch(text, lang, proxy)
	if err != nil {
		return "", err
	}
	return base64.StdEncoding.EncodeToString(data), nil
}

// StartHTTPAPI 启动 HTTP API
func (a *AppService) StartHTTPAPI() error {
	port := a.cfg.Get("http_port")
	a.cfg.Set("http_enabled", "true")
	go a.apiServer.Start(port)
	return nil
}

// StopHTTPAPI 停止 HTTP API
func (a *AppService) StopHTTPAPI() error {
	a.cfg.Set("http_enabled", "false")
	return a.apiServer.Stop()
}

// IsHTTPAPIRunning HTTP API 是否运行中
func (a *AppService) IsHTTPAPIRunning() bool {
	return a.apiServer.Running()
}

// StartMCPServer 启动 MCP Server
func (a *AppService) StartMCPServer() error {
	port := a.cfg.Get("mcp_port")
	a.cfg.Set("mcp_enabled", "true")
	go a.mcpServer.Start(port)
	return nil
}

// StopMCPServer 停止 MCP Server
func (a *AppService) StopMCPServer() error {
	a.cfg.Set("mcp_enabled", "false")
	return a.mcpServer.Stop()
}

// IsMCPServerRunning MCP Server 是否运行中
func (a *AppService) IsMCPServerRunning() bool {
	return a.mcpServer.Running()
}

// TranslateI18nContent 翻译 i18n 内容（支持多种格式）
func (a *AppService) TranslateI18nContent(content string, targetLangs []string, sourceLang string, format string) (map[string]string, error) {
	translateFn := func(text, target, source string) (string, error) {
		r, err := a.svc.Translate(text, target, source)
		if err != nil {
			return "", err
		}
		return r.Translated, nil
	}
	if format == "" {
		format = i18n.DetectFormat("", content)
	}
	return i18n.TranslateByFormat(format, content, targetLangs, sourceLang, translateFn)
}

// GetI18nFormats 获取支持的 i18n 格式列表
func (a *AppService) GetI18nFormats() []string {
	return i18n.SupportedFormats()
}


func main() {
	// 单实例检查（进程退出自动释放）
	releaseFunc := acquireLock()
	defer releaseFunc()

	// 初始化数据库
	database, err := db.Open()
	if err != nil {
		log.Fatal("open database:", err)
	}

	// 初始化翻译引擎
	eng, err := engine.New()
	if err != nil {
		log.Fatal("create translator:", err)
	}

	// 初始化各模块
	cch := cache.New(database)
	hist := history.New(database)
	cfg := config.New(database)

	svc := &service.TranslateService{
		Engine:  eng,
		Cache:   cch,
		History: hist,
		Config:  cfg,
	}

	// 应用已保存的代理配置
	if proxy := cfg.Get("proxy_url"); proxy != "" {
		if err := eng.SetProxy(proxy); err != nil {
			slog.Warn("apply saved proxy failed", "error", err)
		}
	}

	apiSrv := api.New(svc)
	mcpSrv := mcpserver.New(svc)

	appSvc := &AppService{
		svc:       svc,
		apiServer: apiSrv,
		mcpServer: mcpSrv,
		cfg:       cfg,
		hist:      hist,
		cch:       cch,
	}

	// 创建 Wails 应用
	app := application.New(application.Options{
		Name:        "Google Translate",
		Description: "Google Translate Desktop App with MCP Server",
		Services: []application.Service{
			application.NewService(appSvc),
		},
		Assets: application.AssetOptions{
			Handler: application.AssetFileServerFS(assets),
		},
		Mac: application.MacOptions{
			ActivationPolicy: application.ActivationPolicyAccessory,
		},
	})

	// 创建主窗口
	mainWindow := app.Window.NewWithOptions(application.WebviewWindowOptions{
		Title:  "Google Translate",
		Width:  900,
		Height: 650,
		Hidden: cfg.Get("start_minimized") == "true",
		Mac: application.MacWindow{
			InvisibleTitleBarHeight: 50,
			Backdrop:                application.MacBackdropTranslucent,
			TitleBar:                application.MacTitleBarHiddenInset,
		},
		BackgroundColour: application.NewRGB(27, 38, 54),
		URL:              "/",
	})

	// 窗口关闭时隐藏而非退出
	mainWindow.RegisterHook(events.Common.WindowClosing, func(e *application.WindowEvent) {
		mainWindow.Hide()
		e.Cancel()
	})

	// 迷你翻译窗口（始终置顶）
	miniWindow := app.Window.NewWithOptions(application.WebviewWindowOptions{
		Title:       "Quick Translate",
		Width:       420,
		Height:      320,
		Hidden:      true,
		AlwaysOnTop: true,
		Mac: application.MacWindow{
			TitleBar: application.MacTitleBarHidden,
			Backdrop: application.MacBackdropTranslucent,
		},
		BackgroundColour: application.NewRGB(27, 38, 54),
		URL:              "/#/mini",
	})
	miniWindow.RegisterHook(events.Common.WindowClosing, func(e *application.WindowEvent) {
		miniWindow.Hide()
		e.Cancel()
	})
	appSvc.miniWin = miniWindow

	// 剪贴板监听
	go func() {
		lastText := ""
		for {
			time.Sleep(800 * time.Millisecond)
			if appSvc.cfg.Get("clipboard_monitor") != "true" {
				continue
			}
			text := strings.TrimSpace(readClipboard())
			if text == "" || text == lastText {
				continue
			}
			lastText = text
			textJSON, _ := json.Marshal(text)
			miniWindow.ExecJS(fmt.Sprintf(`window.dispatchEvent(new CustomEvent('clipboard-change',{detail:%s}))`, textJSON))
		}
	}()

	// 创建系统托盘
	tray := app.SystemTray.New()
	if runtime.GOOS == "darwin" {
		tray.SetTemplateIcon(trayIconBytes)
	} else {
		tray.SetIcon(trayIconBytes)
	}

	// 托盘菜单
	trayMenu := app.Menu.New()
	trayMenu.Add("Google Translate").SetEnabled(false)
	trayMenu.AddSeparator()
	trayMenu.Add("Show Window").OnClick(func(ctx *application.Context) {
		mainWindow.Show()
		mainWindow.Focus()
	})
	trayMenu.Add("Mini Window").OnClick(func(ctx *application.Context) {
		miniWindow.Show()
		miniWindow.Focus()
	})
	trayMenu.AddSeparator()

	clipItem := trayMenu.Add("Clipboard Monitor: Off")
	clipItem.OnClick(func(ctx *application.Context) {
		if cfg.Get("clipboard_monitor") == "true" {
			cfg.Set("clipboard_monitor", "false")
			clipItem.SetLabel("Clipboard Monitor: Off")
		} else {
			cfg.Set("clipboard_monitor", "true")
			clipItem.SetLabel("Clipboard Monitor: On")
		}
	})
	trayMenu.AddSeparator()

	httpItem := trayMenu.Add("HTTP API: Off")
	httpItem.OnClick(func(ctx *application.Context) {
		if appSvc.IsHTTPAPIRunning() {
			appSvc.StopHTTPAPI()
			httpItem.SetLabel("HTTP API: Off")
		} else {
			appSvc.StartHTTPAPI()
			httpItem.SetLabel("HTTP API: :" + cfg.Get("http_port"))
		}
	})

	mcpItem := trayMenu.Add("MCP Server: Off")
	mcpItem.OnClick(func(ctx *application.Context) {
		if appSvc.IsMCPServerRunning() {
			appSvc.StopMCPServer()
			mcpItem.SetLabel("MCP Server: Off")
		} else {
			appSvc.StartMCPServer()
			mcpItem.SetLabel("MCP Server: :" + cfg.Get("mcp_port"))
		}
	})

	trayMenu.AddSeparator()
	trayMenu.Add("Quit").OnClick(func(ctx *application.Context) {
		appSvc.StopHTTPAPI()
		appSvc.StopMCPServer()
		app.Quit()
	})

	tray.SetMenu(trayMenu)

	// macOS: 托盘图标弹出窗口; Windows/Linux: 点击切换窗口显示
	if runtime.GOOS == "darwin" {
		tray.AttachWindow(mainWindow).WindowOffset(5)
	} else {
		tray.OnClick(func() {
			if mainWindow.IsVisible() {
				mainWindow.Hide()
			} else {
				mainWindow.Show()
				mainWindow.Focus()
			}
		})
	}

	// 根据配置自动启动服务
	if cfg.Get("http_enabled") == "true" {
		go func() {
			appSvc.StartHTTPAPI()
			httpItem.SetLabel("HTTP API: :" + cfg.Get("http_port"))
		}()
	}
	if cfg.Get("mcp_enabled") == "true" {
		go func() {
			appSvc.StartMCPServer()
			mcpItem.SetLabel("MCP Server: :" + cfg.Get("mcp_port"))
		}()
	}

	slog.Info("Starting Google Translate Desktop App")
	if err := app.Run(); err != nil {
		log.Fatal(err)
	}
}
