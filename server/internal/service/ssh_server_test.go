package service

import (
	"testing"

	"iforge/iforge/internal/event"
	gitsvc "iforge/iforge/internal/git"
)

func TestNewSSHServer(t *testing.T) {
	db := setupTestDB(t)
	defer cleanupTestDB(db)

	sshKeyService := &SSHKeyService{db: db}
	repoService := &RepositoryService{db: db}
	gitClient := &gitsvc.Client{}
	accountService := &AccountService{db: db}
	eventBus := event.NewBus()
	collaboratorService := &CollaboratorService{db: db}
	deployKeyService := &DeployKeyService{db: db}

	server := NewSSHServer(sshKeyService, repoService, gitClient, accountService, eventBus, collaboratorService, deployKeyService)

	if server == nil {
		t.Fatal("NewSSHServer returned nil")
	}

	if server.sshKeyService != sshKeyService {
		t.Error("SSHServer.sshKeyService not set correctly")
	}

	if server.repoService != repoService {
		t.Error("SSHServer.repoService not set correctly")
	}

	if server.gitClient != gitClient {
		t.Error("SSHServer.gitClient not set correctly")
	}

	if server.accountService != accountService {
		t.Error("SSHServer.accountService not set correctly")
	}

	if server.eventBus != eventBus {
		t.Error("SSHServer.eventBus not set correctly")
	}

	if server.collaboratorService != collaboratorService {
		t.Error("SSHServer.collaboratorService not set correctly")
	}

	if server.deployKeyService != deployKeyService {
		t.Error("SSHServer.deployKeyService not set correctly")
	}

	if server.logger == nil {
		t.Error("SSHServer.logger is nil")
	}
}

func TestSSHServer_Constants(t *testing.T) {
	// Verify constant definitions
	if maxSSHConns != 256 {
		t.Errorf("maxSSHConns = %d, want 256", maxSSHConns)
	}

	if sshGitTimeout.Minutes() != 30 {
		t.Errorf("sshGitTimeout = %v, want 30m", sshGitTimeout)
	}
}
