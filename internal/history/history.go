package history

import (
	"database/sql"
)

// Entry 历史记录条目
type Entry struct {
	ID             int64  `json:"id"`
	SourceText     string `json:"source_text"`
	TranslatedText string `json:"translated_text"`
	SourceLang     string `json:"source_lang"`
	TargetLang     string `json:"target_lang"`
	IsFavorite     bool   `json:"is_favorite"`
	CreatedAt      string `json:"created_at"`
}

// History 翻译历史管理
type History struct {
	db *sql.DB
}

// New 创建历史管理实例
func New(db *sql.DB) *History {
	return &History{db: db}
}

// Add 添加历史记录
func (h *History) Add(sourceText, translatedText, sourceLang, targetLang string) error {
	_, err := h.db.Exec(
		`INSERT INTO translation_history (source_text, translated_text, source_lang, target_lang)
		 VALUES (?, ?, ?, ?)`,
		sourceText, translatedText, sourceLang, targetLang,
	)
	return err
}

// List 查询历史记录（分页）
func (h *History) List(offset, limit int, keyword string) ([]*Entry, error) {
	query := `SELECT id, source_text, translated_text, source_lang, target_lang, is_favorite, created_at
			  FROM translation_history`
	var args []any

	if keyword != "" {
		query += ` WHERE source_text LIKE ? OR translated_text LIKE ?`
		kw := "%" + keyword + "%"
		args = append(args, kw, kw)
	}
	query += ` ORDER BY created_at DESC LIMIT ? OFFSET ?`
	args = append(args, limit, offset)

	rows, err := h.db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var entries []*Entry
	for rows.Next() {
		e := &Entry{}
		if err := rows.Scan(&e.ID, &e.SourceText, &e.TranslatedText, &e.SourceLang, &e.TargetLang, &e.IsFavorite, &e.CreatedAt); err != nil {
			return nil, err
		}
		entries = append(entries, e)
	}
	return entries, nil
}

// ToggleFavorite 切换收藏状态
func (h *History) ToggleFavorite(id int64) error {
	_, err := h.db.Exec(`UPDATE translation_history SET is_favorite = 1 - is_favorite WHERE id = ?`, id)
	return err
}

// Delete 删除记录
func (h *History) Delete(id int64) error {
	_, err := h.db.Exec(`DELETE FROM translation_history WHERE id = ?`, id)
	return err
}

// Clear 清空历史
func (h *History) Clear() error {
	_, err := h.db.Exec(`DELETE FROM translation_history`)
	return err
}

// Count 记录总数
func (h *History) Count(keyword string) (int64, error) {
	query := `SELECT COUNT(*) FROM translation_history`
	var args []any
	if keyword != "" {
		query += ` WHERE source_text LIKE ? OR translated_text LIKE ?`
		kw := "%" + keyword + "%"
		args = append(args, kw, kw)
	}
	var count int64
	err := h.db.QueryRow(query, args...).Scan(&count)
	return count, err
}
