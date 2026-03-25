# mini-claude-code

一个围绕 Code Agent 的实验与教学仓库。仓库名还是 `mini-claude-code`，但当前主推 project 已切到 `mini-opencode`。

## 当前主推项目

- 项目名：`mini-opencode`
- 文档指代：`@mini-opencode/`
- 仓库目录：`projects/mini-opencode-go/`

`mini-opencode` 是一个基于 Go + Bubble Tea 的终端 Code Agent，目前具备：

- 交互式 TUI：`Conversation` / `Live Trace` / `Composer` 多面板
- 多 provider 支持：`openai`、`openai-compatible`、`anthropic`、`gemini`
- 内置工具：`read`、`write`、`edit`、`list`、`glob`、`grep`、`bash`、`todo`、`webfetch`
- 工作区约束、危险命令拦截、step 追踪与会话管理
- 默认配置文件：`~/.mini-opencode/config.yaml`

## 仓库结构

| 目录                              | 定位              | 说明                                  |
| --------------------------------- | ----------------- | ------------------------------------- |
| `projects/mini-opencode-go/`      | 当前主线          | Go + Bubble Tea 版 `mini-opencode`    |
| `projects/mini-claude-code/`      | TypeScript 教学版 | Bun + Vercel AI SDK 实现的 Code Agent |
| `projects/agent-loop/`            | 最小 ReAct Demo   | 手写 XML 工具调用的天气查询 Agent     |
| `projects/mini-claude-code-rust/` | Rust 教学版       | 文件系统工具 + OpenAI-compatible loop |

`projects/mini-claude-code-with-library/` 目前还是占位空目录，不算正式可运行项目。

## 快速开始

### 运行当前主推项目：mini-opencode

```bash
cd projects/mini-opencode-go
go run ./cmd/mini-opencode
```

首次启动会自动生成 `~/.mini-opencode/config.yaml`。默认 provider 是 OpenAI，需要设置 `OPENAI_API_KEY`。如果改用 `openai-compatible`、`anthropic` 或 `gemini`，请在配置文件里同步填写对应的 `url`、`env_api_key` 和 `model_id`。

### 运行其他教学项目

**agent-loop**

```bash
cd projects/agent-loop
bun install
cp .env.example .env
bun run start
```

需要 `DEEPSEEK_API_KEY`。

**mini-claude-code**

```bash
cd projects/mini-claude-code
bun install
cp .env.example .env
bun start
```

需要 `QINIU_API_KEY`。

**mini-claude-code-rust**

```bash
cd projects/mini-claude-code-rust
cargo run -- "请列出当前目录，并解释这个最小 Agent 的工作流程。"
```

需要 `LLM_API_KEY` 或 `DEEPSEEK_API_KEY`。

## 建议阅读顺序

1. `projects/agent-loop/`：先看最小 ReAct Loop，理解工具调用和 observation 回注。
2. `projects/mini-claude-code/`：再看 TypeScript 工程版，理解 SDK 化后的实现方式。
3. `projects/mini-opencode-go/`：最后看当前主推项目，关注 TUI、配置系统和工具注册表。

## 相关文档

- [mini-opencode README](./projects/mini-opencode-go/README.md)
- [mini-opencode 产品设计](./projects/mini-opencode-go/docs/product-design.md)
- [mini-opencode 架构设计](./projects/mini-opencode-go/docs/architecture-design.md)
- [Mini Claude Code 设计文档](./projects/mini-claude-code/docs)
- [Agent 到底是什么？从原理、开发到落地的一次真实分享；2026 年 3 月 14 日 - 华中科技大学](https://www.bilibili.com/video/BV1eiwRzPE4n/)
- [完整教案 Issue #2](https://github.com/minorcell/mini-claude-code/issues/2)

## 贡献者

![001](https://hub-io-mcells-projects.vercel.app/r/minorcell/mini-claude-code)
