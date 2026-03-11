# GitHub 仓库创建与 CI/CD 发布计划

## 项目信息
- **项目名**: google-translate
- **技术栈**: Wails v3 (Go 1.25 + Vue3 + TypeScript)
- **GitHub 账号**: BlakeLiAFK
- **仓库类型**: 公开（开源）

## 计划步骤

### 1. 初始化 Git 仓库并提交代码
- `git init` + `git add .` + `git commit`

### 2. 用 gh cli 创建 GitHub 开源仓库
- `gh repo create google-translate --public --source=.`

### 3. 创建 GitHub Actions workflow
- 文件: `.github/workflows/release.yml`
- 触发条件: tag push (`v*`) + 手动触发
- 三端构建:
  - **macOS arm64**: `macos-14` (Apple Silicon runner)
  - **Windows amd64**: `windows-latest`, CGO_ENABLED=0
  - **Linux amd64**: `ubuntu-22.04`, CGO_ENABLED=1, 安装 webkit2gtk 依赖
- Go 版本: 1.25.x
- Node 版本: 20
- 产物: `.app.zip` / `.exe` / Linux 二进制
- 自动创建 GitHub Release 并上传产物

### 4. 更新 README.md
- 添加项目描述、功能说明、下载链接

### 5. 推送代码到 GitHub

## 镜像选择说明
| 平台 | Runner | 架构 | CGO | 说明 |
|------|--------|------|-----|------|
| macOS | macos-14 | arm64 | 1 | Apple Silicon 原生 runner |
| Windows | windows-latest | amd64 | 0 | 纯 Go 交叉编译，无需 CGO |
| Linux | ubuntu-22.04 | amd64 | 1 | 需要 webkit2gtk + gcc |

## 状态
- [x] 创建计划文件
- [ ] 初始化 git 仓库并提交代码
- [ ] 用 gh cli 创建 GitHub 开源仓库
- [ ] 创建 GitHub Actions workflow
- [ ] 更新 README.md
- [ ] 推送代码到 GitHub
