# 工具设计

## 设计原则

**工具别贪多。** 每多一个工具，LLM 的决策负担就加重一分。`mini-claude-code` 精选 5 个工具，覆盖 Code Agent 的核心能力：

| 工具 | 能力 | 类比 Claude Code |
|------|------|-----------------|
| `read_file` | 读取文件内容 | Read |
| `write_file` | 写入完整文件 | Write |
| `edit_file` | 局部替换编辑 | Edit |
| `bash` | 执行 Shell 命令 | Bash |
| `web_fetch` | 抓取网页内容 | WebFetch |

有了这 5 个工具，Agent 可以：读代码、改代码、运行测试、查文档、安装依赖——足以完成真实的开发任务。

## 工具注册

```typescript
// tools/index.ts
import { tool } from 'ai'
import { z } from 'zod'
import { readFile } from './read-file'
import { writeFile } from './write-file'
import { editFile } from './edit-file'
import { bash } from './bash'
import { webFetch } from './web-fetch'

export const TOOLS = {
  read_file: tool({
    description: '读取本地文件的内容。对于大文件，建议使用 offset 和 limit 参数分段读取。',
    parameters: z.object({
      path: z.string().describe('文件路径，相对于当前工作目录'),
      offset: z.number().optional().describe('从第几行开始读取（0-indexed）'),
      limit: z.number().optional().describe('最多读取多少行'),
    }),
    execute: readFile,
  }),

  write_file: tool({
    description: '将内容写入文件。如果文件不存在则创建，如果已存在则完整覆盖。适合创建新文件或完整重写，局部修改请用 edit_file。',
    parameters: z.object({
      path: z.string().describe('文件路径，相对于当前工作目录'),
      content: z.string().describe('要写入的完整文件内容'),
    }),
    execute: writeFile,
  }),

  edit_file: tool({
    description: '替换文件中的特定字符串。old_string 必须在文件中唯一存在（不能有重复），否则操作会失败。',
    parameters: z.object({
      path: z.string().describe('文件路径，相对于当前工作目录'),
      old_string: z.string().describe('要被替换的原始字符串，必须在文件中唯一存在'),
      new_string: z.string().describe('替换后的新字符串'),
    }),
    execute: editFile,
  }),

  bash: tool({
    description: '执行 Shell 命令。危险命令（如 rm -rf）会暂停并等待用户确认。命令输出超长时会被截断。',
    parameters: z.object({
      command: z.string().describe('要执行的 Shell 命令'),
      timeout: z.number().optional().describe('超时时间（毫秒），默认 30000'),
    }),
    execute: bash,
  }),

  web_fetch: tool({
    description: '抓取指定 URL 的网页内容，并转换为 Markdown 格式返回。适合查阅文档、读取 README 等。',
    parameters: z.object({
      url: z.string().describe('要抓取的完整 URL'),
    }),
    execute: webFetch,
  }),
}
```

---

## 各工具详细设计

### read_file

```typescript
// tools/read-file.ts
export async function readFile({ path, offset, limit }: ReadFileParams) {
  // 1. 路径安全检查（防止路径穿越）
  const safePath = resolveSafePath(path)

  // 2. 读取文件
  const file = Bun.file(safePath)
  if (!(await file.exists())) {
    return `错误：文件不存在 - ${path}`
  }

  const text = await file.text()
  const lines = text.split('\n')

  // 3. 按 offset/limit 切片
  const startLine = offset ?? 0
  const endLine = limit ? startLine + limit : lines.length
  const slice = lines.slice(startLine, endLine)

  // 4. 加行号（方便 LLM 定位，也方便 edit_file 的 old_string 定位）
  const withLineNumbers = slice
    .map((line, i) => `${startLine + i + 1}\t${line}`)
    .join('\n')

  // 5. 输出截断（见 utils/truncate.ts）
  return truncateOutput('read_file', withLineNumbers)
}
```

**关键设计点：**
- 输出带行号，帮助 LLM 精确定位，减少 edit_file 时 old_string 匹配错误
- `offset`/`limit` 支持分段读取，避免大文件一次性撑爆 context
- 路径安全检查防止读取 `../../etc/passwd` 类的路径穿越攻击

---

### write_file

```typescript
// tools/write-file.ts
export async function writeFile({ path, content }: WriteFileParams) {
  const safePath = resolveSafePath(path)

  // 确保父目录存在
  await mkdir(dirname(safePath), { recursive: true })

  await Bun.write(safePath, content)

  return `success: 已写入 ${path}（${content.length} 字符）`
}
```

**关键设计点：**
- 自动创建父目录（常见场景：创建 `src/components/Button.tsx` 时 `components/` 目录不存在）
- 返回写入字符数，让 LLM 确认写入成功

---

### edit_file

