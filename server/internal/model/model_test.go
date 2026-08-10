package model

import (
	"testing"
)

func TestTableName_Account(t *testing.T) {
	account := Account{}
	if account.TableName() != "account" {
		t.Errorf("Expected table name 'account', got '%s'", account.TableName())
	}
}

func TestTableName_AccountExtraMailAddress(t *testing.T) {
	addr := AccountExtraMailAddress{}
	if addr.TableName() != "account_extra_mail_address" {
		t.Errorf("Expected table name 'account_extra_mail_address', got '%s'", addr.TableName())
	}
}

func TestTableName_AccountPreference(t *testing.T) {
	pref := AccountPreference{}
	if pref.TableName() != "account_preference" {
		t.Errorf("Expected table name 'account_preference', got '%s'", pref.TableName())
	}
}

func TestTableName_OrganizationMember(t *testing.T) {
	member := OrganizationMember{}
	if member.TableName() != "organization_member" {
		t.Errorf("Expected table name 'organization_member', got '%s'", member.TableName())
	}
}

func TestTableName_SSHKey(t *testing.T) {
	key := SSHKey{}
	if key.TableName() != "ssh_key" {
		t.Errorf("Expected table name 'ssh_key', got '%s'", key.TableName())
	}
}

func TestTableName_GPGKey(t *testing.T) {
	key := GPGKey{}
	if key.TableName() != "gpg_key" {
		t.Errorf("Expected table name 'gpg_key', got '%s'", key.TableName())
	}
}

func TestTableName_AccessToken(t *testing.T) {
	token := AccessToken{}
	if token.TableName() != "access_token" {
		t.Errorf("Expected table name 'access_token', got '%s'", token.TableName())
	}
}

func TestTableName_Activity(t *testing.T) {
	activity := Activity{}
	if activity.TableName() != "activity" {
		t.Errorf("Expected table name 'activity', got '%s'", activity.TableName())
	}
}

func TestTableName_AIModelConfig(t *testing.T) {
	config := AIModelConfig{}
	if config.TableName() != "ai_model_config" {
		t.Errorf("Expected table name 'ai_model_config', got '%s'", config.TableName())
	}
}

func TestTableName_AuditLog(t *testing.T) {
	log := AuditLog{}
	if log.TableName() != "audit_logs" {
		t.Errorf("Expected table name 'audit_logs', got '%s'", log.TableName())
	}
}

func TestTableName_Pipeline(t *testing.T) {
	pipeline := Pipeline{}
	if pipeline.TableName() != "cicd_pipeline" {
		t.Errorf("Expected table name 'cicd_pipeline', got '%s'", pipeline.TableName())
	}
}

func TestTableName_Job(t *testing.T) {
	job := Job{}
	if job.TableName() != "cicd_job" {
		t.Errorf("Expected table name 'cicd_job', got '%s'", job.TableName())
	}
}

func TestTableName_JobLog(t *testing.T) {
	log := JobLog{}
	if log.TableName() != "cicd_job_log" {
		t.Errorf("Expected table name 'cicd_job_log', got '%s'", log.TableName())
	}
}

func TestTableName_Artifact(t *testing.T) {
	artifact := Artifact{}
	if artifact.TableName() != "cicd_artifact" {
		t.Errorf("Expected table name 'cicd_artifact', got '%s'", artifact.TableName())
	}
}

func TestTableName_Runner(t *testing.T) {
	runner := Runner{}
	if runner.TableName() != "cicd_runner" {
		t.Errorf("Expected table name 'cicd_runner', got '%s'", runner.TableName())
	}
}

func TestTableName_Secret(t *testing.T) {
	secret := Secret{}
	if secret.TableName() != "cicd_secret" {
		t.Errorf("Expected table name 'cicd_secret', got '%s'", secret.TableName())
	}
}

func TestTableName_Environment(t *testing.T) {
	env := Environment{}
	if env.TableName() != "cicd_environment" {
		t.Errorf("Expected table name 'cicd_environment', got '%s'", env.TableName())
	}
}

func TestTableName_Deployment(t *testing.T) {
	deployment := Deployment{}
	if deployment.TableName() != "cicd_deployment" {
		t.Errorf("Expected table name 'cicd_deployment', got '%s'", deployment.TableName())
	}
}

