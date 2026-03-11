# Google Translate 桌面应用

> 创建: 2026-03-11
> 状态: 已完成

## 目标

基于 Wails v3 + Vue3 + TypeScript + 纯 Go SQLite 构建一个全功能 Google 翻译桌面应用，集成 MCP Server、HTTP API、CLI、i18n 批量翻译，支持系统托盘常驻。

---

## 技术选型

| 组件 | 选择 | 版本/说明 |
|------|------|-----------|
| 桌面框架 | Wails v3 | alpha.55+，原生支持 System Tray |
| 前端 | Vue3 + TypeScript + Vite | SPA |
| 翻译引擎 | dinhcanh303/go_translate | 免费多端点 Google 翻译 |
| MCP SDK | mark3labs/mcp-go | 支持 SSE + Streamable HTTP |
| 数据库 | modernc.org/sqlite | 纯 Go，无 CGO 依赖 |
| 样式 | UnoCSS / TailwindCSS | 轻量 CSS 方案 |

---

## 架构设计

### 分层架构

```
+--------------------------------------------------+
|                  入口层 (cmd/)                     |
|  main.go (桌面+托盘)  |  cli/main.go (CLI 工具)   |
+--------------------------------------------------+
|                  接口层 (服务对外暴露)               |
|  Wails Binding  |  HTTP API  |  MCP Server  |  CLI |
+--------------------------------------------------+
|                  业务层 (internal/)                 |
|  TranslateService  |  CacheService  |  I18nService |
+--------------------------------------------------+
|                  基础层                             |
|  go_translate SDK  |  SQLite DB  |  Config         |
+--------------------------------------------------+
```

### 核心模块

#### 1. 翻译引擎 (internal/engine/)
- 封装 go_translate，统一翻译接口
- 支持多端点自动切换 (TypeRandom / TypeSequential)
- 失败重试 + 端点降级

#### 2. 缓存层 (internal/cache/)
- SQLite 存储翻译缓存
- key = hash(source_lang + target_lang + text)
- 避免重复请求，提升速度

#### 3. 翻译历史 (internal/history/)
- 记录用户翻译历史
- 支持搜索、收藏、导出

#### 4. 配置管理 (internal/config/)
- SQLite 存储设置
- HTTP API 端口、MCP Server 端口
- 翻译偏好（默认目标语言、端点类型等）

---

## 功能模块详细设计

### A. 桌面 UI (Wails v3 + Vue3)

#### 页面
1. **翻译页** (主页)
   - 左右双栏：源文本 / 翻译结果
   - 语言选择下拉框（源语言支持自动检测）
   - 一键交换语言
   - 复制结果按钮

2. **历史页**
   - 翻译历史列表
   - 搜索过滤
   - 收藏标记
   - 批量删除

3. **i18n 工具页**
   - 导入 JSON/YAML 文件
   - 选择目标语言（多选）
   - 进度显示
   - 导出翻译后的文件

4. **设置页**
   - 服务开关（HTTP API / MCP Server）
   - 端口配置
   - 翻译端点选择
   - 缓存管理（清空/统计）
   - 开机启动 / 最小化到托盘

#### 系统托盘
- 左键点击：显示/隐藏主窗口
- 右键菜单：
  - 快速翻译（从剪贴板）
  - 打开主窗口
  - HTTP API: 开启/关闭 (显示端口)
  - MCP Server: 开启/关闭 (显示端口)
  - 设置
  - 退出

### B. HTTP API 服务 (internal/api/)

```
POST /v1/api.json
{
  "action": "translate.text",
  "data": {
    "text": "Hello world",
    "target": "zh",
    "source": "auto"
  }
}

Response:
{
  "code": 0,
  "msg": "success",
  "data": {
    "translated": "你好世界",
    "source_lang": "en",
    "target_lang": "zh"
  }
}
```

支持的 action:
- `translate.text` - 文本翻译
- `translate.batch` - 批量翻译（多段文本）
- `translate.detect` - 语言检测
- `translate.languages` - 支持的语言列表
- `translate.i18n` - i18n JSON 翻译

### C. MCP Server (internal/mcp/)

传输方式: SSE + Streamable HTTP (用户可选)

Tools:
1. **translate** - 文本翻译
   - 参数: text (string), target (string), source (string, 可选)
   - 返回: 翻译结果

2. **detect_language** - 语言检测
   - 参数: text (string)
   - 返回: 语言代码 + 语言名称

3. **translate_batch** - 批量翻译
   - 参数: texts (string[]), target (string), source (string, 可选)
   - 返回: 翻译结果数组

4. **translate_i18n** - i18n 文件翻译
   - 参数: content (string, JSON内容), target_langs (string[])
   - 返回: 各语言翻译后的 JSON

### D. CLI 工具 (cmd/cli/)

