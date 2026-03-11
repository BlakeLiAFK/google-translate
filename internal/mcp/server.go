package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"sync"

	"google-translate/internal/i18n"
	"google-translate/internal/service"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

// Server MCP Server
type Server struct {
	svc       *service.TranslateService
	sseServer *server.SSEServer
	mu        sync.Mutex
}

// New 创建 MCP Server
func New(svc *service.TranslateService) *Server {
	return &Server{svc: svc}
}

// Start 启动 MCP SSE Server
func (s *Server) Start(port string) error {
	s.mu.Lock()
	if s.sseServer != nil {
		s.mu.Unlock()
		return fmt.Errorf("MCP server already running")
	}

	mcpServer := server.NewMCPServer(
		"Google Translate MCP",
		"1.0.0",
		server.WithToolCapabilities(true),
	)

	s.registerTools(mcpServer)

	s.sseServer = server.NewSSEServer(mcpServer,
		server.WithBaseURL(fmt.Sprintf("http://localhost:%s", port)),
	)
	s.mu.Unlock()

	slog.Info("MCP Server started", "port", port)
	err := s.sseServer.Start(":" + port)
	if err != nil {
		s.mu.Lock()
		s.sseServer = nil
		s.mu.Unlock()
	}
	return err
}

// Stop 停止 MCP Server
func (s *Server) Stop() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.sseServer == nil {
		return nil
	}
	err := s.sseServer.Shutdown(context.Background())
	s.sseServer = nil
	slog.Info("MCP Server stopped")
	return err
}

// Running 是否正在运行
func (s *Server) Running() bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.sseServer != nil
}

// registerTools 注册 MCP 工具
func (s *Server) registerTools(mcpServer *server.MCPServer) {
	mcpServer.AddTool(
		mcp.NewTool("translate",
			mcp.WithDescription("Translate text to target language using Google Translate"),
			mcp.WithString("text", mcp.Required(), mcp.Description("Text to translate")),
			mcp.WithString("target", mcp.Required(), mcp.Description("Target language code (e.g. zh, en, ja, ko)")),
			mcp.WithString("source", mcp.Description("Source language code, default auto detect")),
		),
		s.handleTranslate,
	)

	mcpServer.AddTool(
		mcp.NewTool("translate_batch",
			mcp.WithDescription("Translate multiple texts to target language"),
			mcp.WithArray("texts", mcp.Required(), mcp.Description("Array of texts to translate")),
			mcp.WithString("target", mcp.Required(), mcp.Description("Target language code")),
			mcp.WithString("source", mcp.Description("Source language code, default auto detect")),
		),
		s.handleTranslateBatch,
	)

	mcpServer.AddTool(
		mcp.NewTool("translate_i18n",
			mcp.WithDescription("Translate i18n JSON content to multiple languages"),
			mcp.WithString("content", mcp.Required(), mcp.Description("i18n content string (JSON, TypeScript, Markdown, or INI format)")),
			mcp.WithArray("target_langs", mcp.Required(), mcp.Description("Array of target language codes")),
			mcp.WithString("source_lang", mcp.Description("Source language code, default auto detect")),
			mcp.WithString("format", mcp.Description("Content format: json, ts, md, ini. Auto-detected if not specified")),
		),
		s.handleI18n,
	)
}

func (s *Server) handleTranslate(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	args := req.GetArguments()
	text, _ := args["text"].(string)
	target, _ := args["target"].(string)
	source, _ := args["source"].(string)

	result, err := s.svc.Translate(text, target, source)
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	b, _ := json.MarshalIndent(result, "", "  ")
	return mcp.NewToolResultText(string(b)), nil
}

func (s *Server) handleTranslateBatch(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	args := req.GetArguments()
	textsRaw, _ := args["texts"].([]any)
	var texts []string
	for _, t := range textsRaw {
		if str, ok := t.(string); ok {
			texts = append(texts, str)
		}
	}
	target, _ := args["target"].(string)
	source, _ := args["source"].(string)

	results, err := s.svc.TranslateBatch(texts, target, source)
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	b, _ := json.MarshalIndent(results, "", "  ")
	return mcp.NewToolResultText(string(b)), nil
}

func (s *Server) handleI18n(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	args := req.GetArguments()
	content, _ := args["content"].(string)
	langsRaw, _ := args["target_langs"].([]any)
	var langs []string
	for _, l := range langsRaw {
		if str, ok := l.(string); ok {
			langs = append(langs, str)
		}
	}
	sourceLang, _ := args["source_lang"].(string)
	format, _ := args["format"].(string)

	translateFn := func(text, target, source string) (string, error) {
		r, err := s.svc.Translate(text, target, source)
		if err != nil {
			return "", err
		}
		return r.Translated, nil
	}

	if format == "" {
		format = i18n.DetectFormat("", content)
	}
	result, err := i18n.TranslateByFormat(format, content, langs, sourceLang, translateFn)
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	b, _ := json.MarshalIndent(result, "", "  ")
	return mcp.NewToolResultText(string(b)), nil
}