func TestTableName_CronSchedule(t *testing.T) {
	cron := CronSchedule{}
	if cron.TableName() != "cicd_cron_schedule" {
		t.Errorf("Expected table name 'cicd_cron_schedule', got '%s'", cron.TableName())
	}
}

func TestTableName_CommitComment(t *testing.T) {
	comment := CommitComment{}
	if comment.TableName() != "commit_comment" {
		t.Errorf("Expected table name 'commit_comment', got '%s'", comment.TableName())
	}
}

func TestTableName_CommitStatus(t *testing.T) {
	status := CommitStatus{}
	if status.TableName() != "commit_status" {
		t.Errorf("Expected table name 'commit_status', got '%s'", status.TableName())
	}
}

func TestTableName_CustomField(t *testing.T) {
	field := CustomField{}
	if field.TableName() != "custom_field" {
		t.Errorf("Expected table name 'custom_field', got '%s'", field.TableName())
	}
}

func TestTableName_IssueCustomField(t *testing.T) {
	field := IssueCustomField{}
	if field.TableName() != "issue_custom_field" {
		t.Errorf("Expected table name 'issue_custom_field', got '%s'", field.TableName())
	}
}

func TestTableName_Epic(t *testing.T) {
	epic := Epic{}
	if epic.TableName() != "epic" {
		t.Errorf("Expected table name 'epic', got '%s'", epic.TableName())
	}
}

func TestTableName_Issue(t *testing.T) {
	issue := Issue{}
	if issue.TableName() != "issue" {
		t.Errorf("Expected table name 'issue', got '%s'", issue.TableName())
	}
}

func TestTableName_IssueIDCounter(t *testing.T) {
	counter := IssueIDCounter{}
	if counter.TableName() != "issue_id" {
		t.Errorf("Expected table name 'issue_id', got '%s'", counter.TableName())
	}
}

func TestTableName_IssueComment(t *testing.T) {
	comment := IssueComment{}
	if comment.TableName() != "issue_comment" {
		t.Errorf("Expected table name 'issue_comment', got '%s'", comment.TableName())
	}
}

func TestTableName_Label(t *testing.T) {
	label := Label{}
	if label.TableName() != "label" {
		t.Errorf("Expected table name 'label', got '%s'", label.TableName())
	}
}

func TestTableName_Milestone(t *testing.T) {
	milestone := Milestone{}
	if milestone.TableName() != "milestone" {
		t.Errorf("Expected table name 'milestone', got '%s'", milestone.TableName())
	}
}

func TestTableName_Priority(t *testing.T) {
	priority := Priority{}
	if priority.TableName() != "priority" {
		t.Errorf("Expected table name 'priority', got '%s'", priority.TableName())
	}
}

func TestTableName_Repository(t *testing.T) {
	repo := Repository{}
	if repo.TableName() != "repository" {
		t.Errorf("Expected table name 'repository', got '%s'", repo.TableName())
	}
}

func TestTableName_RepositoryStatsCache(t *testing.T) {
	cache := RepositoryStatsCache{}
	if cache.TableName() != "repository_stats_cache" {
		t.Errorf("Expected table name 'repository_stats_cache', got '%s'", cache.TableName())
	}
}

func TestTableName_Collaborator(t *testing.T) {
	collab := Collaborator{}
	if collab.TableName() != "collaborator" {
		t.Errorf("Expected table name 'collaborator', got '%s'", collab.TableName())
	}
}

func TestTableName_DeployKey(t *testing.T) {
	key := DeployKey{}
	if key.TableName() != "deploy_key" {
		t.Errorf("Expected table name 'deploy_key', got '%s'", key.TableName())
	}
}

func TestTableName_MergeRequest(t *testing.T) {
	mr := MergeRequest{}
	if mr.TableName() != "merge_request" {
		t.Errorf("Expected table name 'merge_request', got '%s'", mr.TableName())
	}
}

func TestTableName_Notification(t *testing.T) {
	notif := Notification{}
	if notif.TableName() != "notification" {
		t.Errorf("Expected table name 'notification', got '%s'", notif.TableName())
	}
}

