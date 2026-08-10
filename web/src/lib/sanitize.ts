'use client'

import DOMPurify from 'dompurify'

// 允许常见的安全 HTML 标签和属性，移除 script/事件处理器/javascript: 协议
const config = {
  ALLOWED_TAGS: [
    'a', 'b', 'i', 'em', 'strong', 'u', 's', 'del', 'mark',
    'p', 'br', 'hr', 'blockquote', 'pre', 'code',
    'h1', 'h2', 'h3', 'h4', 'h5', 'h6',
    'ul', 'ol', 'li',
    'table', 'thead', 'tbody', 'tr', 'th', 'td',
    'img', 'span', 'div', 'details', 'summary',
  ],
  ALLOWED_ATTR: ['href', 'src', 'alt', 'title', 'class', 'target', 'rel', 'width', 'height', 'colspan', 'rowspan'],
  ALLOW_DATA_ATTR: false,
  // 仅允许 http/https/mailto 协议，禁止 javascript: data: 等
  ALLOWED_URI_REGEXP: /^(?:(?:https?|mailto):)/i,
}

export function sanitizeHtml(dirty: string): string {
  if (!dirty) return ''
  return DOMPurify.sanitize(dirty, config) as string
}
