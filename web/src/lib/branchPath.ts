import { Branch } from '@/lib/types'

/**
 * 从 catch-all 路由的 pathSegments 中解析出分支名（ref）和文件路径。
 *
 * 背景：分支名可能含 "/"（如 feature/task-15），而 Next.js catch-all 路由
 * 会把 "/tree/feature/task-15/src" 拆成 ['feature', 'task-15', 'src']，
 * 无法从字面区分哪些段属于分支名、哪些属于文件路径。
 *
 * 方案：用已知分支列表反向匹配 pathSegments 的前缀，取最长匹配作为分支名，
 * 剩余段作为文件路径。这样既能正确识别含 "/" 的分支，又能把后续段当作路径。
 *
 * @param pathSegments  catch-all 路由参数（如 useParams().path）
 * @param branches      已知分支列表（来自 useRepo()）
 * @param defaultBranch 默认分支名（当 pathSegments 为空时使用）
 * @returns { ref, subPath }  分支名和剩余路径（dirPath / filePath）
 */
export function parseBranchAndPath(
  pathSegments: string[] | undefined,
  branches: Branch[],
  defaultBranch: string,
): { ref: string; subPath: string } {
  // Next.js useParams() 返回的 catch-all 路径段是 URL 编码的（如 %E6%88%91%E7%9A%84），
  // 需要先解码，否则后续 encodeURIComponent 会双重编码导致后端无法匹配文件
  const segs = (pathSegments || []).map(s => {
    try { return decodeURIComponent(s) } catch { return s }
  })
  // 用已知分支列表反向匹配前缀，取最长匹配（处理 feature 与 feature/task-15 共存的情况）
  const matchingBranchSegs = branches
    .map(b => b.name.split('/'))
    .filter(bs => bs.length <= segs.length && bs.every((s, i) => s === segs[i]))
    .sort((a, b) => b.length - a.length)[0]

  if (matchingBranchSegs) {
    return {
      ref: matchingBranchSegs.join('/'),
      subPath: segs.slice(matchingBranchSegs.length).join('/'),
    }
  }
  // 回退：无匹配时按旧逻辑（分支名不含 "/" 或 branches 尚未加载）
  return {
    ref: segs[0] || defaultBranch,
    subPath: segs.slice(1).join('/'),
  }
}

/**
 * 判断是否需要等待 branches 加载完成。
 *
 * 当 pathSegments 有多段时，可能含 "/" 分支名，需要 branches 数据才能正确反向匹配。
 * 此时应等 branches 加载完再发请求，避免用错误的 ref 请求导致空目录/空内容闪烁。
 *
 * @returns true 表示应等待（branches 尚未加载且路径可能含 "/" 分支）
 */
export function shouldWaitForBranches(
  pathSegments: string[] | undefined,
  branches: Branch[],
): boolean {
  const segs = pathSegments || []
  return segs.length > 1 && branches.length === 0
}
