import { Project, Sprint, UserStory, Task, TaskComment, TaskWorkLog, TaskLabel, ScrumActivity, TaskStatusHistory, TaskBranch, BranchCommits, DependencyTask, AIDecomposeResult, AIOptimizeResult, AIOptimizeTaskResult, AIOptimizeSprintResult, Participant, Epic, EpicProgress, EpicSprintInfo, TaskStatus, TaskStatusTransition, VelocityDataPoint, SprintReport, ProjectMember } from '@/lib/types'
import { ApiError } from '@/lib/errorMessages'

// TaskAssignee 任务指派人信息
interface TaskAssignee {
  userName: string
  fullName?: string
  image?: string | null
}

type RequestFn = <T>(endpoint: string, options?: { method?: string; body?: unknown }) => Promise<T>

// Sideload 任务列表响应:tasks 仅携带 assigneeUserNames(引用),assignees 为去重后的用户数组。
// total 为满足过滤条件的总数(后端分页返回,前端据此计算总页数)。
type SideloadTasksResponse = {
  tasks: (Task & { assigneeUserNames?: string[] })[]
  assignees: { userName: string; fullName?: string; image?: string | null }[]
  total?: number
}

// mapTasksResponse 将 sideload 响应映射为带 assignees 对象的 Task[],供 getTasks / getTasksPaged 复用。
function mapTasksResponse(res: SideloadTasksResponse | Task[]): Task[] {
  if (res && Array.isArray((res as SideloadTasksResponse).tasks)) {
    const r = res as SideloadTasksResponse
    const userMap = new Map<string, TaskAssignee>()
    for (const a of r.assignees || []) {
      if (a?.userName) userMap.set(a.userName, a)
    }
    return r.tasks.map((t) => ({
      ...t,
      assignees: (t.assigneeUserNames || [])
        .map((name) => userMap.get(name))
        .filter((a): a is TaskAssignee => a !== undefined),
    }))
  }
  // 向后兼容:响应直接是数组(旧后端)
  return res as Task[]
}