```typescript
// tools/edit-file.ts
export async function editFile({ path, old_string, new_string }: EditFileParams) {
  const safePath = resolveSafePath(path)

  const original = await Bun.file(safePath).text()

  // 唯一性校验：出现次数必须恰好为 1
  const occurrences = original.split(old_string).length - 1
  if (occurrences === 0) {
    return `错误：old_string 在文件中不存在，请检查是否有空格、换行差异`
  }
  if (occurrences > 1) {
    return `错误：old_string 在文件中出现了 ${occurrences} 次，请提供更多上下文使其唯一`
  }

  const updated = original.replace(old_string, new_string)
  await Bun.write(safePath, updated)

  return `success: 已替换 ${path} 中的目标字符串`
}
```

**关键设计点：**
- `old_string` 唯一性校验是 `edit_file` 最重要的设计决策：不唯一的替换会产生难以追踪的错误
- 错误信息要说明原因，帮助 LLM 自修正（"提供更多上下文"）

---

### bash

```typescript
// tools/bash.ts
export async function bash({ command, timeout = 30_000 }: BashParams) {
  // 1. 危险命令检测（见 utils/safety.ts）
  const dangerLevel = detectDanger(command)

  if (dangerLevel === 'block') {
    return `拒绝执行：该命令被列为高风险操作（${command}），已自动阻止。`
  }

  if (dangerLevel === 'confirm') {
    // 暂停，等待用户在 CLI 输入确认
    const approved = await promptUserConfirm(command)
    if (!approved) {
      return `用户拒绝执行命令：${command}`
    }
  }

  // 2. 执行命令
  const proc = Bun.spawn(['sh', '-c', command], {
    stdout: 'pipe',
    stderr: 'pipe',
  })

  // 3. 超时控制
  const timeoutId = setTimeout(() => proc.kill(), timeout)

  const stdout = await new Response(proc.stdout).text()
  const stderr = await new Response(proc.stderr).text()
  const exitCode = await proc.exited

  clearTimeout(timeoutId)

  const output = [
    stdout,
    stderr ? `[stderr]\n${stderr}` : '',
    exitCode !== 0 ? `[exit code: ${exitCode}]` : '',
  ].filter(Boolean).join('\n')

  // 4. 输出截断
  return truncateOutput('bash', output || '(无输出)')
}
```

**关键设计点：**
- 危险命令分两级：`block`（直接拒绝）和 `confirm`（需要用户审批）
- 同时捕获 stdout 和 stderr，LLM 需要看到完整输出才能判断是否成功
- 超时防止命令挂起（如等待用户输入的交互式命令）

---

### web_fetch

```typescript
// tools/web-fetch.ts
export async function webFetch({ url }: WebFetchParams) {
  let response: Response
  try {
    response = await fetch(url, {
      headers: { 'User-Agent': 'mini-claude-code/1.0' },
      signal: AbortSignal.timeout(15_000),
    })
  } catch (e) {
    return `错误：请求失败 - ${e instanceof Error ? e.message : String(e)}`
  }

  if (!response.ok) {
    return `错误：HTTP ${response.status} - ${url}`
  }

  const contentType = response.headers.get('content-type') ?? ''
  const text = await response.text()

  // HTML 转 Markdown（简单的标签清理，保留代码块）
  const markdown = contentType.includes('text/html')
    ? htmlToMarkdown(text)
    : text

  // 截断
  return truncateOutput('web_fetch', markdown)
}
```

**关键设计点：**
- 15 秒超时，防止网络慢的站点卡住整个 Agent
- HTML 转 Markdown 减少 token 消耗（去掉大量 HTML 标签）
- 不需要完整的 HTML 解析库，简单的正则清理已经足够教学场景

---

## 工具输出截断（Output Truncator）

所有工具都经过截断保护，防止单个工具调用撑爆上下文。

```typescript
// utils/truncate.ts

const MAX_TOOL_OUTPUT = 8_000  // 约 2700 tokens

export function truncateOutput(toolName: string, output: string): string {
  if (output.length <= MAX_TOOL_OUTPUT) return output

  const truncated = output.slice(0, MAX_TOOL_OUTPUT)

  // 附加 system_hint，告知 LLM 有内容被省略
  const hint = `
<system_hint type="tool_output_omitted" tool="${toolName}" reason="too_long" actual_chars="${output.length}" max_chars="${MAX_TOOL_OUTPUT}">
  工具输出过长，已自动截断。如需查看更多内容，请使用分段参数（offset/limit）重新调用。
</system_hint>`

  return truncated + hint
}
```

**为什么是 `system_hint` 而不是直接截断？**

直接截断会让 LLM 误以为它看到了完整输出。`system_hint` 明确告知：
1. 输出被截断了（不是内容就这么少）
2. 实际长度是多少（帮助 LLM 评估是否需要分段读取）
3. 如何处理（用 offset/limit 参数重新调用）
