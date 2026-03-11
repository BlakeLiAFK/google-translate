package cache

import (
	"crypto/sha256"
	"database/sql"
	"fmt"
)

// Cache 翻译缓存
type Cache struct {
	db *sql.DB
}

// New 创建缓存实例
func New(db *sql.DB) *Cache {
	return &Cache{db: db}
}

// 生成缓存 key
func hashKey(sourceLang, targetLang, text string) string {
	h := sha256.Sum256([]byte(sourceLang + "|" + targetLang + "|" + text))
	return fmt.Sprintf("%x", h)
}

// CacheEntry 缓存条目
type CacheEntry struct {
	SourceText     string
	TranslatedText string
	SourceLang     string
	TargetLang     string
}

// Get 查询缓存
func (c *Cache) Get(sourceLang, targetLang, text string) (*CacheEntry, bool) {
	key := hashKey(sourceLang, targetLang, text)
	var entry CacheEntry
	err := c.db.QueryRow(
		`SELECT source_text, translated_text, source_lang, target_lang
		 FROM translation_cache WHERE hash = ?`, key,
	).Scan(&entry.SourceText, &entry.TranslatedText, &entry.SourceLang, &entry.TargetLang)
	if err != nil {
		return nil, false
	}
	return &entry, true
}

// Set 写入缓存
func (c *Cache) Set(sourceLang, targetLang, text, translated string) error {
	key := hashKey(sourceLang, targetLang, text)
	_, err := c.db.Exec(
		`INSERT OR REPLACE INTO translation_cache (hash, source_text, translated_text, source_lang, target_lang)
		 VALUES (?, ?, ?, ?, ?)`,
		key, text, translated, sourceLang, targetLang,
	)
	return err
}

// Clear 清空缓存
func (c *Cache) Clear() error {
	_, err := c.db.Exec(`DELETE FROM translation_cache`)
	return err
}

// Stats 缓存统计
func (c *Cache) Stats() (count int64, err error) {
	err = c.db.QueryRow(`SELECT COUNT(*) FROM translation_cache`).Scan(&count)
	return
}
