# Mini Claude Code Golang：文件系统 + LLM + Loop（Go 标准库）

一个教学版最小 Agent，只保留三件事：

- 文件系统工具：`listFiles`、`readFile`、`writeFile`
- 大模型调用：兼容 OpenAI 的 `/chat/completions`
- ReAct Loop：`<action>` -> `<observation>` -> `<final>`

## 文件

```txt
mini-claude-code-golang/
├── go.mod
├── main.go
├── tools.go
└── prompt.md
```

## 环境变量

- `LLM_API_KEY` 或 `DEEPSEEK_API_KEY`：必填
- `LLM_BASE_URL`：可选，默认 `https://api.deepseek.com/v1`
- `LLM_MODEL`：可选，默认 `deepseek-chat`

## 运行

```bash
cd projects/mini-claude-code-golang
go run . "请列出当前目录，并解释这个最小 Agent 的工作流程。"
```

如果不传问题，会使用内置默认问题。

## 说明

- 只使用 Go 标准库
- 目录访问被限制在当前工作目录内
- 适合教学，不追求完整工程封装
