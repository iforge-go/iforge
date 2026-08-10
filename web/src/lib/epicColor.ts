// Epic 彩色标识工具
//
// 设计参考 Jira:每个 Epic 用稳定颜色标识,在 Backlog/看板/Roadmap 等所有页面一致。
// 采用 HSL 色相空间生成颜色(非持久化):
//   - 零后端改动,立即可用
//   - 同一 slug 永远同一颜色(哈希稳定)
//   - 360 色相值,碰撞率极低(10 色调色板碰撞率约 50%,HSL 几乎为 0)
//   - 固定饱和度(65%)和亮度(45%),保证所有颜色视觉一致性和白底可读性
//   - 后续如需让用户自定义颜色,再加 color 字段持久化

// 默认颜色(无 Epic 或 slug 为空时用灰色)
const EPIC_DEFAULT_COLOR = '#6B7280'

// HSL 转 Hex(#RRGGBB)
function hslToHex(h: number, s: number, l: number): string {
  s /= 100
  l /= 100
  const c = (1 - Math.abs(2 * l - 1)) * s
  const x = c * (1 - Math.abs(((h / 60) % 2) - 1))
  const m = l - c / 2
  let r = 0, g = 0, b = 0
  if (h < 60)       { r = c; g = x; b = 0 }
  else if (h < 120) { r = x; g = c; b = 0 }
  else if (h < 180) { r = 0; g = c; b = x }
  else if (h < 240) { r = 0; g = x; b = c }
  else if (h < 300) { r = x; g = 0; b = c }
  else              { r = c; g = 0; b = x }
  const toHex = (v: number) => Math.round((v + m) * 255).toString(16).padStart(2, '0')
  return `#${toHex(r)}${toHex(g)}${toHex(b)}`
}

// 基于 Epic slug 哈希生成稳定颜色
// 同一 slug 永远返回同一颜色,跨页面/跨会话一致
// 使用 HSL 色相空间(0-360),碰撞率远低于固定调色板取模
export function getEpicColor(epicSlug: string | null | undefined): string {
  if (!epicSlug) return EPIC_DEFAULT_COLOR
  let hash = 0
  for (let i = 0; i < epicSlug.length; i++) {
    hash = ((hash << 5) - hash) + epicSlug.charCodeAt(i)
    hash |= 0 // 转为 32 位整数
  }
  // 映射到 HSL 色相空间,固定饱和度和亮度
  const hue = Math.abs(hash) % 360
  return hslToHex(hue, 65, 45)
}

// 获取颜色的浅色背景版本(用于 Badge 背景,16% 透明度叠加)
// 输入 #RRGGBB,返回 rgba(r,g,b,0.16)
export function getEpicBgColor(color: string): string {
  const hex = color.replace('#', '')
  if (hex.length !== 6) return `${color}20`
  const r = parseInt(hex.substring(0, 2), 16)
  const g = parseInt(hex.substring(2, 4), 16)
  const b = parseInt(hex.substring(4, 6), 16)
  return `rgba(${r}, ${g}, ${b}, 0.16)`
}
