export interface DiffHunkLine {
  type: 'add' | 'delete' | 'context'
  content: string
  oldLine?: number
  newLine?: number
}

export interface DiffHunk {
  oldStart: number
  newStart: number
  lines: DiffHunkLine[]
}

export interface FileDiff {
  filename: string
  status: 'added' | 'deleted' | 'modified'
  additions: number
  deletions: number
  hunks: DiffHunk[]
}

export function parsePatch(filename: string, status: string, patch: string): FileDiff {
  const hunks: DiffHunk[] = []
  let additions = 0
  let deletions = 0

  if (!patch) {
    return { filename, status: status as any, additions: 0, deletions: 0, hunks: [] }
  }

  const lines = patch.split('\n')
  let currentHunk: DiffHunk | null = null
  let oldLine = 0
  let newLine = 0

  for (const line of lines) {
    if (line.startsWith('@@')) {
      const match = line.match(/@@ -(\d+)(?:,\d+)? \+(\d+)(?:,\d+)? @@/)
      if (match) {
        oldLine = parseInt(match[1])
        newLine = parseInt(match[2])
        currentHunk = { oldStart: oldLine, newStart: newLine, lines: [] }
        hunks.push(currentHunk)
      }
      continue
    }

    if (!currentHunk) continue

    if (line.startsWith('+')) {
      currentHunk.lines.push({ type: 'add', content: line.substring(1), newLine: newLine++ })
      additions++
    } else if (line.startsWith('-')) {
      currentHunk.lines.push({ type: 'delete', content: line.substring(1), oldLine: oldLine++ })
      deletions++
    } else if (line.startsWith(' ')) {
      currentHunk.lines.push({ type: 'context', content: line.substring(1), oldLine: oldLine++, newLine: newLine++ })
    }
  }

  return { filename, status: status as any, additions, deletions, hunks }
}

interface SplitLine {
  oldLine?: number
  newLine?: number
  oldContent?: string
  newContent?: string
  type: 'add' | 'delete' | 'context' | 'both'
}

export function toSplitLines(hunks: DiffHunk[]): SplitLine[] {
  const result: SplitLine[] = []

  for (const hunk of hunks) {
    let i = 0
    while (i < hunk.lines.length) {
      const line = hunk.lines[i]

      if (line.type === 'context') {
        result.push({ oldLine: line.oldLine, newLine: line.newLine, oldContent: line.content, newContent: line.content, type: 'both' })
        i++
      } else if (line.type === 'delete') {
        // Collect consecutive deletes and adds
        const deletes: DiffHunkLine[] = []
        const adds: DiffHunkLine[] = []
        while (i < hunk.lines.length && hunk.lines[i].type === 'delete') {
          deletes.push(hunk.lines[i])
          i++
        }
        while (i < hunk.lines.length && hunk.lines[i].type === 'add') {
          adds.push(hunk.lines[i])
          i++
        }
        const maxLen = Math.max(deletes.length, adds.length)
        for (let j = 0; j < maxLen; j++) {
          result.push({
            oldLine: deletes[j]?.oldLine,
            newLine: adds[j]?.newLine,
            oldContent: deletes[j]?.content,
            newContent: adds[j]?.content,
            type: deletes[j] && adds[j] ? 'both' : deletes[j] ? 'delete' : 'add',
          })
        }
      } else if (line.type === 'add') {
        result.push({ newLine: line.newLine, newContent: line.content, type: 'add' })
        i++
      }
    }
  }

  return result
}
