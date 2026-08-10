'use client'

import { useServerInsertedHTML } from 'next/navigation'

// Chakra UI 颜色模式初始化脚本内容(防止页面加载时颜色闪烁)。
// 从 localStorage 读取用户偏好的颜色模式,设置 data-theme 属性和 colorScheme 样式。
const COLOR_MODE_SCRIPT = `(function(){try{var m=localStorage.getItem('chakra-ui-color-mode')||'light';document.documentElement.style.colorScheme=m;document.documentElement.setAttribute('data-theme',m);}catch(e){}})()`

/**
 * 用 useServerInsertedHTML 在 SSR 流中注入颜色模式初始化脚本。
 *
 * 为什么用 useServerInsertedHTML 而不是直接渲染 <script dangerouslySetInnerHTML>?
 *
 * React 19 改变了对 <script> 标签的处理:在客户端渲染时,React 组件内的 <script>
 * 永远不会被执行,并会发出警告 "Encountered a script tag while rendering React
 * component"。即使用 next/script 或 suppressHydrationWarning 也无法避免。
 *
 * useServerInsertedHTML 是 Next.js 提供的 hook,可以在 SSR 时将 HTML 注入到文档流中,
 * 但这段 HTML 不在 React 的客户端渲染树中,因此 React 不会对它发出警告。
 * 脚本会出现在最终 HTML 中,在浏览器加载时立即执行(在 hydration 之前),
 * 正确设置颜色模式,防止闪烁。
 *
 * 参考:https://oleksiimazurenko.dev/en/blog/nextjs-dark-mode-without-flash
 */
export function ColorModeScriptInjector() {
  useServerInsertedHTML(() => {
    return (
      <script dangerouslySetInnerHTML={{ __html: COLOR_MODE_SCRIPT }} />
    )
  })

  return null
}
