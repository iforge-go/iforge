// Type definitions for iForge API

export interface User {
  userName: string
  fullName: string
  mailAddress: string
  isAdmin: boolean
  isOrganization?: boolean
  image?: string | null
  description?: string | null
  registeredDate: string
  location?: string
  createdAt?: string
}

export interface Collaborator {
  userName: string
  repositoryName: string
  collaboratorName: string
  role: string
  fullName?: string
  avatarUrl?: string | null
  email?: string
  isOrganization?: boolean
}

export interface UserSearchResult {
  userName: string
  fullName: string
  image: string | null
  mailAddress: string
}

export interface Repository {
  userName: string
  repositoryName: string
  isPrivate: boolean
  description: string | null
  defaultBranch: string
  registeredDate: string
  updatedDate: string
  lastActivityDate: string
  originUserName?: string | null
  originRepositoryName?: string | null
  parentUserName?: string | null
  parentRepositoryName?: string | null
  isArchived: boolean
  isTemplate: boolean
  options?: {
    issuesOption: string
    externalIssuesUrl?: string | null
    wikiOption: string
    externalWikiUrl?: string | null
    allowFork: boolean
    allowMerge: boolean
    allowRebase: boolean
    allowRebaseMerge: boolean
    allowSquash: boolean
    mergeOptions: string
    defaultMergeOption: string
    safeMode: boolean
  }
}

export interface Branch {
  name: string
  commitId: string
  isDefault: boolean
}

export interface ProtectedBranchInfo {
  branchName: string
  status: string
}

export interface FileChange {
  filename: string
  status: string
  additions: number
  deletions: number
  patch?: string
}

export interface CommitInfo {
  id: string
  message: string
  author: string
  email: string
  timestamp: string
  files?: FileChange[]
}

export interface FileEntry {
  name: string
  path: string
  type: 'file' | 'dir'
  size: number
  mode: string
  lastCommit?: {
    id: string
    message: string
    author: string
    email: string
    timestamp: string
  }
}

export interface FileContent {
  path: string
  content: string
  encoding: string
  size: number
  isBinary: boolean
}

export interface Issue {
  issueId: number
  title: string
  content: string
  state: string
  closed?: boolean
  locked?: boolean
  userName: string
  repositoryName: string
  openedUserName: string
  openedUserImage?: string | null
  openedUserFullName?: string
  assignees?: User[]
  milestoneId?: number
  labels: Label[]
  commentsCount: number
  participantUserNames?: string[]
  registeredDate: string
  updatedDate: string
  isMergeRequest?: boolean
}

export interface IssueTemplate {
  name: string
  content: string
}

export interface Comment {
  commentId: number
  content: string
  userName: string
  commentedUserName: string
  action?: string
  registeredDate: string
  updatedDate: string
}

export interface Participant {
  userName: string
  fullName: string
  image?: string | null
}

export interface MergeRequest {
  issueId: number
  title: string
  content?: string
  state: string
  userName: string
  branch: string
  requestBranch: string
  requestUserName: string
  requestRepositoryName?: string
  isDraft: boolean
  merged?: boolean
  commitIdFrom?: string
  commitIdTo?: string
  mergedCommitIds?: string
  mergedCommits?: string
  mergedFileChanges?: string
  createdAt: string
  updatedAt?: string
}

export type ReviewStatus = 'approved' | 'changes_requested' | 'commented'

export interface Review {
  reviewId: number
  userName: string
  repositoryName: string
  issueId: number
  reviewer: string
  status: ReviewStatus
  content: string | null
  registeredDate: string
  updatedDate: string
}

export interface ReviewComment {
  commentId: number
  reviewId: number
  userName: string
  repositoryName: string
  issueId: number
  commenter: string
  filePath: string
  line: number
  content: string
  registeredDate: string
  updatedDate: string
}

