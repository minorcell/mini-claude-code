import { resolve } from "path"

// ── 危险命令检测 ──────────────────────────────────────────────────────────────

export type DangerLevel = "safe" | "confirm" | "block"

// block 级：直接拒绝，没有合法的 Agent 使用场景
const BLOCK_PATTERNS: RegExp[] = [
  /rm\s+-\S*r\S*f\s+(\/|~|\$HOME)\b/, // rm -rf / 或 rm -rf ~
  /dd\s+if=.*of=\/dev\//, // dd 写入磁盘设备
  /mkfs\./, // 格式化文件系统
  />\s*\/dev\/(sda|hda|nvme)/, // 重定向写入磁盘
  /shutdown|reboot|halt/, // 系统关机重启
]

// confirm 级：暂停并等待用户明确确认
const CONFIRM_PATTERNS: RegExp[] = [
  /rm\s+-\S*[rf]/, // rm -r 或 rm -f 类
  /sudo\s+/, // sudo 命令
  /curl\s+.*\|\s*(sh|bash|zsh)/, // curl pipe to shell
  /wget\s+.*\|\s*(sh|bash|zsh)/, // wget pipe to shell
  /npm\s+publish/, // 发包
  /git\s+push\s+.*--force/, // 强制推送
  /git\s+reset\s+--hard/, // 硬重置
]

export function detectDanger(command: string): DangerLevel {
  if (BLOCK_PATTERNS.some((p) => p.test(command))) return "block"
  if (CONFIRM_PATTERNS.some((p) => p.test(command))) return "confirm"
  return "safe"
}

// ── 路径安全检查 ──────────────────────────────────────────────────────────────

// 防止路径穿越攻击：确保解析后的路径在当前工作目录内
export function resolveSafePath(inputPath: string): string {
  const cwd = process.cwd()
  const resolved = resolve(cwd, inputPath)

  if (!resolved.startsWith(cwd + "/") && resolved !== cwd) {
    throw new Error(
      `路径越界：${inputPath} 解析为 ${resolved}，超出工作目录 ${cwd}`
    )
  }

  return resolved
}

// ── 敏感文件检测 ──────────────────────────────────────────────────────────────

const SENSITIVE_PATTERNS: RegExp[] = [
  /\.env(\.|$)/, // .env 文件
  /\.aws\/credentials/, // AWS 凭证
  /\.ssh\/(id_rsa|id_ed25519)$/, // SSH 私钥
  /secrets?\.(json|yaml|yml)$/i, // secrets 文件
]

export function isSensitivePath(path: string): boolean {
  return SENSITIVE_PATTERNS.some((p) => p.test(path))
}
