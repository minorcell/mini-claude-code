import { join } from "path"

// 系统提示词分三段拼装：
//   Segment 1: 静态核心指令（SYSTEM_PROMPT.md）
//   Segment 2: 工具使用补充说明（动态，随工具增减自动更新）
//   Segment 3: 运行时状态（可选，如上下文压缩摘要）

const PROMPT_FILE = join(import.meta.dir, "../SYSTEM_PROMPT.md")

export async function assembleSystemPrompt(
  runtimeHints: string[] = []
): Promise<string> {
  const segments: string[] = []

  // Segment 1: 静态指令
  segments.push(await Bun.file(PROMPT_FILE).text())

  // Segment 2: 运行时状态（有则注入）
  if (runtimeHints.length > 0) {
    segments.push("---\n# 运行时状态\n\n" + runtimeHints.join("\n\n"))
  }

  return segments.join("\n\n")
}