export interface Notification {
  slug: string
  recipientUserName: string
  repositoryUserName: string
  repositoryName: string
  notificationType: string
  issueId?: number | null
  commentId?: number | null
  projectSlug?: string | null
  taskId?: number | null
  storyId?: number | null
  storySlug?: string | null
  actor: string
  message: string
  read: boolean
  registeredDate: string
}

export interface RepositoryMirror {
  userName: string
  repositoryName: string
  mirrorUrl: string
  syncInterval: number
  lastSyncDate?: string
  nextSyncDate?: string
  enabled: boolean
  syncOnPush: boolean
  authentication: string
  username?: string | null
  password?: string | null
  sshKey?: string | null
}

export interface ParsedRepoURL {
  owner: string
  repo: string
}

export interface Priority {
  userName: string
  repositoryName: string
  priorityId: number
  priorityName: string
  description?: string | null
  color: string
}

export interface CommitStatus {
  userName: string
  repositoryName: string
  commitId: string
  context: string
  state: 'pending' | 'success' | 'failure' | 'error'
  targetUrl?: string | null
  description?: string | null
  updatedDate: string
  creator: string
}

export interface CombinedCommitStatus {
  state: 'pending' | 'success' | 'failure' | 'error'
  statuses: CommitStatus[]
}

export interface CustomField {
  fieldId: number
  userName: string
  repositoryName: string
  fieldName: string
  fieldType: string
  constraints?: string | null
  enableForIssues: boolean
  enableForMergeRequests: boolean
  registeredDate: string
}

export interface FunctionInfo {
  name: string
  file: string
  line: number
  length: number
}

export interface QualityFileInfo {
  path: string
  lines: number
  size: number
}

export interface CodeQualityReport {
  totalLines: number
  totalFiles: number
  languageStats: Record<string, number>
  avgLineLength: number
  maxLineLength: number
  commentRatio: number
  emptyLineRatio: number
  longFunctions: FunctionInfo[]
  largeFiles: QualityFileInfo[]
  complexityScore: number
  maintainability: string
}

export interface Label {
  labelId: number
  labelName: string
  color: string
}

export interface Milestone {
  milestoneId: number
  userName: string
  repositoryName: string
  title: string
  description?: string
  dueDate?: string
  closedDate?: string
}

export interface Tag {
  name: string
  commitId: string
  isRelease: boolean
  createdAt: string
}

export interface Organization {
  userName: string
  fullName: string
  mailAddress: string
  isAdmin: boolean
  url?: string
  registeredDate: string
  updatedDate: string
  lastLoginDate?: string
  image?: string
  isOrganization: boolean
  isRemoved: boolean
  description?: string
}

export interface OrganizationMember {
  organizationName: string
  userName: string
  isManager: boolean
  joinedDate: string
  fullName: string
  image?: string | null
  isAdmin: boolean
}

export interface Activity {
  activityId: string
  userName: string
  repositoryName: string
  activityUserName: string
  activityType: string
  message: string
  additionalInfo?: string | null
  activityDate: string
}

export interface SSHKey {
  sshKeyId: number
  userName: string
  title: string
  publicKey: string
  registeredDate: string
}

export interface GeneralSettings {
  siteName: string
  description: string
  allowRegistration: boolean
  allowAnonymous: boolean
  defaultBranch: string
  timezone: string
}

// Public general settings (no auth required)
export interface PublicGeneralSettings {
  siteName: string
  description: string
  timezone: string
  allowRegistration: boolean
}

export interface SMTPSettings {
  host: string
  port: number
  username: string
  password: string
  from: string
  ssl: boolean
}

export interface LDAPSettings {
  enabled: boolean
  host: string
  port: number
  baseDN: string
  bindDN: string
  bindPassword: string
  userFilter: string
  emailAttr: string
  nameAttr: string
  tls: boolean
}

export interface WebhookSettings {
  blockPrivateAddress: boolean
  whitelist: string
}

export interface UploadSettings {
  maxFileSize: number
  timeout: number
}

