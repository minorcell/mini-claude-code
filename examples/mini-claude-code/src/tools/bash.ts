import { detectDanger } from "../utils/safety"
import { confirmFromUser } from "../utils/confirm"
import { truncateOutput } from "../utils/truncate"

interface Params {
  command: string
  timeout?: number
}

export async function bash({
  command,
  timeout = 30_000,
}: Params): Promise<string> {
  // ── 危险命令检测 ────────────────────────────────────────────────────────────
  const danger = detectDanger(command)

  if (danger === "block") {
    return `拒绝执行：该命令已被自动阻止（高风险操作）。\n命令：${command}`
  }

  if (danger === "confirm") {
    const approved = await confirmFromUser(command)
    if (!approved) {
      // 将拒绝结果返回给 LLM，让它自行调整策略
      return `用户拒绝执行命令：${command}`
    }
  }

  // ── 执行命令 ────────────────────────────────────────────────────────────────
  const controller = new AbortController()
  const timer = setTimeout(() => controller.abort(), timeout)

  let stdout = ""
  let stderr = ""
  let exitCode = 0

  try {
    const proc = Bun.spawn(["sh", "-c", command], {
      stdout: "pipe",
      stderr: "pipe",
    })

    // 超时后 kill
    controller.signal.addEventListener("abort", () => proc.kill())

    ;[stdout, stderr] = await Promise.all([
      new Response(proc.stdout).text(),
      new Response(proc.stderr).text(),
    ])
    exitCode = await proc.exited
  } catch (e) {
    return `执行失败：${(e as Error).message}`
  } finally {
    clearTimeout(timer)
  }

  // ── 整合输出 ────────────────────────────────────────────────────────────────
  const parts: string[] = []
  if (stdout) parts.push(stdout)
  if (stderr) parts.push(`[stderr]\n${stderr}`)
  if (exitCode !== 0) parts.push(`[exit code: ${exitCode}]`)

  const output = parts.join("\n").trim() || "(无输出)"

  return truncateOutput("bash", output)
}
