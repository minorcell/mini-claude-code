import { mkdir } from "fs/promises"
import { dirname } from "path"
import { resolveSafePath } from "../utils/safety"

interface Params {
  path: string
  content: string
}

export async function writeFile({ path, content }: Params): Promise<string> {
  let safePath: string
  try {
    safePath = resolveSafePath(path)
  } catch (e) {
    return `错误：${(e as Error).message}`
  }

  // 确保父目录存在（Bun.write 不会自动创建目录）
  await mkdir(dirname(safePath), { recursive: true })

  await Bun.write(safePath, content)

  return `success: 已写入 ${path}（${content.length} 字符）`
}
