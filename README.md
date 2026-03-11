# Google Translate Desktop

A cross-platform desktop translation app powered by Google Translate, built with [Wails v3](https://v3.wails.io/) (Go + Vue 3 + TypeScript).

## Features

- Google Translate engine with auto language detection
- Translation history & favorites (SQLite)
- Translation cache for faster repeated lookups
- Mini floating window with clipboard monitoring
- System tray with quick access
- HTTP API server for external integrations
- MCP (Model Context Protocol) server for AI tool calling
- i18n file translation (JSON / YAML / Properties / Android XML / iOS Strings)
- TTS (Text-to-Speech) playback
- Proxy support

## Download

Go to [Releases](../../releases) to download the latest version:

| Platform | Architecture          | File                                  |
| -------- | --------------------- | ------------------------------------- |
| macOS    | Apple Silicon (arm64) | `google-translate-macos-arm64.zip`    |
| Windows  | x64                   | `google-translate.exe`                |
| Linux    | x64                   | `google-translate-linux-amd64.tar.gz` |

## Development

### Prerequisites

- [Go](https://go.dev/) 1.25+
- [Node.js](https://nodejs.org/) 20+
- [Wails CLI v3](https://v3.wails.io/)
- [Task](https://taskfile.dev/)

### Run in dev mode

```bash
wails3 dev
```

### Build for production

```bash
task build
```

### Build for specific platform

```bash
task darwin:build ARCH=arm64   # macOS
task windows:build             # Windows
task linux:build               # Linux
```

## Tech Stack

- **Backend**: Go 1.25, Wails v3, SQLite (modernc.org/sqlite)
- **Frontend**: Vue 3, TypeScript, Vite
- **Translation**: Google Translate (free API)
- **AI Integration**: MCP Server (mcp-go)

## License

MIT
