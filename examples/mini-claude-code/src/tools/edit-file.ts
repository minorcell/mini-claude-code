import { resolveSafePath } from "../utils/safety"

interface Params {
  path: string
  old_string: string
  new_string: string
}

export async function editFile({
  path,
  old_string,
  new_string,
}: Params): Promise<string> {
  let safePath: string
  try {
    safePath = resolveSafePath(path)
  } catch (e) {
    return `错误：${(e as Error).message}`
  }

  const file = Bun.file(safePath)
  if (!(await file.exists())) {
    return `错误：文件不存在 - ${path}`
  }

  const original = await file.text()

  // 唯一性校验：old_string 必须恰好出现 1 次
  // 不唯一的替换会产生难以追踪的错误，所以这里严格校验
  const occurrences = original.split(old_string).length - 1

  if (occurrences === 0) {
    return [
      `错误：old_string 在 ${path} 中不存在。`,
      `请先用 read_file 读取文件，确认目标字符串（注意空格和换行）。`,
    ].join("\n")
  }

  if (occurrences > 1) {
    return [
      `错误：old_string 在 ${path} 中出现了 ${occurrences} 次，无法唯一定位。`,
      `请在 old_string 中加入更多上下文（前后几行）使其唯一。`,
    ].join("\n")
  }

  const updated = original.replace(old_string, new_string)
  await Bun.write(safePath, updated)

  return `success: 已替换 ${path} 中的目标字符串`
}
