package router

import (
	"iforge/iforge/internal/container"

	"github.com/gofiber/fiber/v2"
)

// registerIssueRoutes registers issue routes
func registerIssueRoutes(api fiber.Router, c *container.Container) {
	api.Get("/repos/:owner/:repo/issues", c.IssueHandler().ListIssues)
	api.Post("/repos/:owner/:repo/issues", c.IssueHandler().CreateIssue)
	api.Post("/repos/:owner/:repo/issues/batch", c.IssueHandler().BatchUpdateIssues)
	api.Get("/repos/:owner/:repo/issues/templates", c.IssueHandler().GetIssueTemplates)
	api.Get("/repos/:owner/:repo/issues/authors", c.IssueHandler().ListIssueAuthors)
	api.Get("/repos/:owner/:repo/issues/assignees", c.IssueHandler().ListIssueAssignees)
	api.Get("/repos/:owner/:repo/issues/search", c.SearchHandler().SearchIssues)
	api.Get("/repos/:owner/:repo/issues/:id", c.IssueHandler().GetIssue)
	api.Patch("/repos/:owner/:repo/issues/:id", c.IssueHandler().UpdateIssue)
	api.Delete("/repos/:owner/:repo/issues/:id", c.IssueHandler().DeleteIssue)
	api.Post("/repos/:owner/:repo/issues/:id/lock", c.IssueHandler().LockIssue)
	api.Delete("/repos/:owner/:repo/issues/:id/lock", c.IssueHandler().UnlockIssue)
	api.Post("/repos/:owner/:repo/issues/:id/comments", c.IssueHandler().CreateComment)
	api.Get("/repos/:owner/:repo/issues/:id/comments", c.IssueHandler().ListComments)
	api.Patch("/repos/:owner/:repo/issues/:id/comments/:commentId", c.IssueHandler().UpdateComment)
	api.Delete("/repos/:owner/:repo/issues/:id/comments/:commentId", c.IssueHandler().DeleteComment)
	api.Get("/repos/:owner/:repo/issues/:id/assignees", c.IssueHandler().ListAssignees)
	api.Post("/repos/:owner/:repo/issues/:id/assignees/:username", c.IssueHandler().AddAssignee)
	api.Delete("/repos/:owner/:repo/issues/:id/assignees/:username", c.IssueHandler().RemoveAssignee)
	api.Get("/search/issues", c.SearchHandler().SearchAllIssues)
}

// registerMergeRequestRoutes registers merge request routes
func registerMergeRequestRoutes(api fiber.Router, c *container.Container) {
	api.Get("/repos/:owner/:repo/merge-requests", c.MergeRequestHandler().ListMergeRequests)
	api.Post("/repos/:owner/:repo/merge-requests", c.MergeRequestHandler().CreateMergeRequest)
	api.Get("/repos/:owner/:repo/compare/:basehead", c.MergeRequestHandler().Compare)
	api.Get("/repos/:owner/:repo/merge-requests/:id", c.MergeRequestHandler().GetMergeRequest)
	api.Patch("/repos/:owner/:repo/merge-requests/:id", c.MergeRequestHandler().UpdateMergeRequest)
	api.Post("/repos/:owner/:repo/merge-requests/:id/merge", c.MergeRequestHandler().MergeMergeRequest)
	api.Get("/repos/:owner/:repo/merge-requests/:id/diff", c.MergeRequestHandler().GetMergeRequestDiff)
	api.Get("/repos/:owner/:repo/branches/:branch/ahead-behind", c.MergeRequestHandler().GetAheadBehind)
}

// registerLabelRoutes registers label routes
func registerLabelRoutes(api fiber.Router, c *container.Container) {
	api.Get("/repos/:owner/:repo/labels", c.LabelHandler().ListLabels)
	api.Post("/repos/:owner/:repo/labels", c.LabelHandler().CreateLabel)
	api.Patch("/repos/:owner/:repo/labels/:id", c.LabelHandler().UpdateLabel)
	api.Delete("/repos/:owner/:repo/labels/:id", c.LabelHandler().DeleteLabel)
	api.Get("/repos/:owner/:repo/issues/:id/labels", c.LabelHandler().GetIssueLabels)
	api.Post("/repos/:owner/:repo/issues/:id/labels/:labelId", c.LabelHandler().AddLabelToIssue)
	api.Delete("/repos/:owner/:repo/issues/:id/labels/:labelId", c.LabelHandler().RemoveLabelFromIssue)
}

