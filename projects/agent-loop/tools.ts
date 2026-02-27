export type ToolName = "getWeather" | "getTime"
export type ToolFn = (input: string) => Promise<string>

type WeatherInput =
    | { error: string }
    | { city: string; time: string }

/**
 * 解析 getWeather 工具的输入参数，期望输入为 JSON 字符串，包含 city 和 time 字段
 * @param input 原始输入字符串
 * @returns 解析后的 WeatherInput 对象，包含 city 和 time，或 error 字段描述解析错误
 */
function parseWeatherInput(input: string): WeatherInput {
    try {
        const parsed = JSON.parse(input)
        const city = parsed?.city
        const time = parsed?.time

        if (!city || typeof city !== "string") {
            return { error: "getWeather 需要 city 字符串" }
        }

        if (!time || typeof time !== "string") {
            return { error: "getWeather 需要 time 字符串" }
        }

        return { city: city.trim(), time: time.trim() }
    } catch {
        return { error: 'getWeather 参数需为 JSON，如 {"city":"上海","time":"2026-02-27 10:00"}' }
    }
}

/**
 * 根据城市和时间生成模拟天气信息
 * @param city 城市名称
 * @param time 时间字符串，格式不限，但建议包含日期和时间信息
 * @returns 天气信息字符串
 */
function buildMockWeather(city: string, time: string): string {
    const conditions = ["晴", "多云", "阴", "小雨", "阵雨"]
    const winds = ["东北风 2 级", "东风 3 级", "西南风 2 级", "北风 1 级"]
    const seed = Array.from(`${city}|${time}`).reduce(
        (acc, ch) => acc + ch.charCodeAt(0),
        0,
    )
    const condition = conditions[seed % conditions.length] ?? "晴"
    const wind = winds[seed % winds.length] ?? "微风 1 级"
    const temp = 12 + (seed % 20)
    const humidity = 35 + (seed % 55)
    return `天气信息：${city} 在 ${time} 的天气为${condition}，气温 ${temp}°C，${wind}，湿度 ${humidity}%。`
}

/**
 * 工具函数集合，供 Agent 调用
 */
export const TOOLKIT: Record<ToolName, ToolFn> = {
    async getTime() {
        return new Date().toISOString()
    },

    async getWeather(rawInput: string) {
        const parsed = parseWeatherInput(rawInput.trim())
        if ("error" in parsed) return parsed.error
        return buildMockWeather(parsed.city, parsed.time)
    },
}