```bash
# 基础翻译
$ gt "hello world" -t zh
你好世界

# 管道输入
$ echo "hello" | gt -t zh

# 语言检测
$ gt detect "你好世界"
zh (Chinese)

# i18n 批量翻译
$ gt i18n --src en.json --langs zh,ja,ko --out ./locales/

# 文件翻译
$ gt file README.md -t zh -o README_zh.md
```

---

## 数据库设计 (SQLite)

### translation_cache 表
```sql
CREATE TABLE translation_cache (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    hash TEXT UNIQUE NOT NULL,       -- SHA256(source_lang + target_lang + text)
    source_text TEXT NOT NULL,
    translated_text TEXT NOT NULL,
    source_lang TEXT NOT NULL,
    target_lang TEXT NOT NULL,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP
);
CREATE INDEX idx_cache_hash ON translation_cache(hash);
```

### translation_history 表
```sql
CREATE TABLE translation_history (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    source_text TEXT NOT NULL,
    translated_text TEXT NOT NULL,
    source_lang TEXT NOT NULL,
    target_lang TEXT NOT NULL,
    is_favorite INTEGER DEFAULT 0,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP
);
CREATE INDEX idx_history_created ON translation_history(created_at DESC);
```

### settings 表
```sql
CREATE TABLE settings (
    key TEXT PRIMARY KEY,
    value TEXT NOT NULL
);
```

---

## 项目目录结构

```
google-translate/
├── main.go                     # Wails 桌面应用入口
├── cmd/
│   └── cli/
│       └── main.go             # CLI 工具入口
├── internal/
│   ├── engine/
│   │   └── translator.go       # 翻译引擎封装
│   ├── cache/
│   │   └── cache.go            # SQLite 缓存层
│   ├── history/
│   │   └── history.go          # 翻译历史管理
│   ├── config/
│   │   └── config.go           # 配置管理
│   ├── db/
│   │   └── sqlite.go           # SQLite 初始化和迁移
│   ├── api/
│   │   └── server.go           # HTTP API 服务
│   ├── mcp/
│   │   └── server.go           # MCP Server
│   └── service/
│       └── translate.go        # 翻译业务服务 (统一入口)
├── frontend/
│   ├── index.html
│   ├── package.json
│   ├── tsconfig.json
│   ├── vite.config.ts
│   └── src/
│       ├── App.vue
│       ├── main.ts
│       ├── views/
│       │   ├── Translate.vue   # 翻译主页
│       │   ├── History.vue     # 历史记录
│       │   ├── I18n.vue        # i18n 工具
│       │   └── Settings.vue    # 设置页
│       ├── components/
│       │   └── ...
│       └── stores/
│           └── ...
├── build/
│   └── appicon.png
├── go.mod
├── go.sum
├── Taskfile.yml
├── plans/
│   ├── STATUS.md
│   └── active/
│       └── translate-desktop.md
└── README.md
```

---

## 实施步骤

### Phase 1: 基础框架 (核心)
- [ ] 1.1 初始化 Wails v3 项目 (Go + Vue3 + TS)
- [ ] 1.2 集成 modernc.org/sqlite，建表迁移
- [ ] 1.3 集成 go_translate，实现翻译引擎封装
- [ ] 1.4 实现缓存层
- [ ] 1.5 实现翻译业务服务 (TranslateService)

### Phase 2: 桌面 UI
- [ ] 2.1 翻译主页 UI
- [ ] 2.2 Wails Binding 绑定翻译服务
- [ ] 2.3 历史记录页
- [ ] 2.4 设置页
- [ ] 2.5 系统托盘 + 窗口管理

### Phase 3: 服务集成
- [ ] 3.1 HTTP API 服务
- [ ] 3.2 MCP Server (SSE + Streamable HTTP)
- [ ] 3.3 服务的启停控制 (UI 联动)

### Phase 4: 扩展功能
- [ ] 4.1 CLI 工具
- [ ] 4.2 i18n 批量翻译功能
- [ ] 4.3 i18n 工具页 UI

### Phase 5: 完善
- [ ] 5.1 剪贴板快速翻译
- [ ] 5.2 开机启动
- [ ] 5.3 测试验证
- [ ] 5.4 打包发布

---

## 完成标准

- [ ] 桌面应用可正常启动，翻译功能正常
- [ ] 系统托盘常驻，右键菜单可用
- [ ] HTTP API 可通过 curl 调用
- [ ] MCP Server 可被 Claude/其他 AI 工具连接
- [ ] CLI 工具可在终端使用
- [ ] i18n 批量翻译功能正常
- [ ] 翻译缓存生效，重复翻译秒返回
- [ ] 翻译历史可查看和搜索

## 备注

- Wails v3 目前是 alpha 版本 (alpha.55+)，API 基本稳定
- MCP SSE 已被标记 deprecated，但仍可用；同时支持 Streamable HTTP
- go_translate 支持多端点，TypeRandom 模式可提高可用性
- 纯 Go SQLite (modernc.org/sqlite) 无需 CGO，交叉编译友好
