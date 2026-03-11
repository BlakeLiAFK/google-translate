package db

import (
	"database/sql"
	"os"
	"path/filepath"
	"sync"

	_ "modernc.org/sqlite"
)

var (
	once     sync.Once
	instance *sql.DB
)

// 获取数据库存储目录
func dataDir() string {
	home, _ := os.UserHomeDir()
	dir := filepath.Join(home, ".google-translate")
	os.MkdirAll(dir, 0o755)
	return dir
}

// DBPath 返回数据库文件路径
func DBPath() string {
	return filepath.Join(dataDir(), "data.db")
}

// Open 获取全局数据库连接（单例）
func Open() (*sql.DB, error) {
	var err error
	once.Do(func() {
		instance, err = sql.Open("sqlite", DBPath()+"?_journal_mode=WAL&_busy_timeout=5000")
		if err != nil {
			return
		}
		instance.SetMaxOpenConns(1)
		err = migrate(instance)
	})
	return instance, err
}

// migrate 执行数据库迁移
func migrate(db *sql.DB) error {
	ddl := `
	CREATE TABLE IF NOT EXISTS translation_cache (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		hash TEXT UNIQUE NOT NULL,
		source_text TEXT NOT NULL,
		translated_text TEXT NOT NULL,
		source_lang TEXT NOT NULL,
		target_lang TEXT NOT NULL,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP
	);
	CREATE INDEX IF NOT EXISTS idx_cache_hash ON translation_cache(hash);

	CREATE TABLE IF NOT EXISTS translation_history (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		source_text TEXT NOT NULL,
		translated_text TEXT NOT NULL,
		source_lang TEXT NOT NULL,
		target_lang TEXT NOT NULL,
		is_favorite INTEGER DEFAULT 0,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP
	);
	CREATE INDEX IF NOT EXISTS idx_history_created ON translation_history(created_at DESC);

	CREATE TABLE IF NOT EXISTS settings (
		key TEXT PRIMARY KEY,
		value TEXT NOT NULL
	);
	`
	_, err := db.Exec(ddl)
	return err
}
