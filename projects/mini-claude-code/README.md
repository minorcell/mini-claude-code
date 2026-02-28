# mini-claude-code

一个教学用 Code Agent，基于 Vercel AI SDK 构建，具备真实的文件读写、Shell 执行和网页抓取能力。

这是 `projects/agent-loop`（手写 ReAct 演示）的进阶版。`agent-loop` 用原始 fetch 手拼 XML 工具调用，帮你理解 ReAct 循环的底层机制。`mini-claude-code` 则换用 Vercel AI SDK，将工具注册、多步推理、消息拼接交给 SDK 处理，让你看到工业级写法是什么样的。

## 与 agent-loop 的区别

| 方面         | agent-loop          | mini-claude-code                    |
| ------------ | ------------------- | ----------------------------------- |
| 工具调用方式 | 手写 XML + 手动解析 | SDK 原生 tool calling（Zod schema） |
| 多步推理     | 手动循环 fetch      | `generateText + maxSteps`           |
| 消息管理     | 手动拼接数组        | SDK 自动维护 `CoreMessage[]`        |
| 上下文保护   | 无                  | token 计数 + 超限压缩 + 输出截断    |

如果你还没看过 `agent-loop`，建议先看那个再来看这个。

## 技术栈

- Bun + TypeScript
- Vercel AI SDK（`generateText`、`maxSteps`、tool calling）
- 七牛大模型推理服务（OpenAI 兼容接口）
- Zod（工具参数 schema 定义）

## 项目结构

```
src/
├── index.ts              # CLI 入口，readline 多轮对话
├── SYSTEM_PROMPT.md      # 静态系统提示词
├── agent/
│   ├── provider.ts       # 七牛 Provider（createOpenAI）
│   ├── loop.ts           # AgentLoop（generateText + maxSteps）
│   ├── context.ts        # 上下文管理（token 计数 + 压缩）
│   └── prompt.ts         # 系统提示词组装
├── tools/
│   ├── index.ts          # 工具注册表
│   ├── read-file.ts      # 带行号，支持 offset/limit 分段读取
│   ├── write-file.ts     # 写入文件，自动创建父目录
│   ├── edit-file.ts      # 局部替换，old_string 唯一性校验
│   ├── bash.ts           # Shell 执行，危险命令暂停等用户确认
│   └── web-fetch.ts      # 抓取网页，HTML 转 Markdown
└── utils/
    ├── truncate.ts       # 工具输出截断 + system_hint 提示
    ├── safety.ts         # 危险命令检测 + 路径安全检查
    └── confirm.ts        # 命令行确认交互
```

## 可用工具

- `read_file`：读取文件，带行号，支持 `offset`/`limit` 分段读取大文件
- `write_file`：写入文件，自动创建不存在的父目录
- `edit_file`：局部字符串替换，校验 `old_string` 在文件中唯一
- `bash`：执行 Shell 命令，检测到危险命令时暂停并等用户确认
- `web_fetch`：抓取网页内容，转为 Markdown 后返回

## 安装与运行

**1. 安装依赖**

```bash
bun install
```

**2. 配置环境变量**

```bash
cp .env.example .env
```

打开 `.env`，填入你的 `QINIU_API_KEY`。

**3. 启动**

```bash
bun start
```

启动后进入交互式对话，直接输入任务即可。

## slash 命令

在对话中可以使用以下命令：

- `/reset`：清空会话历史，开始新对话
- `/help`：显示帮助信息
- `/exit`：退出程序

## 设计说明

**工具输出截断**

单次工具返回超过 8000 字符时自动截断，并附加一条结构化的 `<system_hint>` 告知模型内容被截断、如何用 `offset`/`limit` 分段获取。这避免了大文件一次性撑爆上下文窗口。

**上下文压缩**

每轮对话结束后，根据 SDK 返回的真实 `promptTokens` 判断上下文用量。超过上下文窗口 80% 时，自动压缩历史消息，保留系统提示词和最近几轮对话，重建 `messages` 数组。

**`compatibility: "compatible"` 的作用**

七牛使用 OpenAI 兼容接口，但不完全兼容 OpenAI 规范（例如不支持某些参数）。SDK 的 `createOpenAI` 提供 `compatibility` 选项，设为 `"compatible"` 后 SDK 会跳过 OpenAI 专属字段，只发送兼容部分，适配第三方推理服务的标准配置。