export interface RepositorySettings {
  maxDiffFiles: number
  maxDiffLines: number
  httpsUrlTemplate: string
  sshUrlTemplate: string
}

export interface OIDCSettings {
  enabled: boolean
  clientId: string
  clientSecret: string
  authURL: string
  tokenURL: string
  userInfoURL: string
  redirectURL: string
}

export interface AISettings {
  enabled: boolean
  provider: string // openai|deepseek|qwen|zhipu|moonshot|custom
  baseUrl: string
  apiKey: string
  model: string
  timeout: number
  maxTokens: number
}

// AIModelConfig 表示一条 AI 模型配置（支持多条，其中 1 条为默认）。
// 与后端 model.AIModelConfig 对应。
export interface AIModelConfig {
  id: number
  name: string
  provider: string
  baseUrl: string
  apiKey: string // GET 时为脱敏形式（末 4 位 + 前缀 *）
  model: string
  timeout: number
  maxTokens: number
  enabled: boolean
  isDefault: boolean
  testStatus: string // "" | "success" | "failed"
  testedAt: string | null // ISO 时间戳，null 表示从未测试
  createdAt: string
  updatedAt: string
}

// AIModelConfigInput 是创建/更新 AI 模型配置时的请求体。
export interface AIModelConfigInput {
  name: string
  provider: string
  baseUrl: string
  apiKey: string
  model: string
  timeout: number
  maxTokens: number
  enabled: boolean
  isDefault: boolean
}

export interface AITaskSuggestion {
  title: string
  description: string
  priority: 'low' | 'medium' | 'high' | 'urgent'
  taskType: 'task' | 'feature' | 'bug' | 'improvement'
  storyPoints?: number | null
}

export interface AIDecomposeResult {
  tasks: AITaskSuggestion[]
  model: string
  usage?: {
    prompt_tokens?: number
    completion_tokens?: number
    total_tokens?: number
  }
}

// AI 优化用户故事草稿的结果：优化后的标题/描述/验收标准 + 建议的优先级/故事点
export interface AIOptimizeResult {
  title: string
  description: string
  acceptanceCriteria: string
  priority: 'low' | 'medium' | 'high' | 'urgent'
  storyPoints: number // Fibonacci: 1,2,3,5,8,13
  summary: string // AI 对本次优化的简要说明
  usage?: AITokenUsage // 本次对话的 token 用量统计
}

// AI 优化任务草稿的结果：与 AIOptimizeResult 的区别是任务有 taskType 而无 acceptanceCriteria
export interface AIOptimizeTaskResult {
  title: string
  description: string
  priority: 'low' | 'medium' | 'high' | 'urgent'
  taskType: 'task' | 'feature' | 'bug' | 'improvement'
  storyPoints: number // Fibonacci: 1,2,3,5,8,13
  summary: string // AI 对本次优化的简要说明
  usage?: AITokenUsage // 本次对话的 token 用量统计
}

// AI 优化迭代草稿的结果：迭代有 goal 而无 priority/taskType/storyPoints/acceptanceCriteria
export interface AIOptimizeSprintResult {
  title: string
  description: string
  goal: string // 迭代目标（具体、可衡量）
  summary: string // AI 对本次优化的简要说明
  usage?: AITokenUsage // 本次对话的 token 用量统计
}

// AI 对话的 token 用量统计（OpenAI 兼容接口返回的 usage 字段）
export interface AITokenUsage {
  prompt_tokens?: number // 输入 token
  completion_tokens?: number // 输出 token
  total_tokens?: number // 总 token
}

export interface Webhook {
  created_by: string
  created_at: string
  updated_at: string
}

export interface ContributionDay {
  date: string
  count: number
}

export interface TableInfo {
  name: string
  rowCount: number
  sizeBytes: number
  engine: string
  createTime: string
}

export interface ColumnInfo {
  name: string
  type: string
  nullable: boolean
  key: string
  defaultValue: string
  extra: string
}

export interface QueryResult {
  columns: string[]
  rows: any[][]
  count: number
}