func TestTableName_PasswordResetToken(t *testing.T) {
	reset := PasswordResetToken{}
	if reset.TableName() != "password_reset_token" {
		t.Errorf("Expected table name 'password_reset_token', got '%s'", reset.TableName())
	}
}

func TestTableName_Plugin(t *testing.T) {
	plugin := Plugin{}
	if plugin.TableName() != "plugin" {
		t.Errorf("Expected table name 'plugin', got '%s'", plugin.TableName())
	}
}

func TestTableName_Project(t *testing.T) {
	project := Project{}
	if project.TableName() != "project" {
		t.Errorf("Expected table name 'project', got '%s'", project.TableName())
	}
}

func TestTableName_ProjectMember(t *testing.T) {
	member := ProjectMember{}
	if member.TableName() != "project_member" {
		t.Errorf("Expected table name 'project_member', got '%s'", member.TableName())
	}
}

func TestTableName_ReleaseTag(t *testing.T) {
	release := ReleaseTag{}
	if release.TableName() != "release_tag" {
		t.Errorf("Expected table name 'release_tag', got '%s'", release.TableName())
	}
}

func TestTableName_ReleaseAsset(t *testing.T) {
	asset := ReleaseAsset{}
	if asset.TableName() != "release_asset" {
		t.Errorf("Expected table name 'release_asset', got '%s'", asset.TableName())
	}
}

func TestTableName_Review(t *testing.T) {
	review := Review{}
	if review.TableName() != "review" {
		t.Errorf("Expected table name 'review', got '%s'", review.TableName())
	}
}

func TestTableName_ScrumActivity(t *testing.T) {
	activity := ScrumActivity{}
	if activity.TableName() != "scrum_activity" {
		t.Errorf("Expected table name 'scrum_activity', got '%s'", activity.TableName())
	}
}

func TestTableName_Sprint(t *testing.T) {
	sprint := Sprint{}
	if sprint.TableName() != "sprint" {
		t.Errorf("Expected table name 'sprint', got '%s'", sprint.TableName())
	}
}

func TestTableName_SystemSetting(t *testing.T) {
	setting := SystemSetting{}
	if setting.TableName() != "system_settings" {
		t.Errorf("Expected table name 'system_settings', got '%s'", setting.TableName())
	}
}

func TestTableName_Task(t *testing.T) {
	task := Task{}
	if task.TableName() != "task" {
		t.Errorf("Expected table name 'task', got '%s'", task.TableName())
	}
}

func TestTableName_TaskBranch(t *testing.T) {
	branch := TaskBranch{}
	if branch.TableName() != "task_branch" {
		t.Errorf("Expected table name 'task_branch', got '%s'", branch.TableName())
	}
}

func TestTableName_TaskStatus(t *testing.T) {
	status := TaskStatus{}
	if status.TableName() != "task_status" {
		t.Errorf("Expected table name 'task_status', got '%s'", status.TableName())
	}
}

func TestTableName_UserStory(t *testing.T) {
	story := UserStory{}
	if story.TableName() != "user_story" {
		t.Errorf("Expected table name 'user_story', got '%s'", story.TableName())
	}
}

func TestTableName_Webhook(t *testing.T) {
	webhook := Webhook{}
	if webhook.TableName() != "webhooks" {
		t.Errorf("Expected table name 'webhooks', got '%s'", webhook.TableName())
	}
}

func TestTableName_WebhookDelivery(t *testing.T) {
	delivery := WebhookDelivery{}
	if delivery.TableName() != "webhook_deliveries" {
		t.Errorf("Expected table name 'webhook_deliveries', got '%s'", delivery.TableName())
	}
}

func TestTableName_WikiPage(t *testing.T) {
	wiki := WikiPage{}
	if wiki.TableName() != "wiki_page" {
		t.Errorf("Expected table name 'wiki_page', got '%s'", wiki.TableName())
	}
}

func TestTableName_LFSObject(t *testing.T) {
	obj := LFSObject{}
	if obj.TableName() != "lfs_object" {
		t.Errorf("Expected table name 'lfs_object', got '%s'", obj.TableName())
	}
}
