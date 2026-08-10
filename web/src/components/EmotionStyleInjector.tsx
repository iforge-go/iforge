'use client'

import createCache from '@emotion/cache'
import { useServerInsertedHTML } from 'next/navigation'
import { CacheProvider as EmotionCacheProvider } from '@emotion/react'
import { useState } from 'react'

// Chakra UI v3 使用 Emotion 注入样式。
// 默认 SSR 时 Emotion 把 <style> 标签渲染为组件树的子元素，
// 导致 hydration mismatch（服务端有 <style>，客户端没有）。
//
// 修复：创建自定义 Emotion cache，用 useServerInsertedHTML 把样式注入到 <head>，
// 而不是组件树中。这样服务端和客户端的组件树结构一致。

export function EmotionStyleInjector({ children }: { children: React.ReactNode }) {
  const [{ cache, flush }] = useState(() => {
    const cache = createCache({ key: 'iforge' })
    cache.compat = true
    const prevInsert = cache.insert
    let inserted: string[] = []
    cache.insert = (...args) => {
      const serialized = args[1]
      if (cache.inserted[serialized.name] === undefined) {
        inserted.push(serialized.name)
      }
      return prevInsert(...args)
    }
    const flush = () => {
      const names = inserted
      inserted = []
      return names
    }
    return { cache, flush }
  })

  useServerInsertedHTML(() => {
    const names = flush()
    if (names.length === 0) return null
    let styles = ''
    for (const name of names) {
      styles += cache.inserted[name]
    }
    return (
      <style
        data-emotion={`${cache.key} ${names.join(' ')}`}
        dangerouslySetInnerHTML={{ __html: styles }}
      />
    )
  })

  return (
    <EmotionCacheProvider value={cache}>
      {children}
    </EmotionCacheProvider>
  )
}