export interface Webhook {
  id: number
  userName: string
  repositoryName: string
  url: string
  contentType: string
  active: boolean
  events: string[]
  sslVerify: boolean
  insecureSSL: boolean
  createdAt: string
  updatedAt: string
}

export interface WebhookDelivery {
  id: number
  webhookId: number
  event: string
  action: string
  statusCode: number
  requestBody: string
  responseBody: string
  duration: number
  success: boolean
  errorMessage: string
  createdAt: string
}

export interface SystemInfo {
  userCount: number
  orgCount: number
  repoCount: number
  projectCount: number
  uptime: {
    seconds: number
    human: string
  }
  memory: {
    alloc: number
    totalAlloc: number
    sys: number
    numGC: number
    allocHuman: string
    sysHuman: string
  }
  cpu: {
    numCPU: number
    numGoroutine: number
  }
  os: {
    name: string
    nameHuman: string
    arch: string
    hostname: string
    compiler: string
  }
  services: {
    http: {
      port: number
      address: string
    }
    ssh: {
      enabled: boolean
      port: number
      address: string
      protocol: string
    }
    database: {
      type: string
      path: string
    }
    reposPath: string
    homeDir: string
  }
  goVersion: string
  version: string
  gitCommit: string
  buildTime: string
  dirty: boolean
  repository: string
  serverTime: string
  timezone: string
}

export interface Release {
  userName: string
  repositoryName: string
  tag: string
  name: string
  content: string | null
  registeredDate: string
  author: string
}

export interface ReleaseAsset {
  userName: string
  repositoryName: string
  tag: string
  assetId: number
  fileName: string
  label: string | null
  size: number
  uploader: string
  registeredDate: string
}

export interface AccountPreference {
  userName: string
  highlighterTheme: string
  notification: boolean
  timezone: string
}

export interface ExtraMailAddress {
  userName: string
  mailAddress: string
}

export interface WikiPage {
  pageName: string
  title: string
  content: string
  author: string
  createdAt: string
  updatedAt: string
}

export interface AccessToken {
  id: number
  note: string
  userName: string
  lastUsedAt?: string | null
  createdAt: string
}

export interface DeployKey {
  id: number
  title: string
  publicKey: string
  allowWrite: boolean
  createdAt: string
}

export interface GPGKey {
  userName: string
  keyId: number
  gpgKeyId: string
  title: string
  publicKey: string
  registeredDate: string
}

export interface Contributor {
  name: string
  email: string
  avatarUrl?: string
  commits: number
  additions: number
  deletions: number
  lastCommitAt?: string
}

export interface AccessTokenWithSecret extends AccessToken {
  value: string
}

export interface LFSObject {
  userName: string
  repositoryName: string
  oid: string
  size: number
  createdAt: string
}

// Scrum types
export interface Project {
  ownerName: string
  name: string
  slug: string
  description: string | null
  isPrivate: boolean
  registeredDate: string
  updatedDate: string
}

export interface ProjectMember {
  projectSlug: string
  userName: string
  role: string
  joinedDate: string
}

export interface Sprint {
  projectSlug: string
  title: string
  slug: string
  description: string | null
  status: string
  goal: string | null
  startDate: string | null
  endDate: string | null
  completedDate: string | null
  createdBy: string
  createdAt: string
  updatedAt: string
  // 统计字段(非持久化,由后端 ListSprints 批量计算填充)
  // 口径与 Story 级 Sprint 规划一致:TaskCount 含父 Story 在该 Sprint 的继承 task
  taskCount?: number
  storyCount?: number
  totalStoryPoints?: number
}

// 速度图数据点：一个已关闭 Sprint 的承诺点数 vs 完成点数（对齐 Jira Velocity Chart）
export interface VelocityDataPoint {
  sprintSlug: string
  title: string
  committedPoints: number
  completedPoints: number
  completedDate: string | null
}

