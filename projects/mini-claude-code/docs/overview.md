# mini-claude-code 项目概览

## 定位

`mini-claude-code` 是一个用于教学的最小可行 Code Agent 实现。

它的上一课是 `projects/agent-loop`——一个用原生 fetch + 手写 XML 解析展示 ReAct 基础原理的天气查询 Agent。`mini-claude-code` 在此基础上向前走一步：用真实的工具（文件读写、Shell 命令、网络请求）、工程级的框架选型，展示一个最接近 Claude Code 这类真实产品的最小内核。

```
agent-loop          →    mini-claude-code
原理演示                  工程落地演示
XML + 手写解析            Vercel AI SDK 原生 tool calling
天气查询（模拟工具）        文件/Shell/网络（真实工具）
无上下文管理              token 计数 + 压缩
无安全机制                危险命令审批
```

## 教学目标

学完本项目，你应当能够：

1. 理解 `agent-loop` 的手写循环如何被工程框架替代
2. 理解 Code Agent 的核心工具集是什么、为什么是这几个
3. 理解上下文管理在长任务中的必要性
4. 理解 Agent 安全问题的最基本防护思路
5. 能基于此代码扩展自己的 Code Agent

## 技术选型

| 模块 | 选型 | 说明 |
|------|------|------|
| Runtime | Bun + TypeScript | 与 agent-loop 一致，启动快，原生 TS |
| LLM 框架 | Vercel AI SDK | 多 Provider 适配、原生 tool calling、类型安全 |
| 模型服务 | 七牛大模型推理服务 | OpenAI 兼容协议，演示方便切换模型 |
| UI | CLI（readline） | 聚焦核心逻辑，不引入 UI 框架 |

### 为什么是 Vercel AI SDK，不是 LangChain？

`agent-loop` 用原生 fetch 说明了原理，但生产级有几个瓶颈：

- **多模型适配麻烦**：OpenAI、Anthropic、Gemini 的 streaming delta 格式不一致
- **工具调用状态机要手写**：`tool_calls → execute → tool_results` 这个循环
- **类型安全缺失**：XML 结果和工具参数都没有校验

Vercel AI SDK 解决了这三个问题，同时足够轻薄，不引入"框架魔法"——你依然能看到 `generateText` 怎么调用、工具怎么注册、循环怎么运转。

LangChain/LangGraph 封装层太厚，教学场景下会让学员困惑"这个魔法是哪里来的"。

## 文档导航

| 文档 | 内容 |
|------|------|
| [architecture.md](./architecture.md) | 整体架构、层级划分、数据流、文件结构 |
| [agent-loop.md](./agent-loop.md) | ReAct Loop 设计、SDK 接入、循环控制 |
| [tools.md](./tools.md) | 5 个工具的详细设计、输出截断机制 |
| [context.md](./context.md) | Token 计数、压缩策略、滑动窗口 |
| [security.md](./security.md) | 危险命令检测、用户审批、路径安全 |
| [prompt-architecture.md](./prompt-architecture.md) | 系统提示词组装、运行时注入 |