export function createScrumApi(request: RequestFn, baseUrl: string) {
  return {
    // Projects
    getProjects: () => request<Project[]>('/projects'),
    getProject: (slug: string) => request<Project>(`/projects/${slug}`),
    createProject: (data: { name: string; description?: string; isPrivate: boolean }) =>
      request<Project>('/projects', { method: 'POST', body: data }),
    updateProject: (slug: string, data: Partial<Project>) =>
      request<Project>(`/projects/${slug}`, { method: 'PUT', body: data }),
    deleteProject: (slug: string) => request<void>(`/projects/${slug}`, { method: 'DELETE' }),

    // Project Members
    getProjectMembers: (projectSlug: string) => request<ProjectMember[]>(`/projects/${projectSlug}/members`),
    addProjectMember: (projectSlug: string, data: { userName: string; role: string }) =>
      request<void>(`/projects/${projectSlug}/members`, { method: 'POST', body: data }),
    updateProjectMemberRole: (projectSlug: string, userName: string, role: string) =>
      request<void>(`/projects/${projectSlug}/members/${userName}`, { method: 'PUT', body: { role } }),
    removeProjectMember: (projectSlug: string, userName: string) =>
      request<void>(`/projects/${projectSlug}/members/${userName}`, { method: 'DELETE' }),

    // Project Repositories (link VCS repos to Scrum projects)
    getProjectRepositories: (projectSlug: string) =>
      request<{ userName: string; repositoryName: string; projectSlug?: string }[]>(`/projects/${projectSlug}/repositories`),
    addProjectRepository: (projectSlug: string, data: { userName: string; repositoryName: string }) =>
      request<{ userName: string; repositoryName: string; projectSlug: string }>(`/projects/${projectSlug}/repositories`, { method: 'POST', body: data }),
    removeProjectRepository: (projectSlug: string, userName: string, repositoryName: string) =>
      request<void>(`/projects/${projectSlug}/repositories/${userName}/${repositoryName}`, { method: 'DELETE' }),

    // Sprints
    getSprints: (projectSlug: string) => request<Sprint[]>(`/projects/${projectSlug}/sprints`),
    getSprint: (slug: string) => request<Sprint>(`/sprints/${slug}`),
    createSprint: (projectSlug: string, data: {
      title: string
      description?: string
      status: string
      goal?: string
      startDate?: string
      endDate?: string
    }) => request<Sprint>(`/projects/${projectSlug}/sprints`, { method: 'POST', body: data }),
    updateSprint: (slug: string, data: Partial<Sprint>) =>
      request<Sprint>(`/sprints/${slug}`, { method: 'PUT', body: data }),
    deleteSprint: (slug: string) => request<void>(`/sprints/${slug}`, { method: 'DELETE' }),
    getSprintTasks: (sprintSlug: string) => request<Task[]>(`/sprints/${sprintSlug}/tasks`),
    getSprintStories: (sprintSlug: string) => request<UserStory[]>(`/sprints/${sprintSlug}/stories`),

    // User Stories
    getUserStories: (projectSlug: string, params?: { status?: string; sprintSlug?: string; backlog?: boolean; epicSlug?: string }) => {
      const query = new URLSearchParams()
      if (params?.status) query.set('status', params.status)
      if (params?.sprintSlug) query.set('sprintSlug', params.sprintSlug)
      if (params?.backlog) query.set('backlog', 'true')
      if (params?.epicSlug) query.set('epicSlug', params.epicSlug)
      const qs = query.toString()
      return request<UserStory[]>(`/projects/${projectSlug}/user-stories${qs ? `?${qs}` : ''}`)
    },
    getUserStory: (slug: string) => request<UserStory>(`/user-stories/${slug}`),
    createUserStory: (projectSlug: string, data: {
      title: string
      description?: string
      status: string
      priority: string
      storyPoints?: number
      acceptanceCriteria?: string
      assigneeName?: string
      reporterName?: string
      sprintSlug?: string
      epicSlug?: string
    }) => request<UserStory>(`/projects/${projectSlug}/user-stories`, { method: 'POST', body: data }),
    updateUserStory: (slug: string, data: Partial<UserStory> & { epicSlug?: string | null }) =>
      request<UserStory>(`/user-stories/${slug}`, { method: 'PUT', body: data }),
    deleteUserStory: (slug: string) => request<void>(`/user-stories/${slug}`, { method: 'DELETE' }),
    getUserStoryTasks: (userStorySlug: string) => request<Task[]>(`/user-stories/${userStorySlug}/tasks`),

    // Epics
    getEpics: (projectSlug: string, params?: { status?: string }) => {
      const query = new URLSearchParams()
      if (params?.status) query.set('status', params.status)
      const qs = query.toString()
      return request<Epic[]>(`/projects/${projectSlug}/epics${qs ? `?${qs}` : ''}`)
    },
    getEpic: (slug: string) => request<Epic>(`/epics/${slug}`),
    createEpic: (projectSlug: string, data: {
      title: string
      description?: string
      status?: string
      priority?: string
      goal?: string
      startDate?: string
      targetDate?: string
      ownerName?: string
    }) => request<Epic>(`/projects/${projectSlug}/epics`, { method: 'POST', body: data }),
    updateEpic: (slug: string, data: Partial<Epic> & { startDate?: string | null; targetDate?: string | null }) =>
      request<Epic>(`/epics/${slug}`, { method: 'PUT', body: data }),
    deleteEpic: (slug: string) => request<void>(`/epics/${slug}`, { method: 'DELETE' }),
    getEpicStories: (slug: string) => request<UserStory[]>(`/epics/${slug}/stories`),
    getEpicProgress: (slug: string) => request<EpicProgress>(`/epics/${slug}/progress`),
    getEpicSprints: (slug: string) => request<EpicSprintInfo[]>(`/epics/${slug}/sprints`),
    assignStoryToEpic: (slug: string, storySlug: string) =>
      request<void>(`/epics/${slug}/stories`, { method: 'POST', body: { storySlug } }),
    removeStoryFromEpic: (slug: string, storySlug: string) =>
      request<void>(`/epics/${slug}/stories/${storySlug}`, { method: 'DELETE' }),

    // Tasks
    getTasks: (projectSlug: string, params?: {
      status?: string
      taskType?: string
      sprintSlug?: string
      userStorySlug?: string
      q?: string
      limit?: number
      offset?: number
    }) => {
      const query = new URLSearchParams()
      if (params?.status) query.set('status', params.status)
      if (params?.taskType) query.set('taskType', params.taskType)
      if (params?.sprintSlug) query.set('sprint_slug', params.sprintSlug)
      if (params?.userStorySlug) query.set('user_story_slug', params.userStorySlug)
      if (params?.q) query.set('q', params.q)
      if (params?.limit !== undefined) query.set('limit', String(params.limit))
      if (params?.offset !== undefined) query.set('offset', String(params.offset))
      const qs = query.toString()
      // Sideload 响应映射见 mapTasksResponse(复用给 getTasksPaged)
      return request<SideloadTasksResponse>(`/projects/${projectSlug}/tasks${qs ? `?${qs}` : ''}`).then(mapTasksResponse)
    },
    // 后端分页版:返回 { tasks, total },total 为满足过滤条件的总数(前端据此计算总页数)。
    // 仅任务列表页使用;看板 / Sprint 详情 / Story 详情等仍用 getTasks 获取全量。
    getTasksPaged: (projectSlug: string, params: {
      status?: string
      taskType?: string
      sprintSlug?: string
      userStorySlug?: string
      q?: string
      limit: number
      offset: number
    }) => {
      const query = new URLSearchParams()
      if (params.status) query.set('status', params.status)
      if (params.taskType) query.set('taskType', params.taskType)
      if (params.sprintSlug) query.set('sprint_slug', params.sprintSlug)
      if (params.userStorySlug) query.set('user_story_slug', params.userStorySlug)
      if (params.q) query.set('q', params.q)
      query.set('limit', String(params.limit))
      query.set('offset', String(params.offset))
      const qs = query.toString()
      return request<SideloadTasksResponse>(`/projects/${projectSlug}/tasks?${qs}`).then((res) => ({
        tasks: mapTasksResponse(res),
        total: (res as SideloadTasksResponse)?.total ?? 0,
      }))
    },
    getTask: (projectSlug: string, taskId: number) => request<{
      task: Task
      assignees: { userName: string; fullName?: string; image?: string | null }[]
      subtaskCount: number
      subtaskCompletedCount: number
    }>(`/projects/${projectSlug}/tasks/${taskId}`),
    createTask: (projectSlug: string, data: {
      title: string
      description?: string
      status: string
      priority: string
      taskType: string
      storyPoints?: number
      estimatedHours?: number
      assigneeName?: string
      reporterName?: string
      sprintSlug?: string
      userStorySlug?: string
      parentId?: number
      position?: number
      dueDate?: string
    }) => request<Task>(`/projects/${projectSlug}/tasks`, { method: 'POST', body: data }),
    updateTask: (projectSlug: string, taskId: number, data: Omit<Partial<Task>, 'assignees'> & { dueDate?: string | null; assignees?: string[] }) =>
      request<{ task: Task; assignees: TaskAssignee[] }>(`/projects/${projectSlug}/tasks/${taskId}`, { method: 'PUT', body: data }),
    deleteTask: (projectSlug: string, taskId: number) => request<void>(`/projects/${projectSlug}/tasks/${taskId}`, { method: 'DELETE' }),
    getTaskStatusHistory: (projectSlug: string, taskId: number) =>
      request<TaskStatusHistory[]>(`/projects/${projectSlug}/tasks/${taskId}/status-history`),
    updateTaskPosition: (projectSlug: string, taskId: number, position: number) =>
      request<void>(`/projects/${projectSlug}/tasks/${taskId}/position`, { method: 'PUT', body: { position } }),

    // Task Comments
    getTaskComments: (projectSlug: string, taskId: number) => request<TaskComment[]>(`/projects/${projectSlug}/tasks/${taskId}/comments`),
    createTaskComment: (projectSlug: string, taskId: number, content: string) =>
      request<TaskComment>(`/projects/${projectSlug}/tasks/${taskId}/comments`, { method: 'POST', body: { content } }),
    updateTaskComment: (commentId: number, content: string) =>
      request<TaskComment>(`/comments/${commentId}`, { method: 'PUT', body: { content } }),
    deleteTaskComment: (commentId: number) =>
      request<void>(`/comments/${commentId}`, { method: 'DELETE' }),

    // Task Work Logs (Time Tracking)
    getWorkLogs: (projectSlug: string, taskId: number) =>
      request<{ workLogs: TaskWorkLog[]; actualHours: number }>(`/projects/${projectSlug}/tasks/${taskId}/work-logs`),
    createWorkLog: (projectSlug: string, taskId: number, data: { hours: number; description?: string }) =>
      request<TaskWorkLog>(`/projects/${projectSlug}/tasks/${taskId}/work-logs`, { method: 'POST', body: data }),
    updateWorkLog: (logId: number, data: { hours: number; description?: string }) =>
      request<TaskWorkLog>(`/work-logs/${logId}`, { method: 'PUT', body: data }),
    deleteWorkLog: (logId: number) =>
      request<void>(`/work-logs/${logId}`, { method: 'DELETE' }),

    // Task Labels
    getProjectLabels: (projectSlug: string) => request<TaskLabel[]>(`/projects/${projectSlug}/labels`),
    createProjectLabel: (projectSlug: string, data: { name: string; color: string }) =>
      request<TaskLabel>(`/projects/${projectSlug}/labels`, { method: 'POST', body: data }),
    getTaskLabels: (projectSlug: string, taskId: number) => request<TaskLabel[]>(`/projects/${projectSlug}/tasks/${taskId}/labels`),
    addTaskLabel: (projectSlug: string, taskId: number, labelId: number) =>
      request<void>(`/projects/${projectSlug}/tasks/${taskId}/labels`, { method: 'POST', body: { labelId } }),
    removeTaskLabel: (projectSlug: string, taskId: number, labelId: number) =>
      request<void>(`/projects/${projectSlug}/tasks/${taskId}/labels/${labelId}`, { method: 'DELETE' }),
    getSubtasks: (projectSlug: string, taskId: number) => request<Task[]>(`/projects/${projectSlug}/tasks/${taskId}/subtasks`),
    createSubtask: (projectSlug: string, parentId: number, data: { title: string; description?: string; priority?: string; assigneeName?: string }) =>
      request<Task>(`/projects/${projectSlug}/tasks`, { method: 'POST', body: { ...data, status: 'todo', taskType: 'task', parentId } }),

    // Task Assignees
    getTaskAssignees: (projectSlug: string, taskId: number) => request<{ assignees: TaskAssignee[] }>(`/projects/${projectSlug}/tasks/${taskId}/assignees`),
    addTaskAssignee: (projectSlug: string, taskId: number, username: string) =>
      request<{ assignees: TaskAssignee[] }>(`/projects/${projectSlug}/tasks/${taskId}/assignees/${username}`, { method: 'POST' }),
    removeTaskAssignee: (projectSlug: string, taskId: number, username: string) =>
      request<{ assignees: TaskAssignee[] }>(`/projects/${projectSlug}/tasks/${taskId}/assignees/${username}`, { method: 'DELETE' }),

    // Task Branches (松耦合关联：仅记录分支物理位置，不引用 VCS 表)
    getTaskBranches: (projectSlug: string, taskId: number) =>
      request<{ branches: TaskBranch[] }>(`/projects/${projectSlug}/tasks/${taskId}/branches`),
    associateTaskBranch: (projectSlug: string, taskId: number, data: { repoFullName: string; branchName: string }) =>
      request<TaskBranch>(`/projects/${projectSlug}/tasks/${taskId}/branches`, { method: 'POST', body: data }),
    createAndAssociateTaskBranch: (projectSlug: string, taskId: number, data: { repoFullName: string; branchName: string; from?: string }) =>
      request<TaskBranch>(`/projects/${projectSlug}/tasks/${taskId}/branches/create`, { method: 'POST', body: data }),
    removeTaskBranch: (projectSlug: string, taskId: number, branchId: number) =>
      request<void>(`/projects/${projectSlug}/tasks/${taskId}/branches/${branchId}`, { method: 'DELETE' }),

    // Task Dependencies (硬阻塞:前置依赖未完成时拒绝推进到进行中态)
    getTaskDependencies: (projectSlug: string, taskId: number) =>
      request<{ dependencies: DependencyTask[] }>(`/projects/${projectSlug}/tasks/${taskId}/dependencies`),
    addTaskDependency: (projectSlug: string, taskId: number, dependsOnTaskId: number, dependencyType: string = 'fs') =>
      request<{ taskId: number; dependsOnTaskId: number; dependencyType: string }>(`/projects/${projectSlug}/tasks/${taskId}/dependencies`, {
        method: 'POST',
        body: { dependsOnTaskId, dependencyType },
      }),
    removeTaskDependency: (projectSlug: string, taskId: number, dependsOnTaskId: number) =>
      request<void>(`/projects/${projectSlug}/tasks/${taskId}/dependencies/${dependsOnTaskId}`, { method: 'DELETE' }),
    getTaskBlocks: (projectSlug: string, taskId: number) =>
      request<{ blocks: DependencyTask[] }>(`/projects/${projectSlug}/tasks/${taskId}/blocks`),

    // Task Commits (关联分支的 commit log，对比默认分支取 ahead commits)
    getTaskCommits: (projectSlug: string, taskId: number) =>
      request<{ branches: BranchCommits[] }>(`/projects/${projectSlug}/tasks/${taskId}/commits`),

    // Task Statuses (Kanban columns)
    getTaskStatuses: (projectSlug: string) =>
      request<TaskStatus[]>(`/projects/${projectSlug}/task-statuses`),
    createTaskStatus: (projectSlug: string, data: {
      name: string
      slug: string
      color?: string
      position?: number
      isClosed?: boolean
      // 状态三态分类:todo / in_progress / done(对齐 Jira status category)
      category?: string
      // 看板列在制品上限;null/0=不限制(对齐 Jira Kanban WIP)
      wipLimit?: number | null
    }) => request<TaskStatus>(`/projects/${projectSlug}/task-statuses`, { method: 'POST', body: data }),
    updateTaskStatus: (statusId: number, data: {
      name?: string
      slug?: string
      color?: string
      position?: number
      isClosed?: boolean
      category?: string
      // nil=不更新;非 nil 含 0=更新(0 表示不限制)
      wipLimit?: number | null
    }) => request<TaskStatus>(`/task-statuses/${statusId}`, { method: 'PUT', body: data }),
    deleteTaskStatus: (statusId: number) =>
      request<void>(`/task-statuses/${statusId}`, { method: 'DELETE' }),

    // Task Status Transitions (工作流流转规则,对齐 Jira workflow)
    // 空规则集=全允许(向后兼容);非空时任务状态 from→to 必须命中某条
    getTransitions: (projectSlug: string) =>
      request<TaskStatusTransition[]>(`/projects/${projectSlug}/task-statuses/transitions`),
    setTransitions: (projectSlug: string, transitions: { from: string; to: string }[]) =>
      request<TaskStatusTransition[]>(`/projects/${projectSlug}/task-statuses/transitions`, { method: 'PUT', body: { transitions } }),

    // Scrum Activities (project-scoped: project_id filter prevents cross-project
    // bleed-through since entity_id for tasks is a composite key with project_id)
    // 后端 sideload participants：批量预取操作者头像信息，前端无需逐条 /users/:name
    getActivities: (projectSlug: string, entityType: string, entityId: number | string) =>
      request<{ activities: ScrumActivity[]; participants: Participant[] }>(`/projects/${projectSlug}/activities/${entityType}/${entityId}`),

    // Velocity Chart（跨 Sprint 速度图，对齐 Jira Velocity Report）
    // 返回最近 N 个已关闭 Sprint 的承诺/完成点数，按时间正序排列
    getVelocity: (projectSlug: string, count?: number) => {
      const qs = count ? `?count=${count}` : ''
      return request<VelocityDataPoint[]>(`/projects/${projectSlug}/reports/velocity${qs}`)
    },

    // Sprint Report（范围变更追踪 + 承诺 vs 实际，对齐 Jira Sprint Report）
    getSprintReport: (sprintSlug: string) =>
      request<SprintReport>(`/sprints/${sprintSlug}/report`),

    // AI 拆解用户故事为任务建议（不入库，前端确认后循环调用 createTask）
    // locale 控制输出语言（"zh"/"en"），后端据此指示 LLM 用对应语言生成标题与描述
    // 三种 SSE 回调：
    //   onPrompt   — 收到完整提示时调用（让用户看到「发给 AI 的请求」）
    //   onThinking — 收到 reasoning_content delta 时调用（AI 的思考过程，DeepSeek 等模型支持）
    //   onDelta    — 收到 content delta 时调用（AI 的最终 JSON 输出）
    // 注意：此处不走通用 request() 封装，因为需要直接读取 ReadableStream 消费 SSE 事件
    aiDecomposeUserStory: async (
      userStorySlug: string,
      locale?: string,
      onDelta?: (text: string) => void,
      onThinking?: (text: string) => void,
      onPrompt?: (text: string) => void,
    ): Promise<AIDecomposeResult> => {
      const endpoint = `/user-stories/${userStorySlug}/ai-decompose${locale ? `?lang=${locale}` : ''}`
      const url = `${baseUrl}${endpoint}`
      const token = typeof window !== 'undefined' ? localStorage.getItem('token') : null
      const headers: Record<string, string> = {
        'Content-Type': 'application/json',
        Accept: 'text/event-stream',
      }
      if (token) headers['Authorization'] = token

      const response = await fetch(url, {
        method: 'POST',
        headers,
        credentials: 'include',
      })

      if (!response.ok) {
        // 非 SSE 错误（如 401/403/404，在流开始前返回的 JSON 错误）
        const errBody = await response.json().catch(() => ({ error: response.statusText }))
        const err = new Error(errBody.error || errBody.message || `API error: ${response.status}`) as ApiError
        err.status = response.status
        if (response.status === 401) {
          localStorage.removeItem('token')
          localStorage.removeItem('cached_user')
          if (typeof window !== 'undefined' && !window.location.pathname.startsWith('/login')) {
            window.location.href = '/login'
          }
        }
        throw err
      }

      const reader = response.body!.getReader()
      const decoder = new TextDecoder()
      let buffer = ''
      let result: AIDecomposeResult | null = null
      let streamError: { message: string; reason?: string } | null = null

      while (true) {
        const { done, value } = await reader.read()
        if (done) break
        buffer += decoder.decode(value, { stream: true })

        // SSE 事件以 \n\n 分隔
        const events = buffer.split('\n\n')
        buffer = events.pop() || '' // 保留最后不完整的事件

        for (const eventBlock of events) {
          const lines = eventBlock.split('\n')
          let eventType = ''
          let eventData = ''
          for (const line of lines) {
            if (line.startsWith('event: ')) eventType = line.slice(7).trim()
            else if (line.startsWith('data: ')) eventData = line.slice(6)
          }
          if (!eventType) continue

          if (eventType === 'prompt' && onPrompt) {
            try {
              const parsed = JSON.parse(eventData)
              onPrompt(parsed.content || '')
            } catch {
              /* 忽略 */
            }
          } else if (eventType === 'thinking' && onThinking) {
            try {
              const parsed = JSON.parse(eventData)
              onThinking(parsed.content || '')
            } catch {
              /* 忽略 */
            }
          } else if (eventType === 'delta' && onDelta) {
            try {
              const parsed = JSON.parse(eventData)
              onDelta(parsed.content || '')
            } catch {
              /* 跳过无法解析的 delta */
            }
          } else if (eventType === 'done') {
            try {
              result = JSON.parse(eventData) as AIDecomposeResult
            } catch {
              /* 忽略解析错误 */
            }
          } else if (eventType === 'error') {
            try {
              const parsed = JSON.parse(eventData)
              streamError = { message: parsed.error || 'Unknown error', reason: parsed.reason }
            } catch {
              streamError = { message: eventData }
            }
          }
        }
      }

      if (streamError) {
        const err = new Error(streamError.message) as ApiError
        err.status = streamError.reason === 'ai_not_configured' ? 412 : 502
        err.reason = streamError.reason
        throw err
      }

      if (!result) {
        throw new Error('AI stream ended without result')
      }

      return result
    },

    // AI 优化用户故事草稿（不入库，前端确认后回填表单字段）
    // 接收当前草稿内容（title/description/acceptanceCriteria），返回优化后的内容 + 建议优先级/故事点
    // 与 aiDecomposeUserStory 同样采用 SSE 流式，三种回调用于实时展示 prompt/thinking/delta
    aiOptimizeUserStory: async (
      projectSlug: string,
      draft: { title: string; description?: string; acceptanceCriteria?: string },
      locale?: string,
      onDelta?: (text: string) => void,
      onThinking?: (text: string) => void,
      onPrompt?: (text: string) => void,
    ): Promise<AIOptimizeResult> => {
      const endpoint = `/projects/${projectSlug}/user-stories/ai-optimize${locale ? `?lang=${locale}` : ''}`
      const url = `${baseUrl}${endpoint}`
      const token = typeof window !== 'undefined' ? localStorage.getItem('token') : null
      const headers: Record<string, string> = {
        'Content-Type': 'application/json',
        Accept: 'text/event-stream',
      }
      if (token) headers['Authorization'] = token

      const response = await fetch(url, {
        method: 'POST',
        headers,
        credentials: 'include',
        body: JSON.stringify({
          title: draft.title,
          description: draft.description || '',
          acceptanceCriteria: draft.acceptanceCriteria || '',
        }),
      })

      if (!response.ok) {
        const errBody = await response.json().catch(() => ({ error: response.statusText }))
        const err = new Error(errBody.error || errBody.message || `API error: ${response.status}`) as ApiError
        err.status = response.status
        if (response.status === 401) {
          localStorage.removeItem('token')
          localStorage.removeItem('cached_user')
          if (typeof window !== 'undefined' && !window.location.pathname.startsWith('/login')) {
            window.location.href = '/login'
          }
        }
        throw err
      }

      const reader = response.body!.getReader()
      const decoder = new TextDecoder()
      let buffer = ''
      let result: AIOptimizeResult | null = null
      let streamError: { message: string; reason?: string } | null = null

      while (true) {
        const { done, value } = await reader.read()
        if (done) break
        buffer += decoder.decode(value, { stream: true })

        const events = buffer.split('\n\n')
        buffer = events.pop() || ''

        for (const eventBlock of events) {
          const lines = eventBlock.split('\n')
          let eventType = ''
          let eventData = ''
          for (const line of lines) {
            if (line.startsWith('event: ')) eventType = line.slice(7).trim()
            else if (line.startsWith('data: ')) eventData = line.slice(6)
          }
          if (!eventType) continue

          if (eventType === 'prompt' && onPrompt) {
            try {
              const parsed = JSON.parse(eventData)
              onPrompt(parsed.content || '')
            } catch {
              /* 忽略 */
            }
          } else if (eventType === 'thinking' && onThinking) {
            try {
              const parsed = JSON.parse(eventData)
              onThinking(parsed.content || '')
            } catch {
              /* 忽略 */
            }
          } else if (eventType === 'delta' && onDelta) {
            try {
              const parsed = JSON.parse(eventData)
              onDelta(parsed.content || '')
            } catch {
              /* 跳过无法解析的 delta */
            }
          } else if (eventType === 'done') {
            try {
              result = JSON.parse(eventData) as AIOptimizeResult
            } catch {
              /* 忽略解析错误 */
            }
          } else if (eventType === 'error') {
            try {
              const parsed = JSON.parse(eventData)
              streamError = { message: parsed.error || 'Unknown error', reason: parsed.reason }
            } catch {
              streamError = { message: eventData }
            }
          }
        }
      }

      if (streamError) {
        const err = new Error(streamError.message) as ApiError
        err.status = streamError.reason === 'ai_not_configured' ? 412 : 502
        err.reason = streamError.reason
        throw err
      }

      if (!result) {
        throw new Error('AI stream ended without result')
      }

      return result
    },

    // AI 优化任务草稿（不入库，前端确认后回填表单字段）
    // 与 aiOptimizeUserStory 镜像，但针对任务结构（无 acceptanceCriteria，有 taskType）
    aiOptimizeTask: async (
      projectSlug: string,
      draft: { title: string; description?: string },
      locale?: string,
      onDelta?: (text: string) => void,
      onThinking?: (text: string) => void,
      onPrompt?: (text: string) => void,
    ): Promise<AIOptimizeTaskResult> => {
      const endpoint = `/projects/${projectSlug}/tasks/ai-optimize${locale ? `?lang=${locale}` : ''}`
      const url = `${baseUrl}${endpoint}`
      const token = typeof window !== 'undefined' ? localStorage.getItem('token') : null
      const headers: Record<string, string> = {
        'Content-Type': 'application/json',
        Accept: 'text/event-stream',
      }
      if (token) headers['Authorization'] = token

      const response = await fetch(url, {
        method: 'POST',
        headers,
        credentials: 'include',
        body: JSON.stringify({
          title: draft.title,
          description: draft.description || '',
        }),
      })

      if (!response.ok) {
        const errBody = await response.json().catch(() => ({ error: response.statusText }))
        const err = new Error(errBody.error || errBody.message || `API error: ${response.status}`) as ApiError
        err.status = response.status
        if (response.status === 401) {
          localStorage.removeItem('token')
          localStorage.removeItem('cached_user')
          if (typeof window !== 'undefined' && !window.location.pathname.startsWith('/login')) {
            window.location.href = '/login'
          }
        }
        throw err
      }

      const reader = response.body!.getReader()
      const decoder = new TextDecoder()
      let buffer = ''
      let result: AIOptimizeTaskResult | null = null
      let streamError: { message: string; reason?: string } | null = null

      while (true) {
        const { done, value } = await reader.read()
        if (done) break
        buffer += decoder.decode(value, { stream: true })

        const events = buffer.split('\n\n')
        buffer = events.pop() || ''

        for (const eventBlock of events) {
          const lines = eventBlock.split('\n')
          let eventType = ''
          let eventData = ''
          for (const line of lines) {
            if (line.startsWith('event: ')) eventType = line.slice(7).trim()
            else if (line.startsWith('data: ')) eventData = line.slice(6)
          }
          if (!eventType) continue

          if (eventType === 'prompt' && onPrompt) {
            try {
              const parsed = JSON.parse(eventData)
              onPrompt(parsed.content || '')
            } catch {
              /* 忽略 */
            }
          } else if (eventType === 'thinking' && onThinking) {
            try {
              const parsed = JSON.parse(eventData)
              onThinking(parsed.content || '')
            } catch {
              /* 忽略 */
            }
          } else if (eventType === 'delta' && onDelta) {
            try {
              const parsed = JSON.parse(eventData)
              onDelta(parsed.content || '')
            } catch {
              /* 跳过无法解析的 delta */
            }
          } else if (eventType === 'done') {
            try {
              result = JSON.parse(eventData) as AIOptimizeTaskResult
            } catch {
              /* 忽略解析错误 */
            }
          } else if (eventType === 'error') {
            try {
              const parsed = JSON.parse(eventData)
              streamError = { message: parsed.error || 'Unknown error', reason: parsed.reason }
            } catch {
              streamError = { message: eventData }
            }
          }
        }
      }

      if (streamError) {
        const err = new Error(streamError.message) as ApiError
        err.status = streamError.reason === 'ai_not_configured' ? 412 : 502
        err.reason = streamError.reason
        throw err
      }

      if (!result) {
        throw new Error('AI stream ended without result')
      }

      return result
    },

    // AI 优化迭代草稿（不入库，前端确认后回填表单字段）
    // 与 aiOptimizeTask 镜像，但针对迭代结构（有 goal，无 priority/taskType/storyPoints/acceptanceCriteria）
    aiOptimizeSprint: async (
      projectSlug: string,
      draft: { title: string; description?: string; goal?: string },
      locale?: string,
      onDelta?: (text: string) => void,
      onThinking?: (text: string) => void,
      onPrompt?: (text: string) => void,
    ): Promise<AIOptimizeSprintResult> => {
      const endpoint = `/projects/${projectSlug}/sprints/ai-optimize${locale ? `?lang=${locale}` : ''}`
      const url = `${baseUrl}${endpoint}`
      const token = typeof window !== 'undefined' ? localStorage.getItem('token') : null
      const headers: Record<string, string> = {
        'Content-Type': 'application/json',
        Accept: 'text/event-stream',
      }
      if (token) headers['Authorization'] = token

      const response = await fetch(url, {
        method: 'POST',
        headers,
        credentials: 'include',
        body: JSON.stringify({
          title: draft.title,
          description: draft.description || '',
          goal: draft.goal || '',
        }),
      })

      if (!response.ok) {
        const errBody = await response.json().catch(() => ({ error: response.statusText }))
        const err = new Error(errBody.error || errBody.message || `API error: ${response.status}`) as ApiError
        err.status = response.status
        if (response.status === 401) {
          localStorage.removeItem('token')
          localStorage.removeItem('cached_user')
          if (typeof window !== 'undefined' && !window.location.pathname.startsWith('/login')) {
            window.location.href = '/login'
          }
        }
        throw err
      }

      const reader = response.body!.getReader()
      const decoder = new TextDecoder()
      let buffer = ''
      let result: AIOptimizeSprintResult | null = null
      let streamError: { message: string; reason?: string } | null = null

      while (true) {
        const { done, value } = await reader.read()
        if (done) break
        buffer += decoder.decode(value, { stream: true })

        const events = buffer.split('\n\n')
        buffer = events.pop() || ''

        for (const eventBlock of events) {
          const lines = eventBlock.split('\n')
          let eventType = ''
          let eventData = ''
          for (const line of lines) {
            if (line.startsWith('event: ')) eventType = line.slice(7).trim()
            else if (line.startsWith('data: ')) eventData = line.slice(6)
          }
          if (!eventType) continue

          if (eventType === 'prompt' && onPrompt) {
            try {
              const parsed = JSON.parse(eventData)
              onPrompt(parsed.content || '')
            } catch {
              /* 忽略 */
            }
          } else if (eventType === 'thinking' && onThinking) {
            try {
              const parsed = JSON.parse(eventData)
              onThinking(parsed.content || '')
            } catch {
              /* 忽略 */
            }
          } else if (eventType === 'delta' && onDelta) {
            try {
              const parsed = JSON.parse(eventData)
              onDelta(parsed.content || '')
            } catch {
              /* 跳过无法解析的 delta */
            }
          } else if (eventType === 'done') {
            try {
              result = JSON.parse(eventData) as AIOptimizeSprintResult
            } catch {
              /* 忽略解析错误 */
            }
          } else if (eventType === 'error') {
            try {
              const parsed = JSON.parse(eventData)
              streamError = { message: parsed.error || 'Unknown error', reason: parsed.reason }
            } catch {
              streamError = { message: eventData }
            }
          }
        }
      }

      if (streamError) {
        const err = new Error(streamError.message) as ApiError
        err.status = streamError.reason === 'ai_not_configured' ? 412 : 502
        err.reason = streamError.reason
        throw err
      }

      if (!result) {
        throw new Error('AI stream ended without result')
      }

      return result
    },
  }
}