// Sprint 范围变更事件（条目加入/移出 Sprint，对齐 Jira Sprint Report scope change）
export interface SprintScopeChange {
  userName: string
  itemType: string // "story" 或 "task"
  itemTitle: string
  action: string // sprint_scope_added / sprint_scope_removed
  createdAt: string
}

// Sprint 报告：承诺 vs 完成统计 + 范围变更时间线
export interface SprintReport {
  committedPoints: number
  completedPoints: number
  scopeChanges: SprintScopeChange[]
}

export interface UserStory {
  projectSlug: string
  sprintSlug: string | null
  title: string
  slug: string
  description: string | null
  status: string
  priority: string
  storyPoints: number | null
  acceptanceCriteria: string | null
  assigneeName: string | null
  reporterName: string
  // Epic 关联(可空):Backlog 中游离 Story 不属于任何 Epic
  epicSlug: string | null
  epicTitle: string | null
  // 关联任务数(非持久化,后端 fillTaskCountBatch 批量填充),Backlog 故事行显示任务数徽标
  taskCount?: number
  createdAt: string
  updatedAt: string
  closedAt: string | null
}

// Epic 表示跨多个 Sprint 的大型需求(Epic → Story → Task 三层最顶层)
// Status 懒推导:StatusManual=false 时由后端实时聚合 Story 状态计算
export interface Epic {
  projectSlug: string
  title: string
  slug: string
  description: string | null
  status: string // open, in_progress, done, closed
  statusManual: boolean // true=手动设置,false=自动推导
  priority: string // low, medium, high, urgent
  goal: string | null
  startDate: string | null
  targetDate: string | null
  ownerName: string | null
  reporterName: string
  createdAt: string
  updatedAt: string
  closedAt: string | null
  // 聚合字段(非持久化,后端 service 填充)
  totalStories: number
  doneStories: number
  totalStoryPoints: number
  doneStoryPoints: number
  progressPercent: number // DoneStories/TotalStories*100,无 Story 时 0
  sprintCount: number // 跨多少个 Sprint
}

// EpicProgress 表示 Epic 的进度概要(独立接口 /epics/:slug/progress)
export interface EpicProgress {
  totalStories: number
  doneStories: number
  totalStoryPoints: number
  doneStoryPoints: number
  progressPercent: number
  sprintCount: number
}

// EpicSprintInfo 表示 Epic 关联到的 Sprint 概要(经 task → story → epic 聚合)
export interface EpicSprintInfo {
  sprintId: number
  sprintSlug: string
  title: string
  status: string
  taskCount: number
  startDate: string | null
  endDate: string | null
}

export interface Task {
  projectSlug: string
  taskId: number
  sprintSlug: string | null
  userStorySlug: string | null
  title: string
  slug: string
  description: string | null
  status: string
  priority: string
  taskType: string
  storyPoints: number | null
  estimatedHours: number | null
  assigneeName: string | null
  reporterName: string
  parentId: number | null
  rootId: number | null
  position: number
  dueDate: string | null
  subtaskCount?: number
  subtaskCompletedCount?: number
  createdAt: string
  updatedAt: string
  closedAt: string | null
  closedByName: string | null
  // Sideload: assignees batch-loaded for list responses
  assignees?: { userName: string; fullName?: string; image?: string | null }[]
  // Sideload: Epic 关联信息(通过 user_story 间接关联),用于看板卡片显示 Epic 彩色标签
  // 后端 ListTasks handler 批量查询填充,无关联 Epic 时为 null
  epicSlug?: string | null
  epicTitle?: string | null
  // Sideload: 关联 UserStory 标题(userStorySlug 已在上方),后端 ListTasks 批量填充,看板卡片显示故事 Badge
  userStoryTitle?: string | null
  // Sideload: 未完成前置依赖数,后端 ListTasks/GetTask 填充,>0 时看板卡片显示阻塞标记
  blockedByCount?: number
  // Sideload: 阻塞当前任务的未完成前置依赖列表,后端 GetTask 填充,供 TaskDrawer 展示
  blockingTasks?: BlockedTaskInfo[]
  // Sideload: 工时记录,后端 GetTask 填充,供 TaskDrawer 展示
  workLogs?: TaskWorkLog[]
  // Sideload: 实际工时(小时),后端 GetTask 填充
  actualHours?: number
}