// registerMilestoneRoutes registers milestone routes
func registerMilestoneRoutes(api fiber.Router, c *container.Container) {
	api.Get("/repos/:owner/:repo/milestones", c.MilestoneHandler().ListMilestones)
	api.Post("/repos/:owner/:repo/milestones", c.MilestoneHandler().CreateMilestone)
	api.Get("/repos/:owner/:repo/milestones/:id", c.MilestoneHandler().GetMilestone)
	api.Patch("/repos/:owner/:repo/milestones/:id", c.MilestoneHandler().UpdateMilestone)
	api.Delete("/repos/:owner/:repo/milestones/:id", c.MilestoneHandler().DeleteMilestone)
	api.Post("/repos/:owner/:repo/milestones/:id/close", c.MilestoneHandler().CloseMilestone)
	api.Post("/repos/:owner/:repo/milestones/:id/reopen", c.MilestoneHandler().ReopenMilestone)
}

// registerCustomFieldRoutes registers custom field routes
func registerCustomFieldRoutes(api fiber.Router, c *container.Container) {
	api.Get("/repos/:owner/:repo/custom_fields", c.CustomFieldHandler().ListCustomFields)
	api.Post("/repos/:owner/:repo/custom_fields", c.CustomFieldHandler().CreateCustomField)
	api.Get("/repos/:owner/:repo/custom_fields/:id", c.CustomFieldHandler().GetCustomField)
	api.Patch("/repos/:owner/:repo/custom_fields/:id", c.CustomFieldHandler().UpdateCustomField)
	api.Delete("/repos/:owner/:repo/custom_fields/:id", c.CustomFieldHandler().DeleteCustomField)
	api.Get("/repos/:owner/:repo/issues/:id/custom_fields", c.CustomFieldHandler().GetIssueCustomFieldValues)
	api.Post("/repos/:owner/:repo/issues/:id/custom_fields/:fieldId", c.CustomFieldHandler().SetIssueCustomFieldValue)
}

// registerPriorityRoutes registers priority routes
func registerPriorityRoutes(api fiber.Router, c *container.Container) {
	api.Get("/repos/:owner/:repo/priorities", c.PriorityHandler().ListPriorities)
	api.Post("/repos/:owner/:repo/priorities", c.PriorityHandler().CreatePriority)
	// Static paths must be registered before /:id to avoid being captured by the parameter
	api.Post("/repos/:owner/:repo/priorities/reorder", c.PriorityHandler().ReorderPriorities)
	api.Get("/repos/:owner/:repo/priorities/default", c.PriorityHandler().GetDefaultPriority)
	api.Post("/repos/:owner/:repo/priorities/default", c.PriorityHandler().SetDefaultPriority)
	api.Get("/repos/:owner/:repo/priorities/:id", c.PriorityHandler().GetPriority)
	api.Patch("/repos/:owner/:repo/priorities/:id", c.PriorityHandler().UpdatePriority)
	api.Delete("/repos/:owner/:repo/priorities/:id", c.PriorityHandler().DeletePriority)
}

// registerReviewRoutes registers review routes
func registerReviewRoutes(api fiber.Router, c *container.Container) {
	api.Get("/repos/:owner/:repo/merge-requests/:id/reviews", c.ReviewHandler().ListReviews)
	api.Post("/repos/:owner/:repo/merge-requests/:id/reviews", c.ReviewHandler().CreateReview)
	api.Get("/repos/:owner/:repo/merge-requests/:id/reviews/:reviewId", c.ReviewHandler().GetReview)
	api.Patch("/repos/:owner/:repo/merge-requests/:id/reviews/:reviewId", c.ReviewHandler().UpdateReview)
	api.Delete("/repos/:owner/:repo/merge-requests/:id/reviews/:reviewId", c.ReviewHandler().DeleteReview)
	api.Get("/repos/:owner/:repo/merge-requests/:id/reviews/:reviewId/comments", c.ReviewHandler().ListReviewComments)
	api.Post("/repos/:owner/:repo/merge-requests/:id/reviews/:reviewId/comments", c.ReviewHandler().CreateReviewComment)
	api.Patch("/repos/:owner/:repo/merge-requests/:id/reviews/:reviewId/comments/:commentId", c.ReviewHandler().UpdateReviewComment)
	api.Delete("/repos/:owner/:repo/merge-requests/:id/reviews/:reviewId/comments/:commentId", c.ReviewHandler().DeleteReviewComment)
}
