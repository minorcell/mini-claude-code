# mini-claude-code

> 一个面向 Agent 开发初学者的教学仓库：从 TypeScript + AI SDK 的原型，到 Go + Bubble Tea 的可运行终端 Code Agent。
>
> Bilibili 视频教学：[Agent 到底是什么？从原理、开发到落地的一次真实分享；2026 年 3 月 14 日 - 华中科技大学](https://www.bilibili.com/video/BV1eiwRzPE4n/)
>
> Issue 风格教案：[完整教案 Issue #2](https://github.com/minorcell/mini-claude-code/issues/2)

`mini-opencode` 是当前主线实现。它在终端（TUI）里提供文件操作、工具调用、Shell 执行、Todo 管理和多模型接入能力，方便边学边改边验证。

## 1. 这是什么项目

这是一个“教学 + 实战”混合仓库，目标是帮助你理解并构建一个最小可用的 Code Agent。

- 教学线：`examples/` 里保留从手写 agent loop 到 SDK 化实现的过程
- 工程线：`cmd/mini-opencode` + `internal/*` 提供 Go 版可运行实现
- 产品线：通过 Bubble Tea TUI 把会话、上下文、工具执行放在一个交互界面

## 2. 仓库结构

- `cmd/mini-opencode`：程序入口
- `internal/config`：配置加载与默认值
- `internal/core`：agent loop、session、step 事件广播
- `internal/provider`：统一模型接口，适配 `openai` / `openai-compatible` / `anthropic` / `gemini`
- `internal/tools`：工具注册中心、工作区约束、安全拦截器、内置工具
- `internal/tui`：Bubble Tea 交互界面
- `docs/`：主线项目设计文档
- `examples/`：教学示例（`agent-loop`、`mini-claude-code`）

## 3. 与 TS AI SDK 示例的关系

`examples/mini-claude-code` 是上一阶段教学实现（TypeScript + Vercel AI SDK），重点展示：

- SDK 原生 tool calling
- `generateText + maxSteps` 的多步推理
- 上下文压缩和工具输出截断

Go 主线 `mini-opencode` 在这个基础上继续演进：

- 从脚本式 CLI 过渡到 Bubble Tea TUI
- 从示例级代码过渡到分层架构（config/core/provider/tools/tui）
- 保留教学可读性的同时提高工程可维护性

## 4. 功能概览

界面由三个区域组成：

- `Conversation`：展示 user / assistant / tool 叙事和结果
- `Context`：展示会话状态、token 统计、step 进度、todo 侧栏
- `Composer`：输入区，支持排队下一条消息和 `@文件` 候选补全

## 5. 快速开始

```bash
go run ./cmd/mini-opencode
```

首次启动会自动生成 `~/.mini-opencode/config.yaml`。

如果使用默认 OpenAI 配置，先设置：

```bash
export OPENAI_API_KEY="your_api_key"
```

运行测试：

```bash
go test ./...
```

## 6. 配置

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

## 7. 内置工具

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

## 8. Go Agent 库调研（对照 TS AI SDK）

调研时间：2026-04-13（UTC）

TS 侧常见参考：

- `vercel/ai`（AI SDK）
- `openai/openai-agents-js`（多 Agent 与语音）
- `mastra-ai/mastra`（TS Agent 应用框架）

Go 侧可选库：

| 库 | 定位 | 对 TS AI SDK 的参考关系 | 适用场景 |
| --- | --- | --- | --- |
| `google/adk-go` | Google ADK 的 Go 实现，完整 agent toolkit | 更接近“全栈 agent 框架” | 想快速搭建复杂多 Agent、评估与部署流程 |
| `tmc/langchaingo` | LangChain 的 Go 生态实现 | 类似 TS 里用链路/组件编排替代手写 loop | 需要丰富组件生态与抽象层 |
| `cloudwego/eino` | CloudWeGo 的 LLM/Agent 框架 | 更偏工程化编排与生产实践 | 企业内 Go 服务化集成 |
| `mark3labs/mcp-go` | MCP 协议 Go 实现（client/server） | 对应 TS 生态里的 MCP 接入能力 | 需要与 MCP 工具生态对接 |
| `openai/openai-go` + `anthropics/anthropic-sdk-go` | 官方模型 SDK | 对应 TS SDK 的 provider 层 | 自建轻量 Agent Loop，保留高可控性 |

### 选型建议（结合本仓库）

- 如果保持“教学 + 高可控”路线：继续当前 `core + tools + provider + tui` 架构，优先增强可观测性和工具安全。
- 如果要补齐生态互联：优先引入 `mcp-go`，把外部工具能力标准化接入。
- 如果要快速扩展复杂工作流：可评估 `adk-go` 或 `eino` 做实验分支，对比学习成本与运行开销。

## 9. 交互方式

- `Enter`：发送消息；如果 turn 还在运行，会排队 1 条后续消息
- `Ctrl+J`：在 Composer 里插入换行
- `Esc`：如果有已排队草稿，恢复到 Composer
- `Esc Esc`：中断正在运行的 turn
- `@`：在 Composer 中触发工作区文件候选
- `Up` / `Down`：选择文件候选
- `Tab` 或 `Enter`：接受所选文件候选
- 鼠标滚轮：滚动 `Conversation` 记录
- `Ctrl+C`：退出程序

## 10. 已知边界

- `glob` 走的是 Go `filepath.Glob` 语义，不支持把 `**` 当成递归 doublestar
- 右侧 `Context` 用于展示 token、step、todo 等状态信息；详细工具输出显示在 `Conversation`
- `bash` 默认超时 20 秒，最大 2 分钟，输出上限 64KB
- `read` 和 `webfetch` 的内容上限都是 64KB

## Contributors

![](https://hub-io-mcells-projects.vercel.app/r/minorcell/mini-claude-code)