// TaskStatus 项目可配置的任务状态(看板列),对齐 Jira status
export interface TaskStatus {
  id: number
  projectSlug: string
  name: string
  slug: string
  color: string
  position: number
  isDefault: boolean
  isClosed: boolean
  // 状态三态分类:todo / in_progress / done(对齐 Jira status category,驱动报表口径与看板列分组)
  category: string
  // 看板列在制品上限(对齐 Jira Kanban WIP);null=不限制,>0 时列头显示 N/Limit 超限标红
  wipLimit: number | null
  createdAt: string
  updatedAt: string
}

// TaskStatusTransition 项目配置的允许状态流转规则(工作流约束,对齐 Jira workflow transitions)
// 空规则集=全允许(向后兼容);非空时任务状态从 fromSlug→toSlug 必须命中某条
export interface TaskStatusTransition {
  projectSlug: string
  fromStatusId: number
  toStatusId: number
  fromSlug: string
  toSlug: string
}

export interface TaskStatusHistory {
  id: number
  projectSlug: string
  taskId: number
  oldStatus: string
  newStatus: string
  changedByName: string
  comment: string | null
  createdAt: string
}

export interface TaskComment {
  commentId: number
  projectSlug: string
  taskId: number
  authorName: string
  content: string
  createdAt: string
  updatedAt: string
}

export interface TaskWorkLog {
  logId: number
  projectSlug?: string
  taskId: number
  userName: string
  hours: number
  description: string
  createdAt: string
  updatedAt: string
}

export interface TaskLabel {
  id: number
  projectSlug: string
  name: string
  color: string
}

// TaskBranch 表示任务与 Git 分支的松耦合关联（仅记录分支物理位置，不引用 VCS 表）
export interface TaskBranch {
  id: number
  projectSlug: string
  taskId: number
  repoFullName: string // "owner/repo" 格式
  branchName: string
  userName: string
  createdAt: string
}

// TaskCommit 表示任务关联分支上的一次提交（与后端 git.PushCommitInfo 对应）
export interface TaskCommit {
  id: string
  message: string
  author: string
  email: string
  time: string
}

// BranchCommits 表示某条关联分支上独有的提交列表（相对默认分支的 ahead commits）
export interface BranchCommits {
  repoFullName: string
  branchName: string
  commits: TaskCommit[]
}

// DependencyTask 表示依赖关系中的一端任务概要(供前端展示 id/title/status/type)
export interface DependencyTask {
  taskId: number
  title: string
  status: string
  dependencyType?: string
}

// BlockedTaskInfo 描述一个阻塞当前任务的前置任务(未完成)
export interface BlockedTaskInfo {
  taskId: number
  title: string
  status: string
  dependencyType?: string
}

export interface ScrumActivity {
  slug?: string
  userName: string
  entityType: string // task, story, sprint
  entityId: number
  action: string // created, updated, status_changed, deleted, commit_pushed
  field: string // status, priority, assignee, etc.
  oldValue: string
  newValue: string
  createdAt: string
}

// AuditLog 记录安全相关的管理操作（管理员可见）
export interface AuditLog {
  id: number
  actorUserName: string
  actorIP: string
  action: string // e.g., "repository.delete"
  resourceType: string // e.g., "repository"
  resourceId: string // e.g., "owner/repo"
  detail: string // JSON string
  success: boolean
  createdAt: string
}

export interface AuditLogListResponse {
  logs: AuditLog[]
  total: number
  page: number
  pageSize: number
}

// CodeSearchResult 表示仓库内代码搜索的一个匹配文件
// matches 字段为字符串数组，格式 "L<行号>: <行内容>"
export interface CodeSearchResult {
  path: string
  matches: string[]
}

