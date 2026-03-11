package engine

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"sync"
	"time"

	gt "github.com/dinhcanh303/go_translate"
)

// Translator 翻译引擎封装
type Translator struct {
	mu         sync.RWMutex
	translator gt.Translator
}

// TranslateResult 翻译结果
type TranslateResult struct {
	Text       string `json:"text"`
	Translated string `json:"translated"`
	SourceLang string `json:"source_lang"`
	TargetLang string `json:"target_lang"`
}

// New 创建翻译引擎
func New() (*Translator, error) {
	t, err := gt.NewTranslator(&gt.TranslateOptions{
		Provider:            gt.ProviderGoogle,
		GoogleAPIType:       gt.TypeRandom,
		UseRandomUserAgents: true,
	})
	if err != nil {
		return nil, fmt.Errorf("create translator: %w", err)
	}
	return &Translator{translator: t}, nil
}

// SetProxy 设置代理并重建翻译引擎
// proxyURL 为空则取消代理，支持 http:// socks5:// 格式
func (t *Translator) SetProxy(proxyURL string) error {
	opts := &gt.TranslateOptions{
		Provider:            gt.ProviderGoogle,
		GoogleAPIType:       gt.TypeRandom,
		UseRandomUserAgents: true,
	}

	if proxyURL != "" {
		u, err := url.Parse(proxyURL)
		if err != nil {
			return fmt.Errorf("invalid proxy url: %w", err)
		}
		opts.HTTPClient = &http.Client{
			Timeout:   15 * time.Second,
			Transport: &http.Transport{Proxy: http.ProxyURL(u)},
		}
	}

	newT, err := gt.NewTranslator(opts)
	if err != nil {
		return fmt.Errorf("recreate translator with proxy: %w", err)
	}

	t.mu.Lock()
	t.translator = newT
	t.mu.Unlock()
	return nil
}

// Translate 翻译文本
func (t *Translator) Translate(ctx context.Context, text, target, source string) (*TranslateResult, error) {
	if text == "" {
		return &TranslateResult{TargetLang: target}, nil
	}

	t.mu.RLock()
	defer t.mu.RUnlock()

	var args []string
	if source != "" && source != "auto" {
		args = append(args, source)
	}

	results, err := t.translator.TranslateText(ctx, []string{text}, target, args...)
	if err != nil {
		return nil, fmt.Errorf("translation failed: %w", err)
	}
	if len(results) == 0 {
		return nil, fmt.Errorf("empty translation result")
	}

	srcLang := source
	if srcLang == "" || srcLang == "auto" {
		srcLang = "auto"
	}

	return &TranslateResult{
		Text:       text,
		Translated: results[0],
		SourceLang: srcLang,
		TargetLang: target,
	}, nil
}

// TranslateBatch 批量翻译
func (t *Translator) TranslateBatch(ctx context.Context, texts []string, target, source string) ([]*TranslateResult, error) {
	if len(texts) == 0 {
		return nil, nil
	}

	t.mu.RLock()
	defer t.mu.RUnlock()

	var args []string
	if source != "" && source != "auto" {
		args = append(args, source)
	}

	results, err := t.translator.TranslateText(ctx, texts, target, args...)
	if err != nil {
		return nil, fmt.Errorf("batch translation failed: %w", err)
	}

	srcLang := source
	if srcLang == "" || srcLang == "auto" {
		srcLang = "auto"
	}

	var out []*TranslateResult
	for i, r := range results {
		out = append(out, &TranslateResult{
			Text:       texts[i],
			Translated: r,
			SourceLang: srcLang,
			TargetLang: target,
		})
	}
	return out, nil
}
