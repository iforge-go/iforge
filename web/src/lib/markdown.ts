/**
 * Markdown 预处理工具
 * 
 * react-markdown + rehype-raw 遇到 HTML 块时，块内内容会被当作纯文本，
 * 导致 <div align="center"> 内的 # heading、![badge] 等不会被解析。
 * 
 * 此函数提取 HTML 块内的 markdown 并转换，再重新包裹 HTML 标签。
 */

// 将简单的 markdown 片段转为 HTML（仅转换语法，不包裹段落）
function convertInlineMarkdown(md: string): string {
  return md
    // 图片 ![alt](url)
    .replace(/!\[([^\]]*)\]\(([^)]+)\)/g, '<img src="$2" alt="$1" />')
    // 标题 ###### ~ #
    .replace(/^###### (.+)$/gm, '<h6>$1</h6>')
    .replace(/^##### (.+)$/gm, '<h5>$1</h5>')
    .replace(/^#### (.+)$/gm, '<h4>$1</h4>')
    .replace(/^### (.+)$/gm, '<h3>$1</h3>')
    .replace(/^## (.+)$/gm, '<h2>$1</h2>')
    .replace(/^# (.+)$/gm, '<h1>$1</h1>')
    // 粗体 **text**
    .replace(/\*\*([^*]+)\*\*/g, '<strong>$1</strong>')
    // 链接 [text](url)
    .replace(/\[([^\]]+)\]\(([^)]+)\)/g, '<a href="$2">$1</a>')
}

/**
 * 预处理 markdown 内容，使 HTML 块内的 markdown 能被正确解析
 * 
 * 对于 align="center" 容器，移除 badges 之间的空行，
 * 让 react-markdown 把它们放在同一个 <p> 里水平排列
 */
export function preprocessMarkdown(content: string): string {
  if (!content) return content
  
  // 匹配 <tag ...>...</tag> 块，提取内部内容并转换
  // 使用 [\s\S]*? 跨行匹配（非贪婪），属性部分可选
  const result = content.replace(
    /<(div|p|section|article|aside|main|header|footer|nav)(\s[^>]*)?>([\s\S]*?)<\/\1>/gi,
    (match, tag, attrs, inner) => {
      // 先转换 markdown 语法
      let converted = convertInlineMarkdown(inner)
      
      // 对于 align="center" 的容器：
      // 1. 连续的图片（badges）合并到同一行
      // 2. 每行文本之间插入空行，确保 react-markdown 渲染为独立 <p>
      if (attrs && /align\s*=\s*["']center["']/i.test(attrs)) {
        const lines = converted.split('\n')
        const mergedLines: string[] = []
        let currentLine = ''
        let lastWasImage = false
        let inImageGroup = false
        
        const flushLine = () => {
          if (currentLine) {
            mergedLines.push(currentLine)
            currentLine = ''
          }
        }
        
        for (const line of lines) {
          const trimmed = line.trim()
          
          // 空行：如果不在图片组中，结束当前行并插入分隔
          if (trimmed === '') {
            if (!inImageGroup) {
              flushLine()
              mergedLines.push('')
            }
            continue
          }
          
          // 块级元素（标题等）：结束当前行，单独成行
          const isBlock = /^<(h[1-6]|p|div|ul|ol|blockquote|pre|table|hr|br)(\s|>)/i.test(trimmed)
          
          if (isBlock) {
            flushLine()
            mergedLines.push(trimmed)
            mergedLines.push('')
            lastWasImage = false
            inImageGroup = false
          } else {
            // 匹配纯 <img> 或 <a><img></a>（链接包裹的徽章）
            const isImage = /^<img\s/i.test(trimmed) || /^<a\s[^>]*>\s*<img\s/i.test(trimmed)
            
            if (isImage) {
              if (lastWasImage || inImageGroup) {
                // 连续图片：合并到同一行（badges 水平排列，不插入空行）
                currentLine += ' ' + trimmed
              } else {
                // 第一个图片：结束前面的文本行，开始图片行
                flushLine()
                mergedLines.push('') // 与前面内容分隔
                currentLine = trimmed
              }
              lastWasImage = true
              inImageGroup = true
            } else {
              // 非图片（文本、strong 等）：结束图片组，开始新文本行
              flushLine()
              mergedLines.push('')
              currentLine = trimmed
              lastWasImage = false
              inImageGroup = false
            }
          }
        }
        
        // 处理最后一行（不需要尾部空行）
        if (currentLine) {
          mergedLines.push(currentLine)
        }
        
        // 移除末尾多余空行
        while (mergedLines.length > 0 && mergedLines[mergedLines.length - 1] === '') {
          mergedLines.pop()
        }
        
        converted = mergedLines.join('\n')
        
      }
      
      return `<${tag}${attrs || ''}>${converted}</${tag}>`
    }
  )
  
  return result
}

/**
 * 从 markdown 内容中提取第一个 h1 标题
 * 支持两种格式：
 * - # Heading（原始 markdown）
 * - <h1>Heading</h1>（HTML）
 */
export function extractFirstH1(content: string): string | null {
  if (!content) return null
  
  // 尝试匹配 # Heading（原始 markdown 格式）
  const hashMatch = content.match(/^# (.+)$/m)
  if (hashMatch) return hashMatch[1].trim()
  
  // 尝试匹配 <h1>Heading</h1>（HTML 格式）
  const htmlMatch = content.match(/<h1[^>]*>(.+?)<\/h1>/i)
  if (htmlMatch) return htmlMatch[1].trim()
  
  return null
}
