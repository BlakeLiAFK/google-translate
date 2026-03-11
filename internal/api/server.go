package api

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"sync"

	"google-translate/internal/i18n"
	"google-translate/internal/service"
)

// Server HTTP API 服务
type Server struct {
	svc    *service.TranslateService
	server *http.Server
	mu     sync.Mutex
}

// New 创建 HTTP API 服务
func New(svc *service.TranslateService) *Server {
	return &Server{svc: svc}
}

// 统一请求结构
type apiRequest struct {
	Action string          `json:"action"`
	Data   json.RawMessage `json:"data"`
}

// 统一响应结构
type apiResponse struct {
	Code int    `json:"code"`
	Msg  string `json:"msg"`
	Data any    `json:"data,omitempty"`
}

// Start 启动 HTTP API 服务
func (s *Server) Start(port string) error {
	s.mu.Lock()
	if s.server != nil {
		s.mu.Unlock()
		return fmt.Errorf("server already running")
	}

	mux := http.NewServeMux()
	mux.HandleFunc("POST /v1/api.json", s.handleAPI)
	mux.HandleFunc("GET /v1/health", func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(apiResponse{Code: 0, Msg: "ok"})
	})

	s.server = &http.Server{Addr: ":" + port, Handler: mux}
	s.mu.Unlock()

	slog.Info("HTTP API started", "port", port)
	err := s.server.ListenAndServe()
	if err == http.ErrServerClosed {
		return nil
	}
	return err
}

// Stop 停止 HTTP API 服务
func (s *Server) Stop() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.server == nil {
		return nil
	}
	err := s.server.Shutdown(context.Background())
	s.server = nil
	slog.Info("HTTP API stopped")
	return err
}

// Running 是否正在运行
func (s *Server) Running() bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.server != nil
}

func (s *Server) handleAPI(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	var req apiRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, apiResponse{Code: -1, Msg: "invalid request"})
		return
	}

	switch req.Action {
	case "translate.text":
		s.handleTranslateText(w, req.Data)
	case "translate.batch":
		s.handleTranslateBatch(w, req.Data)
	case "translate.languages":
		s.handleLanguages(w)
	case "translate.i18n":
		s.handleI18n(w, req.Data)
	default:
		writeJSON(w, apiResponse{Code: -1, Msg: "unknown action: " + req.Action})
	}
}

type translateTextReq struct {
	Text   string `json:"text"`
	Target string `json:"target"`
	Source string `json:"source"`
}

func (s *Server) handleTranslateText(w http.ResponseWriter, data json.RawMessage) {
	var req translateTextReq
	if err := json.Unmarshal(data, &req); err != nil {
		writeJSON(w, apiResponse{Code: -1, Msg: "invalid data"})
		return
	}
	result, err := s.svc.Translate(req.Text, req.Target, req.Source)
	if err != nil {
		writeJSON(w, apiResponse{Code: -1, Msg: err.Error()})
		return
	}
	writeJSON(w, apiResponse{Code: 0, Msg: "success", Data: result})
}

type translateBatchReq struct {
	Texts  []string `json:"texts"`
	Target string   `json:"target"`
	Source string   `json:"source"`
}

func (s *Server) handleTranslateBatch(w http.ResponseWriter, data json.RawMessage) {
	var req translateBatchReq
	if err := json.Unmarshal(data, &req); err != nil {
		writeJSON(w, apiResponse{Code: -1, Msg: "invalid data"})
		return
	}
	results, err := s.svc.TranslateBatch(req.Texts, req.Target, req.Source)
	if err != nil {
		writeJSON(w, apiResponse{Code: -1, Msg: err.Error()})
		return
	}
	writeJSON(w, apiResponse{Code: 0, Msg: "success", Data: results})
}

func (s *Server) handleLanguages(w http.ResponseWriter) {
	writeJSON(w, apiResponse{Code: 0, Msg: "success", Data: s.svc.SupportedLanguages()})
}

type i18nReq struct {
	Content     string   `json:"content"`
	TargetLangs []string `json:"target_langs"`
	SourceLang  string   `json:"source_lang"`
	Format      string   `json:"format"`
}

func (s *Server) handleI18n(w http.ResponseWriter, data json.RawMessage) {
	var req i18nReq
	if err := json.Unmarshal(data, &req); err != nil {
		writeJSON(w, apiResponse{Code: -1, Msg: "invalid data"})
		return
	}

	translateFn := func(text, target, source string) (string, error) {
		r, err := s.svc.Translate(text, target, source)
		if err != nil {
			return "", err
		}
		return r.Translated, nil
	}

	format := req.Format
	if format == "" {
		format = i18n.DetectFormat("", req.Content)
	}
	result, err := i18n.TranslateByFormat(format, req.Content, req.TargetLangs, req.SourceLang, translateFn)
	if err != nil {
		writeJSON(w, apiResponse{Code: -1, Msg: err.Error()})
		return
	}
	writeJSON(w, apiResponse{Code: 0, Msg: "success", Data: result})
}

func writeJSON(w http.ResponseWriter, v any) {
	json.NewEncoder(w).Encode(v)
}
