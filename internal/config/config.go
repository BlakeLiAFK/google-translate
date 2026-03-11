package config

import (
	"database/sql"
)

// 默认配置
const (
	DefaultHTTPPort    = "9700"
	DefaultMCPPort     = "9701"
	DefaultTargetLang  = "zh"
)

// Config 配置管理
type Config struct {
	db *sql.DB
}

// New 创建配置管理实例
func New(db *sql.DB) *Config {
	c := &Config{db: db}
	// 初始化默认值
	c.SetDefault("http_port", DefaultHTTPPort)
	c.SetDefault("mcp_port", DefaultMCPPort)
	c.SetDefault("target_lang", DefaultTargetLang)
	c.SetDefault("http_enabled", "false")
	c.SetDefault("mcp_enabled", "false")
	c.SetDefault("start_minimized", "false")
	c.SetDefault("proxy_url", "")
	c.SetDefault("clipboard_monitor", "false")
	return c
}

// Get 获取配置值
func (c *Config) Get(key string) string {
	var value string
	c.db.QueryRow(`SELECT value FROM settings WHERE key = ?`, key).Scan(&value)
	return value
}

// Set 设置配置值
func (c *Config) Set(key, value string) error {
	_, err := c.db.Exec(
		`INSERT OR REPLACE INTO settings (key, value) VALUES (?, ?)`,
		key, value,
	)
	return err
}

// SetDefault 仅当 key 不存在时设置
func (c *Config) SetDefault(key, value string) {
	c.db.Exec(
		`INSERT OR IGNORE INTO settings (key, value) VALUES (?, ?)`,
		key, value,
	)
}

// GetAll 获取所有配置
func (c *Config) GetAll() map[string]string {
	rows, err := c.db.Query(`SELECT key, value FROM settings`)
	if err != nil {
		return nil
	}
	defer rows.Close()

	m := make(map[string]string)
	for rows.Next() {
		var k, v string
		rows.Scan(&k, &v)
		m[k] = v
	}
	return m
}
