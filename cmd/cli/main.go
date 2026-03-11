package main

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"google-translate/internal/cache"
	"google-translate/internal/config"
	"google-translate/internal/db"
	"google-translate/internal/engine"
	"google-translate/internal/history"
	"google-translate/internal/i18n"
	"google-translate/internal/service"
)

func main() {
	if len(os.Args) < 2 {
		printUsage()
		os.Exit(1)
	}

	cmd := os.Args[1]
	switch cmd {
	case "help", "-h", "--help":
		printUsage()
	case "i18n":
		runI18n(os.Args[2:])
	default:
		runTranslate(os.Args[1:])
	}
}

func printUsage() {
	fmt.Println(`gt - Google Translate CLI

Usage:
  gt <text> [-t target] [-s source]    Translate text
  gt i18n -f <file> -l <langs>         Translate i18n JSON file
  echo "text" | gt [-t target]         Translate from stdin

Options:
  -t    Target language (default: zh)
  -s    Source language (default: auto)
  -f    Source i18n JSON file
  -l    Target languages, comma separated (e.g. zh,ja,ko)
  -o    Output directory (default: current dir)

Examples:
  gt "hello world" -t zh
  gt "hello" -t ja -s en
  echo "hello" | gt -t zh
  gt i18n -f en.json -l zh,ja,ko -o ./locales/`)
}

func initService() *service.TranslateService {
	database, err := db.Open()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: open database: %v\n", err)
		os.Exit(1)
	}
	eng, err := engine.New()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: create translator: %v\n", err)
		os.Exit(1)
	}
	return &service.TranslateService{
		Engine:  eng,
		Cache:   cache.New(database),
		History: history.New(database),
		Config:  config.New(database),
	}
}

func runTranslate(args []string) {
	target := "zh"
	source := "auto"
	var text string

	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "-t":
			if i+1 < len(args) {
				target = args[i+1]
				i++
			}
		case "-s":
			if i+1 < len(args) {
				source = args[i+1]
				i++
			}
		default:
			if text == "" {
				text = args[i]
			}
		}
	}

	// 从 stdin 读取
	if text == "" {
		stat, _ := os.Stdin.Stat()
		if (stat.Mode() & os.ModeCharDevice) == 0 {
			b, _ := io.ReadAll(os.Stdin)
			text = strings.TrimSpace(string(b))
		}
	}

	if text == "" {
		fmt.Fprintln(os.Stderr, "Error: no text to translate")
		os.Exit(1)
	}

	svc := initService()
	result, err := svc.Translate(text, target, source)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
	fmt.Println(result.Translated)
}

func runI18n(args []string) {
	var file, langs, outDir string
	sourceLang := "auto"

	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "-f":
			if i+1 < len(args) {
				file = args[i+1]
				i++
			}
		case "-l":
			if i+1 < len(args) {
				langs = args[i+1]
				i++
			}
		case "-o":
			if i+1 < len(args) {
				outDir = args[i+1]
				i++
			}
		case "-s":
			if i+1 < len(args) {
				sourceLang = args[i+1]
				i++
			}
		}
	}

	if file == "" || langs == "" {
		fmt.Fprintln(os.Stderr, "Error: -f and -l are required")
		fmt.Fprintln(os.Stderr, "Usage: gt i18n -f <file> -l <langs> [-o <outdir>]")
		os.Exit(1)
	}

	if outDir == "" {
		outDir = "."
	}

	content, err := os.ReadFile(file)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: read file: %v\n", err)
		os.Exit(1)
	}

	targetLangs := strings.Split(langs, ",")
	for i := range targetLangs {
		targetLangs[i] = strings.TrimSpace(targetLangs[i])
	}

	// 初始化翻译引擎（不通过 service, 直接用 engine 更轻量）
	eng, err := engine.New()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: create translator: %v\n", err)
		os.Exit(1)
	}

	translateFn := func(text, target, source string) (string, error) {
		r, err := eng.Translate(context.Background(), text, target, source)
		if err != nil {
			return "", err
		}
		return r.Translated, nil
	}

	// 自动检测文件格式
	format := i18n.DetectFormat(file, string(content))
	fmt.Printf("Translating %s (%s) to [%s]...\n", file, format, langs)
	results, err := i18n.TranslateByFormat(format, string(content), targetLangs, sourceLang, translateFn)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	os.MkdirAll(outDir, 0o755)

	// 根据格式确定输出文件扩展名
	extMap := map[string]string{"json": ".json", "ts": ".ts", "md": ".md", "ini": ".ini"}
	ext := extMap[format]
	if ext == "" {
		ext = filepath.Ext(file)
	}

	for lang, translated := range results {
		outFile := filepath.Join(outDir, lang+ext)
		if err := os.WriteFile(outFile, []byte(translated), 0o644); err != nil {
			fmt.Fprintf(os.Stderr, "Error: write %s: %v\n", outFile, err)
			os.Exit(1)
		}
			fmt.Printf("  -> %s\n", outFile)
	}
	fmt.Println("Done!")
}
