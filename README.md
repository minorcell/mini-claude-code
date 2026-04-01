# mini-claude-code

`mini-opencode` 是一个 Code Agent 项目。它基于 Go + Bubble Tea，提供终端内的代码读取、工具调用、shell 执行与 todo 管理能力，并将过程保留在同一个 TUI 会话里。

## 仓库结构

- `cmd/mini-opencode`：程序入口
- `internal/config`：配置加载与默认值
- `internal/core`：agent loop、session、逐步事件广播
- `internal/provider`：统一模型接口，适配 `openai` / `openai-compatible` / `anthropic` / `gemini`
- `internal/tools`：工具注册中心、工作区约束、安全拦截器、内置工具
- `internal/tui`：Bubble Tea 交互界面
- `docs/`：主线项目设计文档
- `examples/`：教学示例（`agent-loop`、`mini-claude-code`）

## 功能概览

界面由三个区域组成：

- `Conversation`：完整展示 user / assistant / tool 的叙事和结果
- `Context`：展示会话状态、token 统计、step 进度、todo 侧栏
- `Composer`：输入区，支持排队下一条消息和 `@文件` 候选补全

## 快速开始

```bash
go run ./cmd/mini-opencode
```

首次启动会自动生成 `~/.mini-opencode/config.yaml`。

如果使用默认 OpenAI 配置，先设置：

```bash
export OPENAI_API_KEY="your_api_key"
```

跑测试：

```bash
go test ./...
```

## 配置

默认配置文件路径：

```txt
~/.mini-opencode/config.yaml
```

最小示例：

```yaml
workspace: ~/code/my-project
max_tokens: 1024
max_steps: 24
temperature: 0.2

provider:
  name: OpenAI
  type: openai
  url: https://api.openai.com/v1
  env_api_key: OPENAI_API_KEY
  model_id: gpt-4.1-mini
```

使用 DeepSeek 这类 OpenAI-compatible 网关时：

```yaml
provider:
  name: DeepSeek
  type: openai-compatible
  url: https://api.deepseek.com/v1
  env_api_key: DEEPSEEK_API_KEY
  model_id: deepseek-chat
```

`provider.type` 支持：

- `openai`
- `openai-compatible`
- `anthropic`
- `gemini`

默认值规则：

- `openai -> url=https://api.openai.com/v1 env_api_key=OPENAI_API_KEY model_id=gpt-4.1-mini`
- `anthropic -> url=https://api.anthropic.com/v1 env_api_key=ANTHROPIC_API_KEY model_id=claude-3-7-sonnet-latest`
- `gemini -> url=https://generativelanguage.googleapis.com/v1beta env_api_key=GEMINI_API_KEY model_id=gemini-2.0-flash`
- `openai-compatible -> 不预设 url / env_api_key / model_id，需要显式配置`

补充说明：

- `provider.name` 只是界面显示名，不参与协议分发
- `workspace` 为空时默认使用启动程序时的工作目录
- `workspace`、`provider.url`、`provider.model_id` 支持环境变量展开
- `provider.type` 会自动规范化别名，例如 `compatible` 会折叠成 `openai-compatible`

## 内置工具

| 工具 | 说明 |
| --- | --- |
| `read` | 读取文件，返回带行号内容，支持 `offset` / `limit` / `max_bytes` |
| `write` | 写入文件，自动创建父目录，支持覆盖或追加 |
| `edit` | 精确替换已有内容，默认要求 `old_content` 唯一 |
| `list` | 列目录，支持递归和显示隐藏文件 |
| `glob` | 按 Go 标准 glob 规则匹配文件名 |
| `grep` | 在文件内容中搜索文本或正则，返回 `file:line: content` |
| `bash` | 在工作区内执行 `/bin/sh -lc` 命令 |
| `todo` | 维护任务 todo 列表，右侧 `Context` 会渲染状态 |
| `webfetch` | 抓取 HTTP(S) 页面或接口，HTML 会被剥离为纯文本 |

## 交互方式

- `Enter`：发送消息；如果 turn 还在运行，会排队 1 条后续消息
- `Ctrl+J`：在 Composer 里插入换行
- `Esc`：如果有已排队草稿，恢复到 Composer
- `Esc Esc`：中断正在运行的 turn
- `@`：在 Composer 中触发工作区文件候选
- `Up` / `Down`：选择文件候选
- `Tab` 或 `Enter`：接受所选文件候选
- 鼠标滚轮：滚动 `Conversation` 记录
- `Ctrl+C`：退出程序

## 已知边界

- `glob` 走的是 Go `filepath.Glob` 语义，不支持把 `**` 当成递归 doublestar
- 右侧 `Context` 用于展示 token、step、todo 等状态信息；详细工具输出显示在 `Conversation`
- `bash` 默认超时 20 秒，最大 2 分钟，输出上限 64KB
- `read` 和 `webfetch` 的内容上限都是 64KB

## 相关文档

- [产品设计](./docs/product-design.md)
- [架构设计](./docs/architecture-design.md)
- [Mini Claude Code 设计文档](./examples/mini-claude-code/docs)
- [Agent 到底是什么？从原理、开发到落地的一次真实分享；2026 年 3 月 14 日 - 华中科技大学](https://www.bilibili.com/video/BV1eiwRzPE4n/)
- [完整教案 Issue #2](https://github.com/minorcell/mini-claude-code/issues/2)
