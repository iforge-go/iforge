package router

import (
	"iforge/iforge/internal/container"

	"github.com/gofiber/fiber/v2"
)

// registerProjectRoutes registers project management routes
func registerProjectRoutes(api fiber.Router, c *container.Container) {
	// Projects
	api.Get("/projects", c.ProjectHandler().ListProjects)
	api.Post("/projects", c.ProjectHandler().CreateProject)
	api.Get("/projects/:projectSlug", c.ProjectHandler().GetProject)
	api.Put("/projects/:projectSlug", c.ProjectHandler().UpdateProject)
	api.Delete("/projects/:projectSlug", c.ProjectHandler().DeleteProject)

	// Project Members
	api.Get("/projects/:projectSlug/members", c.ProjectHandler().GetMembers)
	api.Post("/projects/:projectSlug/members", c.ProjectHandler().AddMember)
	api.Put("/projects/:projectSlug/members/:userName", c.ProjectHandler().UpdateMemberRole)
	api.Delete("/projects/:projectSlug/members/:userName", c.ProjectHandler().RemoveMember)

	// Project Repositories (link VCS repos to Scrum projects)
	api.Get("/projects/:projectSlug/repositories", c.ProjectHandler().ListRepositories)
	api.Post("/projects/:projectSlug/repositories", c.ProjectHandler().AddRepository)
	api.Delete("/projects/:projectSlug/repositories/:userName/:repoName", c.ProjectHandler().RemoveRepository)

	// Sprints
	api.Get("/projects/:projectSlug/sprints", c.SprintHandler().ListSprints)
	api.Post("/projects/:projectSlug/sprints", c.SprintHandler().CreateSprint)
	// AI-optimized sprint draft (no pre-existing sprint required, for create/edit scenarios)
	api.Post("/projects/:projectSlug/sprints/ai-optimize", c.SprintHandler().AIOptimizeSprint)
	api.Get("/sprints/:sprintSlug", c.SprintHandler().GetSprint)
	api.Put("/sprints/:sprintSlug", c.SprintHandler().UpdateSprint)
	api.Delete("/sprints/:sprintSlug", c.SprintHandler().DeleteSprint)
	api.Get("/sprints/:sprintSlug/tasks", c.SprintHandler().GetSprintTasks)
	api.Get("/sprints/:sprintSlug/stories", c.SprintHandler().GetSprintStories)
	// Sprint Report (scope change tracking + commitment vs actual, aligned with Jira Sprint Report)
	api.Get("/sprints/:sprintSlug/report", c.SprintHandler().GetSprintReport)
	// Velocity Chart (cross-sprint velocity chart, aligned with Jira Velocity Report)
	api.Get("/projects/:projectSlug/reports/velocity", c.SprintHandler().GetVelocity)

	// Epics
	api.Get("/projects/:projectSlug/epics", c.EpicHandler().ListEpics)
	api.Post("/projects/:projectSlug/epics", c.EpicHandler().CreateEpic)
	api.Get("/epics/:epicSlug", c.EpicHandler().GetEpic)
	api.Put("/epics/:epicSlug", c.EpicHandler().UpdateEpic)
	api.Delete("/epics/:epicSlug", c.EpicHandler().DeleteEpic)
	api.Get("/epics/:epicSlug/stories", c.EpicHandler().ListEpicStories)
	api.Get("/epics/:epicSlug/progress", c.EpicHandler().GetEpicProgress)
	api.Get("/epics/:epicSlug/sprints", c.EpicHandler().GetEpicSprints)
	api.Post("/epics/:epicSlug/stories", c.EpicHandler().AssignStoryToEpic)
	api.Delete("/epics/:epicSlug/stories/:storySlug", c.EpicHandler().RemoveStoryFromEpic)

	// User Stories
	api.Get("/projects/:projectSlug/user-stories", c.UserStoryHandler().ListUserStories)
	api.Post("/projects/:projectSlug/user-stories", c.UserStoryHandler().CreateUserStory)
	api.Get("/user-stories/:userStorySlug", c.UserStoryHandler().GetUserStory)
	api.Put("/user-stories/:userStorySlug", c.UserStoryHandler().UpdateUserStory)
	api.Delete("/user-stories/:userStorySlug", c.UserStoryHandler().DeleteUserStory)
	api.Get("/user-stories/:userStorySlug/tasks", c.UserStoryHandler().GetUserStoryTasks)
	api.Post("/user-stories/:userStorySlug/ai-decompose", c.UserStoryHandler().AIDecomposeUserStory)
	// AI-optimized user story draft (no pre-existing story required, for create/edit scenarios)
	api.Post("/projects/:projectSlug/user-stories/ai-optimize", c.UserStoryHandler().AIOptimizeUserStory)

	// Tasks
	api.Get("/projects/:projectSlug/tasks", c.TaskHandler().ListTasks)
	api.Post("/projects/:projectSlug/tasks", c.TaskHandler().CreateTask)
	api.Post("/projects/:projectSlug/tasks/ai-optimize", c.TaskHandler().AIOptimizeTask)
	api.Get("/projects/:projectSlug/tasks/:taskId", c.TaskHandler().GetTask)
	api.Put("/projects/:projectSlug/tasks/:taskId", c.TaskHandler().UpdateTask)
	api.Delete("/projects/:projectSlug/tasks/:taskId", c.TaskHandler().DeleteTask)
	api.Put("/projects/:projectSlug/tasks/:taskId/position", c.TaskHandler().UpdatePosition)
	api.Get("/projects/:projectSlug/tasks/:taskId/subtasks", c.TaskHandler().GetSubtasks)
	api.Get("/projects/:projectSlug/tasks/:taskId/status-history", c.TaskHandler().GetStatusHistory)

	// Task Assignees
	api.Get("/projects/:projectSlug/tasks/:taskId/assignees", c.TaskHandler().ListAssignees)
	api.Post("/projects/:projectSlug/tasks/:taskId/assignees/:username", c.TaskHandler().AddAssignee)
	api.Delete("/projects/:projectSlug/tasks/:taskId/assignees/:username", c.TaskHandler().RemoveAssignee)

	// Task Branches (loosely coupled: only records branch physical location, no VCS table references)
	api.Get("/projects/:projectSlug/tasks/:taskId/branches", c.TaskHandler().ListTaskBranches)
	api.Post("/projects/:projectSlug/tasks/:taskId/branches", c.TaskHandler().AssociateBranch)
	api.Post("/projects/:projectSlug/tasks/:taskId/branches/create", c.TaskHandler().CreateAndAssociateBranch)
	api.Delete("/projects/:projectSlug/tasks/:taskId/branches/:branchId", c.TaskHandler().RemoveTaskBranch)
	// Task Dependencies (hard blocking: prevents transition to in-progress when predecessor is incomplete)
	api.Get("/projects/:projectSlug/tasks/:taskId/dependencies", c.TaskHandler().ListDependencies)
	api.Post("/projects/:projectSlug/tasks/:taskId/dependencies", c.TaskHandler().AddDependency)
	api.Delete("/projects/:projectSlug/tasks/:taskId/dependencies/:dependsOnTaskId", c.TaskHandler().RemoveDependency)
	api.Get("/projects/:projectSlug/tasks/:taskId/blocks", c.TaskHandler().ListBlocks)
	// Task Commits (commit log from associated branches, compared against default branch for ahead commits)
	api.Get("/projects/:projectSlug/tasks/:taskId/commits", c.TaskHandler().GetTaskCommits)

	// Task Comments
	api.Get("/projects/:projectSlug/tasks/:taskId/comments", c.TaskHandler().GetComments)
	api.Post("/projects/:projectSlug/tasks/:taskId/comments", c.TaskHandler().CreateComment)
	api.Put("/comments/:commentId", c.TaskHandler().UpdateComment)
	api.Delete("/comments/:commentId", c.TaskHandler().DeleteComment)

	// Task Work Logs (Time Tracking)
	api.Get("/projects/:projectSlug/tasks/:taskId/work-logs", c.TaskHandler().GetWorkLogs)
	api.Post("/projects/:projectSlug/tasks/:taskId/work-logs", c.TaskHandler().CreateWorkLog)
	api.Put("/work-logs/:logId", c.TaskHandler().UpdateWorkLog)
	api.Delete("/work-logs/:logId", c.TaskHandler().DeleteWorkLog)

	// Task Labels
	api.Get("/projects/:projectSlug/labels", c.TaskHandler().GetProjectLabels)
	api.Post("/projects/:projectSlug/labels", c.TaskHandler().CreateLabel)
	api.Get("/projects/:projectSlug/tasks/:taskId/labels", c.TaskHandler().GetLabels)
	api.Post("/projects/:projectSlug/tasks/:taskId/labels", c.TaskHandler().AddLabel)
	api.Delete("/projects/:projectSlug/tasks/:taskId/labels/:labelId", c.TaskHandler().RemoveLabel)

	// Task Statuses (Kanban columns)
	api.Get("/projects/:projectSlug/task-statuses", c.TaskStatusHandler().ListTaskStatuses)
	api.Post("/projects/:projectSlug/task-statuses", c.TaskStatusHandler().CreateTaskStatus)
	api.Put("/task-statuses/:statusId", c.TaskStatusHandler().UpdateTaskStatus)
	api.Delete("/task-statuses/:statusId", c.TaskStatusHandler().DeleteTaskStatus)
	// Task Status Transitions (workflow transition rules, aligned with Jira workflow)
	api.Get("/projects/:projectSlug/task-statuses/transitions", c.TaskStatusHandler().ListTransitions)
	api.Put("/projects/:projectSlug/task-statuses/transitions", c.TaskStatusHandler().SetTransitions)

	// Scrum activities (project-scoped: entity_id for tasks is a composite key
	// with project_id, so the project filter is required to avoid cross-project
	// activity bleed-through)
	api.Get("/projects/:projectSlug/activities/:entityType/:entityId", c.ScrumActivityHandler().GetActivities)
}
