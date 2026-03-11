package service

import (
	"context"

	"google-translate/internal/cache"
	"google-translate/internal/config"
	"google-translate/internal/engine"
	"google-translate/internal/history"
)

// TranslateService 翻译业务服务（统一入口）
type TranslateService struct {
	Engine  *engine.Translator
	Cache   *cache.Cache
	History *history.History
	Config  *config.Config
}

// TranslateResult 翻译结果
type TranslateResult struct {
	Translated string `json:"translated"`
	SourceLang string `json:"source_lang"`
	TargetLang string `json:"target_lang"`
	Cached     bool   `json:"cached"`
}

// Translate 翻译文本（带缓存和历史记录）
func (s *TranslateService) Translate(text, target, source string, skipHistory ...bool) (*TranslateResult, error) {
	if text == "" {
		return &TranslateResult{}, nil
	}
	if target == "" {
		target = s.Config.Get("target_lang")
	}
	if source == "" {
		source = "auto"
	}

	// 查缓存
	if entry, ok := s.Cache.Get(source, target, text); ok {
		return &TranslateResult{
			Translated: entry.TranslatedText,
			SourceLang: entry.SourceLang,
			TargetLang: entry.TargetLang,
			Cached:     true,
		}, nil
	}

	// 调用翻译引擎
	result, err := s.Engine.Translate(context.Background(), text, target, source)
	if err != nil {
		return nil, err
	}

	// 写缓存
	s.Cache.Set(result.SourceLang, target, text, result.Translated)

	// 写历史（自动翻译时跳过）
	if len(skipHistory) == 0 || !skipHistory[0] {
		s.History.Add(text, result.Translated, result.SourceLang, target)
	}

	return &TranslateResult{
		Translated: result.Translated,
		SourceLang: result.SourceLang,
		TargetLang: target,
		Cached:     false,
	}, nil
}

// TranslateBatch 批量翻译
func (s *TranslateService) TranslateBatch(texts []string, target, source string) ([]*TranslateResult, error) {
	if target == "" {
		target = s.Config.Get("target_lang")
	}
	if source == "" {
		source = "auto"
	}

	// 区分缓存命中和未命中
	var needTranslate []string
	var needTranslateIdx []int
	results := make([]*TranslateResult, len(texts))

	for i, text := range texts {
		if entry, ok := s.Cache.Get(source, target, text); ok {
			results[i] = &TranslateResult{
				Translated: entry.TranslatedText,
				SourceLang: entry.SourceLang,
				TargetLang: entry.TargetLang,
				Cached:     true,
			}
		} else {
			needTranslate = append(needTranslate, text)
			needTranslateIdx = append(needTranslateIdx, i)
		}
	}

	if len(needTranslate) > 0 {
		batchResults, err := s.Engine.TranslateBatch(context.Background(), needTranslate, target, source)
		if err != nil {
			return nil, err
		}
		for j, r := range batchResults {
			idx := needTranslateIdx[j]
			s.Cache.Set(r.SourceLang, target, r.Text, r.Translated)
			results[idx] = &TranslateResult{
				Translated: r.Translated,
				SourceLang: r.SourceLang,
				TargetLang: target,
				Cached:     false,
			}
		}
	}

	return results, nil
}

// SupportedLanguages 支持的语言列表
func (s *TranslateService) SupportedLanguages() []Language {
	return Languages
}
