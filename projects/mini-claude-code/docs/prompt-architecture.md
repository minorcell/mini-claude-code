# 系统提示词架构

## 问题：提示词不只是一个文件

简单的 Agent 把系统提示词写死在代码或一个 `.md` 文件里。但在 `mini-claude-code` 中，系统提示词需要包含**不同来源、不同性质**的内容：

| 内容类型 | 来源 | 特点 |
|---------|------|------|
| 基础行为指令 | `SYSTEM_PROMPT.md` 静态文件 | 固定，不变 |
| 工具描述 | 工具注册表动态生成 | 随工具增减变化 |
| 运行时状态 | 执行过程中动态注入 | 每次请求可能不同 |

这就需要一套**组装逻辑**，而不是一个静态字符串。

## 提示词结构

最终发送给 LLM 的系统提示词由三个段落拼接：

```
┌─────────────────────────────────────────────────────┐
│  Segment 1: 核心指令（静态）                          │
│  来源：SYSTEM_PROMPT.md                              │
│  内容：角色定义、行为准则、输出格式要求               │
├─────────────────────────────────────────────────────┤
│  Segment 2: 工具能力（动态）                          │
│  来源：tools/index.ts 自动生成                       │
│  内容：每个工具的用途和使用建议                       │
├─────────────────────────────────────────────────────┤
│  Segment 3: 运行时状态（动态，可选）                  │
│  来源：执行过程中注入                                 │
│  内容：上下文压缩摘要、异常提示、特殊约束             │
└─────────────────────────────────────────────────────┘
```

## SYSTEM_PROMPT.md（静态段）

```markdown
# 角色

你是 mini-claude-code，一个运行在用户本地终端的 Code Agent。你可以读写文件、执行 Shell 命令、访问网络，帮助用户完成代码开发任务。

# 行为准则

1. **先理解，再行动**：在执行任何修改前，先读取相关文件，确认你理解了上下文。
2. **最小化操作**：只修改完成任务所必需的内容，不引入无关改动。
3. **及时汇报**：每次工具调用后，说明你做了什么，发现了什么。
4. **遇到不确定，停下来问**：不要猜测用户意图，不确定时直接提问。

# 输出规范

- 使用中文与用户交流
- 代码块使用对应语言的语法高亮
- 任务完成后给出简洁的总结，说明做了哪些修改

# 工具使用建议

- 修改文件前，先用 read_file 读取内容，确认目标位置
- 对大文件，用 read_file 的 offset/limit 参数分段读取，不要一次读全量
- 优先用 edit_file 做局部修改，只在需要完整重写时用 write_file
- bash 命令执行前，确认命令的影响范围
```

## 组装逻辑

```typescript
// agent/prompt.ts

export async function assembleSystemPrompt(
  runtimeHints?: string[]
): Promise<string> {
  const segments: string[] = []

  // Segment 1: 静态核心指令
  const staticPrompt = await Bun.file('SYSTEM_PROMPT.md').text()
  segments.push(staticPrompt)

  // Segment 2: 工具能力描述
  // 注意：Vercel AI SDK 会自动将工具 schema 注入到 function calling，
  // 这里的工具描述是面向"使用建议"层面的补充说明，不是参数 schema
  const toolsDescription = buildToolsDescription()
  if (toolsDescription) {
    segments.push(`\n---\n# 可用工具补充说明\n\n${toolsDescription}`)
  }

  // Segment 3: 运行时状态（如有）
  if (runtimeHints && runtimeHints.length > 0) {
    segments.push(`\n---\n# 运行时状态\n\n${runtimeHints.join('\n\n')}`)
  }

  return segments.join('\n')
}

function buildToolsDescription(): string {
  return `
- **read_file**：读取文件内容。超长文件会被截断，使用 offset/limit 分段读取。
- **write_file**：写入完整文件内容，适合创建新文件或完整重写。
- **edit_file**：替换文件中的特定字符串，old_string 必须在文件中唯一存在。
- **bash**：执行 Shell 命令。危险命令会暂停并等待用户确认。
- **web_fetch**：抓取网页内容并转换为 Markdown 格式。
`.trim()
}
```

## 运行时状态注入

某些特殊情况需要在系统提示词中动态插入状态说明：

### 上下文压缩后的摘要注入

```typescript
// 压缩完成后，将摘要作为运行时 hint 注入
const hint = `[上下文摘要 - 之前会话的执行记录]
${summary}

注意：以上是对之前执行历史的压缩摘要，你现在处于会话重启后的状态，请基于上述摘要继续完成任务。`

const newSystem = await assembleSystemPrompt([hint])
```

### 工具异常状态注入

如果某个工具出现持续异常，可以注入提示：

```typescript
const hint = `<system_hint type="tool_degraded" tool="web_fetch">
  web_fetch 工具当前不可用（网络连接问题），请避免使用该工具。
</system_hint>`
```

## system_hint 在提示词中的位置

`system_hint` 有两种使用场景，位置不同：

| 场景 | 位置 | 原因 |
|------|------|------|
| 工具输出超限 | 工具返回值中（作为 tool_result） | 直接跟随工具调用，模型立刻知道结果被截断 |
| 工具不可用、全局状态 | 系统提示词 Segment 3 | 影响全局行为，放在系统提示词层面更合适 |

## 提示词工程注意事项

### 工具描述的质量直接影响模型行为

Vercel AI SDK 自动生成的工具 schema 会注入到 function calling，模型会读取这些描述来决定何时调用工具。`tool()` 中的 `description` 要写清楚：

```typescript
// 差：太模糊
description: '读取文件'

// 好：包含何时用、有什么限制
description: '读取本地文件的内容。对于大文件，建议使用 offset 和 limit 参数分段读取，而不是一次性读取全量内容。'
```

### 避免在静态提示词中写死工具参数格式

工具参数格式由 Zod schema 自动生成，在静态提示词里重复描述参数格式会导致不一致（代码改了但提示词没改）。静态提示词只写**使用策略**，不写**参数格式**。

### 提示词分段用 `---` 分隔线

各段之间用 Markdown 分隔线（`---`）和 `#` 标题区分，帮助模型识别不同来源的指令，也方便开发者调试时快速定位各段内容。
