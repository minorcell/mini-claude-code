import type { CoreMessage } from "ai"
import { generateText } from "ai"
import { model } from "./provider"

// 主流模型上下文长度（以 128k 为例，调整此值适配不同模型）
const MODEL_CONTEXT_LIMIT = 128_000
// 超过 80% 时触发压缩，留余量给 LLM 输出和工具结果
const COMPRESS_THRESHOLD = 0.8

// 使用 AI SDK 返回的真实 promptTokens 判断是否需要压缩
export function shouldCompress(promptTokens: number): boolean {
  return promptTokens > MODEL_CONTEXT_LIMIT * COMPRESS_THRESHOLD
}

// 将完整 history 压缩为结构化摘要
// 摘要要能支撑下一轮继续工作：不追求"漂亮"，追求"够用"
export async function compressHistory(
  history: CoreMessage[]
): Promise<string> {
  const COMPRESS_SYSTEM = `
你是一个 Agent 执行历史压缩器。将以下执行历史总结为结构化摘要，输出格式如下（使用 XML 标签）：

<completed>
已完成的具体操作（每行一条，保留关键细节）
</completed>

<remaining>
还未完成的任务或子任务
</remaining>

<current_state>
当前状态：已修改的文件路径、关键变量、环境状态等
</current_state>

<notes>
注意事项：踩过的坑、特殊处理、边界条件
</notes>

要求：信息密度高，去掉废话，保留所有对后续执行有用的细节。
`.trim()

  const historyText = history
    .map((m) => {
      const content =
        typeof m.content === "string"
          ? m.content
          : JSON.stringify(m.content)
      return `[${m.role}]\n${content}`
    })
    .join("\n\n---\n\n")

  const { text } = await generateText({
    model,
    system: COMPRESS_SYSTEM,
    prompt: historyText,
    maxSteps: 1,
  })

  return text
}

// 用压缩摘要重建最小 history 和运行时 hint
export function buildCompressionHint(summary: string): string {
  return [
    "[执行历史摘要 - 之前会话已压缩]",
    "",
    summary,
    "",
    "注意：以上是对之前执行历史的摘要，你处于重建会话状态。",
    "请基于摘要继续完成原始任务，不要重复已完成的操作。",
  ].join("\n")
}
