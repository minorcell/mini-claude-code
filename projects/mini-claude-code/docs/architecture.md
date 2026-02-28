# 整体架构

## 架构图

```bash
┌─────────────────────────────────────────────────────────────┐
│                      mini-claude-code                        │
│                                                             │
│  ┌─────────────────────────────────────────────────────┐   │
│  │  UI Layer — CLI                                     │   │
│  │                                                     │   │
│  │  readline 多轮对话 → 接收用户输入 → 打印执行过程       │   │
│  └──────────────────────────┬──────────────────────────┘   │
│                             │ question                      │
│                             ▼                               │
│  ┌─────────────────────────────────────────────────────┐   │
│  │  Agent Layer                                        │   │
│  │                                                     │   │
│  │  ┌───────────────────────────────────────────────┐  │   │
│  │  │  Context Manager                              │  │   │
│  │  │  token 计数 → 超阈值时压缩 → 重建会话          │  │   │
│  │  └───────────────────────────────────────────────┘  │   │
│  │                                                     │   │
│  │  ┌───────────────────────────────────────────────┐  │   │
│  │  │  AgentLoop (generateText + maxSteps)          │  │   │
│  │  │                                               │  │   │
│  │  │  Observe → Think → Act → Observe → ...       │  │   │
│  │  │                                               │  │   │
│  │  │  onStepFinish: 打印每步，检查 context 大小     │  │   │
│  │  └───────────────────────────────────────────────┘  │   │
│  │                                                     │   │
│  │  ┌───────────────────────────────────────────────┐  │   │
│  │  │  Prompt Assembler                             │  │   │
│  │  │  静态指令 + 工具描述 + 运行时状态              │  │   │
│  │  └───────────────────────────────────────────────┘  │   │
│  └──────────────────────────┬──────────────────────────┘   │
│                             │ tool calls                    │
│                             ▼                               │
│  ┌─────────────────────────────────────────────────────┐   │
│  │  Tools Layer                                        │   │
│  │                                                     │   │
│  │  read_file   write_file   edit_file                 │   │
│  │  bash        web_fetch                              │   │
│  │                                                     │   │
│  │  ┌───────────────────────────────────────────────┐  │   │
│  │  │  Safety Guard                                 │  │   │
│  │  │  危险命令检测 → 用户审批 → 执行 / 取消         │  │   │
│  │  └───────────────────────────────────────────────┘  │   │
│  │                                                     │   │
│  │  ┌───────────────────────────────────────────────┐  │   │
│  │  │  Output Truncator                             │  │   │
│  │  │  超限时截断 → 注入 system_hint                │  │   │
│  │  └───────────────────────────────────────────────┘  │   │
│  └─────────────────────────────────────────────────────┘   │
└─────────────────────────────────────────────────────────────┘
```

## 层级说明

### UI Layer

负责与用户交互，保持多轮对话会话。具体职责：

- 启动 readline 交互
- 接收用户问题，传入 Agent Layer
- 实时展示 Agent 每步的思考和行动
- 支持 `/exit`、`/reset` 等基础 slash 命令

### Agent Layer

核心业务逻辑。包含三个组件：

**AgentLoop**：基于 Vercel AI SDK 的 `generateText`，设置 `maxSteps=20` 实现 ReAct 循环。每步通过 `onStepFinish` 回调打印进度（教学用）并触发 context 检查。

**Context Manager**：在每步完成后计算当前 token 用量，超过阈值时中断循环，执行压缩并重建会话。

**Prompt Assembler**：将静态系统提示词、动态工具描述、运行时状态拼装成最终系统提示词。

### Tools Layer

提供 5 个核心工具，每个工具都经过两道防护：

- **Safety Guard**（仅 bash）：危险命令检测 + 用户审批
- **Output Truncator**（所有工具）：超过字符阈值时截断并附加 `system_hint`

## 数据流

```
用户输入
    │
    ▼
AgentLoop.run(question)
    │
    ├── assembleSystemPrompt()
    │       └── 静态提示词 + 工具描述 + 运行时 hints
    │
    ├── generateText({ system, messages: history, tools, maxSteps: 20 })
    │       │
    │       │  [循环，每步]
    │       ├── LLM 推理 → 输出文本 或 tool_call
    │       │
    │       ├── [如果 tool_call]
    │       │       ├── Safety Guard（bash 工具）
    │       │       ├── 执行工具
    │       │       ├── Output Truncator → tool_result
    │       │       └── tool_result 注入 history，继续循环
    │       │
    │       └── onStepFinish → 打印步骤 + 检查 context 大小
    │
    └── 返回最终文本给 UI Layer
```

## 文件结构

```
projects/mini-claude-code/
│
├── index.ts                    ← CLI 入口，readline 多轮对话
│
├── agent/
│   ├── loop.ts                 ← AgentLoop：generateText 封装
│   ├── context.ts              ← Context Manager：token 计数 + 压缩
│   └── prompt.ts               ← Prompt Assembler：分段组装
│
├── tools/
│   ├── index.ts                ← 工具注册表（ToolRegistry）
│   ├── read-file.ts
│   ├── write-file.ts
│   ├── edit-file.ts
│   ├── bash.ts
│   └── web-fetch.ts
│
├── utils/
│   ├── token.ts                ← token 数量近似估算
│   ├── truncate.ts             ← 工具输出截断 + system_hint 生成
│   └── safety.ts               ← 危险命令正则检测
│
├── SYSTEM_PROMPT.md            ← 静态系统提示词
├── package.json
├── tsconfig.json
└── .env.example
```

## 与 agent-loop 的对比

| 维度 | agent-loop | mini-claude-code |
|------|-----------|-----------------|
| LLM 调用 | 原生 fetch | Vercel AI SDK `generateText` |
| 工具协议 | 手写 XML | SDK 原生 tool calling（JSON schema） |
| 循环控制 | 手写 `for` + 手写解析 | `maxSteps` 参数 |
| 工具类型 | 模拟（天气、时间） | 真实（文件、Shell、网络） |
| 上下文管理 | 无 | token 计数 + 压缩重建 |
| 安全机制 | 无 | 危险命令检测 + 审批 |
| 输出保护 | 无 | 截断 + system_hint |
| 多模型支持 | 需手改代码 | 换 Provider 一行代码 |
