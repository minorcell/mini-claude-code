# Mini Claude Code

> Agent 开发实战课 —— 从零构建一个 Mini Claude Code（TypeScript）

## 收益

- 理解为什么 Agent 能"做事情"，而 ChatBot 不能
- 听得懂 ReAct 与 Agent 基本架构
- 能跑通最小 TypeScript Agent
- 带走可落地的工程经验

## 核心概念

**Agent = ReAct + Tools + UI**

| 概念       | 说明                                             |
| ---------- | ------------------------------------------------ |
| 上下文窗口 | 维护消息历史数组，作为模型的完整输入             |
| 系统提示词 | 定义 Agent 身份、能力边界与输出规范              |
| 工具调用   | 模型生成结构化指令，触发外部能力，结果回注上下文 |
| ReAct 循环 | 观察 → 思考 → 行动，控制流从人转移到模型         |

## 教案目录

1. **Agent 是怎么工作的？** — ChatBot vs Agent 的本质差异
2. **ReAct：让模型"边想边做"** — 循环架构与控制流转移
3. **Agent 最小架构** — `ReAct + Tools + UI` 公式拆解
4. **最小 Agent 实现** — 天气查询 Demo，40 行核心 Loop
5. **Mini Claude Code 设计** — 拆解 Claude Code，引入 Vercel AI SDK
6. **工程经验** — 上下文管理、安全防护、系统提示词架构

完整教案：[Issue #2](https://github.com/minorcell/mini-claude-code/issues/2)

## 项目结构

```
mini-claude-code/
├── projects/
│   ├── agent-loop/          # 最小 Agent Demo：天气查询（Bun + TypeScript）
│   │   ├── main.ts          # 40 行 AgentLoop 核心实现
│   │   ├── tools.ts         # 工具定义（获取时间、查询天气）
│   │   └── prompt.md        # ReAct 格式系统提示词
│   └── mini-claude-code/    # Mini Claude Code 完整实现
│       ├── src/             # 核心源码
│       └── docs/            # 设计文档
└── README.md
```

## 快速开始

**运行最小 Agent Demo（天气查询）：**

```bash
cd projects/agent-loop
bun install
bun main.ts
```

**运行 Mini Claude Code：**

```bash
cd projects/mini-claude-code
bun install
bun src/index.ts
```

> 需要配置 `QINIU_API_KEY` 等对应的环境变量，详见具体项目内容。

## Mini Claude Code 工具集

精简到 4 个核心工具，够用不乱：

| 工具         | 说明            |
| ------------ | --------------- |
| Read         | 读取文件内容    |
| Write / Edit | 写入与修改文件  |
| Bash         | 执行 Shell 命令 |
| WebFetch     | 网络请求        |

> 工具别贪多。每多一个，模型负担就加重。

## 技术栈

- **Runtime**：[Bun](https://bun.sh/)
- **Language**：TypeScript
- **AI SDK**：[Vercel AI SDK](https://ai-sdk.dev/)
- **模型服务**：兼容 OpenAI 协议的任意服务（演示使用七牛大模型推理）

## 相关链接

- [完整教案 Issue #2](https://github.com/minorcell/mini-claude-code/issues/2)
- [Vercel AI SDK 最小用法 Issue #3](https://github.com/minorcell/mini-claude-code/issues/3)
- [Mini Claude Code 设计文档](./projects/mini-claude-code/docs)
