package model

import "time"

// Activity represents an activity log entry
type Activity struct {
	ActivityID       string    `gorm:"primaryKey;column:activity_id" json:"activityId"`
	// Composite index (user_name, repository_name) covers GetActivities
	UserName         string    `gorm:"column:user_name;index:idx_activity_repo,priority:1" json:"userName"`
	RepositoryName   string    `gorm:"column:repository_name;index:idx_activity_repo,priority:2" json:"repositoryName"`
	// Composite index (activity_user_name, activity_date) covers GetUserActivities + GetUserContributionDays
	ActivityUserName string    `gorm:"column:activity_user_name;index:idx_activity_user_date,priority:1" json:"activityUserName"`
	ActivityType     string    `gorm:"column:activity_type" json:"activityType"`
	Message          string    `gorm:"column:message" json:"message"`
	AdditionalInfo   *string   `gorm:"column:additional_info" json:"additionalInfo"`
	ActivityDate     time.Time `gorm:"column:activity_date;index:idx_activity_user_date,priority:2" json:"activityDate"`
	// ActivityDateDay is a denormalized local-date column (format "2006-01-02")
	// computed by the service layer on CreateActivity. Enables GROUP BY on the
	// contribution heatmap to use the index instead of wrapping activity_date
	// with DATE(..., 'localtime'), which would invalidate the index.
	ActivityDateDay string `gorm:"column:activity_date_day;index:idx_activity_user_day,priority:1" json:"-"`
}

func (Activity) TableName() string { return "activity" }

// ContributionDay represents daily contribution count
type ContributionDay struct {
	Date  string `json:"date"`
	Count int    `json:"count"`
}