// CodeSearchResponse 是搜索 API 的响应包装
// truncated=true 表示结果被截断（达到 200 文件上限或 5 秒超时）
export interface CodeSearchResponse {
  results: CodeSearchResult[]
  truncated: boolean
}

// ============================================================================
// CI/CD types (路径 C：内置 Pipeline/Job/Runner/Deployment 全链路)
// ============================================================================

// Pipeline 表示一次 CI/CD 运行
export interface Pipeline {
  id: number
  userName: string
  repositoryName: string
  triggerEvent: 'push' | 'merge_request' | 'manual' | 'cron' | 'external'
  ref: string
  commitSha: string
  status: 'pending' | 'running' | 'success' | 'failed' | 'canceled'
  yamlConfig: string
  triggeredBy: string
  mergeRequestId?: number | null
  createdAt: string
  startedAt?: string | null
  finishedAt?: string | null
  duration?: number | null // 毫秒
  message: string
}

// Job 表示 Pipeline 内的一个具体任务
export interface CICDJob {
  id: number
  pipelineId: number
  stageName: string
  jobName: string
  status: 'pending' | 'running' | 'success' | 'failed' | 'canceled' | 'skipped'
  runnerId?: number | null
  image: string
  script: string
  needs: string
  vars: string
  onlyRefs: string
  exceptRefs: string
  when: 'on_success' | 'manual' | 'always' | 'never'
  environment?: string | null
  artifacts: string
  cache: string
  startedAt?: string | null
  finishedAt?: string | null
  exitCode?: number | null
  failureReason?: string | null
  createdAt: string
}

// JobLog 表示 Job 的日志（按行存储）
export interface JobLog {
  id: number
  jobId: number
  lineNo: number
  content: string
  timestamp: string
  stream: 'stdout' | 'stderr'
}

// Artifact 表示 Job 的构建产物
export interface Artifact {
  id: number
  jobId: number
  name: string
  path: string
  size: number
  storagePath: string
  expiresAt?: string | null
  createdAt: string
}

// Secret 表示仓库级加密变量（列表时不返回 value）
export interface Secret {
  id: number
  userName: string
  repositoryName: string
  key: string
  createdBy: string
  createdAt: string
  updatedAt: string
}

// Environment 表示部署环境
export interface Environment {
  id: number
  userName: string
  repositoryName: string
  name: string
  url?: string | null
  deploymentPolicy: 'manual' | 'auto'
  protectionRules: string
  createdAt: string
  updatedAt: string
}

// Deployment 表示一次部署记录
export interface Deployment {
  id: number
  environmentId: number
  userName: string
  repositoryName: string
  pipelineId?: number | null
  jobId?: number | null
  commitSha: string
  status: 'in_progress' | 'success' | 'failed' | 'canceled'
  deployedBy: string
  deployedAt: string
  finishedAt?: string | null
  rollbackOfId?: number | null
  note?: string | null
}

// Runner 表示执行器
export interface Runner {
  id: number
  name: string
  status: 'online' | 'offline' | 'busy'
  type: 'shell' | 'docker' | 'kubernetes'
  tags: string
  lastHeartbeat?: string | null
  version: string
  ipAddress: string
  description: string
  isBuiltin: boolean
  maxJobs: number
  createdAt: string
  userName?: string | null
  repositoryName?: string | null
}

// CronSchedule 表示仓库的定时触发配置
// 5 字段 cron 表达式(分 时 日 月 周),支持 *, */N, N, N-M, N,M,K, N-M/step
// day-of-month 与 day-of-week 取 AND(更严格,符合 CI/CD 工具行为)
export interface CronSchedule {
  id: number
  userName: string
  repositoryName: string
  name: string
  schedule: string
  branch: string
  yamlConfig?: string
  enabled: boolean
  creator: string
  lastRunAt?: string | null
  nextRunAt?: string | null
  createdAt: string
  updatedAt: string
}
