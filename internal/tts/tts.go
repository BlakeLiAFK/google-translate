package tts

import (
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"
)

// Fetch 从 Google TTS 获取语音数据
func Fetch(text, lang, proxyURL string) ([]byte, error) {
	if text == "" || lang == "" {
		return nil, fmt.Errorf("text and lang required")
	}
	// Google TTS 单次限制约 200 字符
	runes := []rune(text)
	if len(runes) > 200 {
		text = string(runes[:200])
	}

	ttsURL := fmt.Sprintf(
		"https://translate.google.com/translate_tts?ie=UTF-8&tl=%s&client=tw-ob&q=%s&total=1&idx=0&textlen=%d",
		url.QueryEscape(lang),
		url.QueryEscape(text),
		len([]byte(text)),
	)

	client := &http.Client{Timeout: 10 * time.Second}
	if proxyURL != "" {
		if u, err := url.Parse(proxyURL); err == nil {
			client.Transport = &http.Transport{Proxy: http.ProxyURL(u)}
		}
	}

	req, err := http.NewRequest("GET", ttsURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36")
	req.Header.Set("Referer", "https://translate.google.com/")

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("tts request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("tts status %d", resp.StatusCode)
	}

	return io.ReadAll(resp.Body)
}
