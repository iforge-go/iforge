import RepoHome from './RepoHome'

// 仓库首页(server component wrapper)
// title 由父 layout 的 generateMetadata 兜底("owner/repo: description")
// 实际页面内容由 RepoHome(client component)渲染
export default function Page() {
  return <RepoHome />
}
