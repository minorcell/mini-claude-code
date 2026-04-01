# 上下文管理

## 问题背景

ReAct 循环的每一步都会往 `history` 里追加消息：LLM 的思考、工具调用、工具结果……任务越复杂，history 越长，迟早触碰模型的上下文长度上限。

更糟糕的是：**上下文变长不只是会报错，还会让模型开始"遗忘"**——太长的上下文里，早期的关键信息会被"稀释"，模型的表现会下降。

`mini-claude-code` 的上下文管理策略：**计数 → 预警 → 压缩 → 重建**。

## Token 计数

精确的 token 计数需要分词器，成本高且与模型绑定。教学场景下，用字符数近似：

```typescript
// utils/token.ts

// 中英文混合，平均每 token 约 2-3 个字符，这里用 2.5 近似
const CHARS_PER_TOKEN = 2.5

export function estimateTokens(text: string): number {
  return Math.ceil(text.length / CHARS_PER_TOKEN)
}

export function estimateMessagesTokens(messages: CoreMessage[]): number {
  return messages.reduce((total, msg) => {
    const content = typeof msg.content === 'string'
      ? msg.content
      : JSON.stringify(msg.content)  // tool_call / tool_result 是对象
    return total + estimateTokens(content)
  }, 0)
}
```

## 阈值设计

```typescript
// agent/context.ts

// 主流模型上下文长度（以 128k 为例）
const MODEL_CONTEXT_LIMIT = 128_000   // tokens

// 80% 时触发压缩：留 20% 给 LLM 输出和本轮工具结果
const COMPRESS_THRESHOLD = 0.8

// 系统提示词占用（静态部分）
const SYSTEM_PROMPT_RESERVE = 2_000

export function shouldCompress(
  history: CoreMessage[],
  systemPrompt: string
): boolean {
  const historyTokens = estimateMessagesTokens(history)
  const systemTokens = estimateTokens(systemPrompt)
  const totalUsed = historyTokens + systemTokens

  return totalUsed > MODEL_CONTEXT_LIMIT * COMPRESS_THRESHOLD
}
```

## 压缩流程

触发压缩后，执行以下步骤：

```
当前 history（很长）
        │
        ▼
  让 LLM 生成摘要
        │
        ▼
  摘要包含 4 部分：
    1. 已完成的任务
    2. 尚未完成的任务
    3. 当前状态（修改了哪些文件、关键发现）
    4. 注意事项（踩过的坑、边界条件）
        │
        ▼
  重建 history：
    [system] 原系统提示词 + 压缩摘要 hint
    [user]   原始用户问题（保留）
        │
        ▼
  继续执行 AgentLoop
```

### 压缩提示词

```typescript
// agent/context.ts

const COMPRESS_PROMPT = `
请对以下 Agent 执行历史进行压缩总结，输出格式如下（使用 XML 标签）：

<completed>
已完成的具体操作列表（每行一条）
</completed>

<remaining>
还未完成的任务或子任务
</remaining>

<current_state>
当前状态：已修改的文件、关键变量值、环境状态等
</current_state>

<notes>
重要注意事项：踩过的坑、特殊处理、边界条件
</notes>

要求：
- 信息密度高，不要废话
- 保留所有对后续执行有用的细节
- 忘记所有已经完成且不影响后续的步骤细节
`.trim()
```

### 压缩实现

```typescript
export async function compressHistory(
  history: CoreMessage[],
  model: LanguageModelV1
): Promise<string> {
  // 把完整 history 拼成文本交给 LLM 总结
  const historyText = history
    .map(m => {
      const content = typeof m.content === 'string'
        ? m.content
        : JSON.stringify(m.content)
      return `[${m.role}]\n${content}`
    })
    .join('\n\n---\n\n')

  const { text } = await generateText({
    model,
    system: COMPRESS_PROMPT,
    prompt: historyText,
    maxSteps: 1,  // 只要一次输出，不循环
  })

  return text
}
```

## 重建会话

压缩完成后，用摘要重建 history：

```typescript
export async function rebuildHistory(
  originalQuestion: string,
  summary: string,
  systemPrompt: string,
  model: LanguageModelV1,
): Promise<{ history: CoreMessage[], newSystem: string }> {

  // 将摘要注入系统提示词（作为运行时状态段）
  const summaryHint = `
[执行历史摘要 - 之前会话已压缩]

${summary}

注意：以上是对之前执行历史的摘要，你当前处于重建会话的状态。
请基于上述摘要继续完成原始任务，不要重复已完成的操作。`.trim()

  const newSystem = await assembleSystemPrompt([summaryHint])

  // 重建 history：只保留原始问题
  const history: CoreMessage[] = [
    { role: 'user', content: originalQuestion },
  ]

  return { history, newSystem }
}
```

## 完整流程图

```
AgentLoop.run(question)
    │
    ├── [每步 onStepFinish]
    │       │
    │       ▼
    │   shouldCompress(history) ?
    │       │
    │    是 │                    否
    │       ▼                    ▼
    │   throw ContextOverflow   继续
    │
    ├── [捕获 ContextOverflow]
    │       │
    │       ▼
    │   summary = compressHistory(history, model)
    │       │
    │       ▼
    │   { history, newSystem } = rebuildHistory(question, summary)
    │       │
    │       ▼
    │   AgentLoop.run(question, { history, system: newSystem })
    │   （递归，直到任务完成）
    │
    └── 返回最终文本
```

## 工具输出是最大的上下文"杀手"

上下文暴涨通常不是用户消息造成的，而是工具输出——一次 `read_file` 读取大文件，一次 `bash` 输出大量日志，瞬间消耗数千 token。

所以**工具输出截断**（见 [tools.md](./tools.md)）是上下文管理的第一道防线，压缩机制是第二道防线。两道防线配合，才能支撑长时间运行的任务。

## 教学提示

对于简单任务（几步就完成的），这套机制完全不会触发——上下文几千 token 远未到阈值。

它的价值在"长任务"场景：让 Agent 调试一个复杂的 Bug、重构一个模块、写一整套测试……这时上下文管理的存在与否，决定了 Agent 能不能坚持到终点。
