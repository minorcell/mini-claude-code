# mini-claude-code 设计文档

本目录是 `mini-claude-code`（教学用 Code Agent）的完整设计文档。

## 阅读顺序

如果你是第一次看，按以下顺序阅读：

1. **[overview.md](./overview.md)** — 项目定位、教学目标、技术选型
2. **[architecture.md](./architecture.md)** — 整体架构、层级划分、数据流、文件结构
3. **[agent-loop.md](./agent-loop.md)** — ReAct Loop 核心、Vercel AI SDK 接入
4. **[tools.md](./tools.md)** — 5 个工具的详细设计与输出截断机制
5. **[context.md](./context.md)** — Token 计数、压缩策略、长任务支持
6. **[security.md](./security.md)** — 危险命令检测、路径安全、用户审批
7. **[prompt-architecture.md](./prompt-architecture.md)** — 系统提示词分段组装

## 项目上下文

```
mini-claude-code/
├── docs/               ← 你在这里
│   └── *.md
├── projects/
│   ├── agent-loop/     ← 前置项目：ReAct 基础演示（XML + 手写解析）
│   └── mini-claude-code/ ← 本项目：Code Agent 工程落地演示
└── README.md
```

`projects/agent-loop` 是前置课——看懂那 40 行核心代码之后，再看这里的工程化版本。
