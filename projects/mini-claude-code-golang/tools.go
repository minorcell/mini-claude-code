package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// 这两个结构体对应工具输入的 JSON。
type pathInput struct {
	Path string `json:"path"`
}

type writeInput struct {
	Path    string `json:"path"`
	Content string `json:"content"`
}

// callTool 负责按名字分发工具。
func callTool(root, name, raw string) string {
	switch name {
	case "listFiles":
		return listFiles(root, raw)
	case "readFile":
		return readFile(root, raw)
	case "writeFile":
		return writeFile(root, raw)
	default:
		return "未知工具: " + name
	}
}

// listFiles 列出目录内容。
func listFiles(root, raw string) string {
	in := pathInput{Path: "."}
	if strings.TrimSpace(raw) != "" && json.Unmarshal([]byte(raw), &in) != nil {
		return `listFiles 参数应为 {"path":"."}`
	}

	target, show, err := safePath(root, in.Path)
	if err != nil {
		return err.Error()
	}

	entries, err := os.ReadDir(target)
	if err != nil {
		return err.Error()
	}
	if len(entries) == 0 {
		return show + " 是空目录"
	}

	lines := []string{"目录 " + show + ":"}
	for _, entry := range entries {
		name := entry.Name()
		if entry.IsDir() {
			name += "/"
		}
		lines = append(lines, name)
	}
	return strings.Join(lines, "\n")
}

// readFile 读取文件内容；太大时只截前面一部分。
func readFile(root, raw string) string {
	var in pathInput
	if json.Unmarshal([]byte(raw), &in) != nil || strings.TrimSpace(in.Path) == "" {
		return `readFile 参数应为 {"path":"main.go"}`
	}

	target, show, err := safePath(root, in.Path)
	if err != nil {
		return err.Error()
	}

	data, err := os.ReadFile(target)
	if err != nil {
		return err.Error()
	}
	if len(data) > 8000 {
		return "文件 " + show + ":\n" + string(data[:8000]) + "\n...[已截断]"
	}
	return "文件 " + show + ":\n" + string(data)
}

// writeFile 写入文件；父目录不存在时顺手创建。
func writeFile(root, raw string) string {
	var in writeInput
	if json.Unmarshal([]byte(raw), &in) != nil || strings.TrimSpace(in.Path) == "" {
		return `writeFile 参数应为 {"path":"a.txt","content":"hello"}`
	}

	target, show, err := safePath(root, in.Path)
	if err != nil {
		return err.Error()
	}

	if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
		return err.Error()
	}
	if err := os.WriteFile(target, []byte(in.Content), 0o644); err != nil {
		return err.Error()
	}
	return "已写入 " + show
}

// safePath 保证工具只能访问当前工作目录。
func safePath(root, p string) (string, string, error) {
	if strings.TrimSpace(p) == "" {
		p = "."
	}

	rootAbs, err := filepath.Abs(root)
	if err != nil {
		return "", "", err
	}
	full, err := filepath.Abs(filepath.Join(rootAbs, p))
	if err != nil {
		return "", "", err
	}

	rel, err := filepath.Rel(rootAbs, full)
	if err != nil {
		return "", "", err
	}
	if rel == ".." || strings.HasPrefix(rel, ".."+string(os.PathSeparator)) {
		return "", "", fmt.Errorf("路径超出工作目录: %s", p)
	}
	if rel == "." {
		return full, ".", nil
	}
	return full, rel, nil
}
