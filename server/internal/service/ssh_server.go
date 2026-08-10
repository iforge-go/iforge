package service

import (
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"iforge/iforge/internal/event"
	gitsvc "iforge/iforge/internal/git"
	"iforge/iforge/internal/model"

	"github.com/go-git/go-git/v5/plumbing"
	"golang.org/x/crypto/ssh"
)

// maxSSHConns limits the number of simultaneous connections to prevent Slowloris-style resource exhaustion attacks.
const maxSSHConns = 256

// sshGitTimeout limits the execution time for git-upload-pack/git-receive-pack to prevent slow clone/push operations from blocking goroutines.
const sshGitTimeout = 30 * time.Minute

// SSHServer runs an SSH server that serves git-upload-pack / git-receive-pack
// over SSH, enabling `git clone git@host:port:owner/repo.git`.
type SSHServer struct {
	sshKeyService       *SSHKeyService
	repoService         *RepositoryService
	gitClient           *gitsvc.Client
	accountService      *AccountService
	eventBus            *event.Bus
	collaboratorService *CollaboratorService
	deployKeyService    *DeployKeyService
	logger              *gitsvc.Logger
}

// NewSSHServer creates a new SSHServer.
func NewSSHServer(sshKeyService *SSHKeyService, repoService *RepositoryService, gitClient *gitsvc.Client, accountService *AccountService, eventBus *event.Bus, collaboratorService *CollaboratorService, deployKeyService *DeployKeyService) *SSHServer {
	return &SSHServer{
		sshKeyService:       sshKeyService,
		repoService:         repoService,
		gitClient:           gitClient,
		accountService:      accountService,
		eventBus:            eventBus,
		collaboratorService: collaboratorService,
		deployKeyService:    deployKeyService,
		logger:              gitsvc.GetLogger(),
	}
}

// Start loads (or generates) the host key at hostKeyPath and listens for SSH
// connections on the given port. Blocks until the listener closes.
func (s *SSHServer) Start(hostKeyPath string, port int) error {
	hostSigner, err := s.loadOrCreateHostKey(hostKeyPath)
	if err != nil {
		return fmt.Errorf("failed to load host key: %w", err)
	}

	config := &ssh.ServerConfig{
		PublicKeyCallback: func(conn ssh.ConnMetadata, key ssh.PublicKey) (*ssh.Permissions, error) {
			pubKeyStr := strings.TrimRight(string(ssh.MarshalAuthorizedKey(key)), "\n")
			// First try to find as user SSH key
			account, err := s.sshKeyService.GetAccountByPublicKey(pubKeyStr)
			if err == nil {
				return &ssh.Permissions{
					Extensions: map[string]string{
						"username":  account.UserName,
						"is_admin":  strconv.FormatBool(account.IsAdmin),
						"auth_type": "user",
					},
				}, nil
			}
			// Try to find as deploy key
			deployKey, err := s.deployKeyService.GetDeployKeyByPublicKey(pubKeyStr)
			if err == nil {
				return &ssh.Permissions{
					Extensions: map[string]string{
						"username":    deployKey.UserName,
						"repository":  deployKey.RepositoryName,
						"allow_write": strconv.FormatBool(deployKey.AllowWrite),
						"auth_type":   "deploy_key",
					},
				}, nil
			}
			s.logger.Warn("SSH auth failed", map[string]interface{}{
				"user":    conn.User(),
				"address": conn.RemoteAddr().String(),
			})
			return nil, fmt.Errorf("unknown public key")
		},
	}
	config.AddHostKey(hostSigner)

	addr := ":" + strconv.Itoa(port)
	listener, err := net.Listen("tcp", addr)
	if err != nil {
		return fmt.Errorf("failed to listen on %s: %w", addr, err)
	}

	s.logger.Info("SSH server listening", map[string]interface{}{
		"addr": addr,
	})

	// Use semaphore to limit concurrent connections and prevent resource exhaustion
	sem := make(chan struct{}, maxSSHConns)
	for {
		conn, err := listener.Accept()
		if err != nil {
			return fmt.Errorf("listener accept failed: %w", err)
		}
		select {
		case sem <- struct{}{}:
			go func() {
				defer func() { <-sem }()
				s.handleConnection(conn, config)
			}()
		default:
			// Reject connection immediately when limit reached to prevent infinite queuing
			s.logger.Warn("SSH connection rejected: too many connections", map[string]interface{}{
				"address": conn.RemoteAddr().String(),
			})
			conn.Close()
		}
	}
}

func (s *SSHServer) handleConnection(nconn net.Conn, config *ssh.ServerConfig) {
	defer nconn.Close()
	conn, chans, reqs, err := ssh.NewServerConn(nconn, config)
	if err != nil {
		s.logger.Warn("SSH handshake failed", map[string]interface{}{
			"address": nconn.RemoteAddr().String(),
			"error":   err.Error(),
		})
		return
	}
	defer conn.Close()

	go ssh.DiscardRequests(reqs)

	for newChannel := range chans {
		if newChannel.ChannelType() != "session" {
			newChannel.Reject(ssh.UnknownChannelType, "only session channel is supported")
			continue
		}
		go s.handleChannel(newChannel, conn)
	}
}

func (s *SSHServer) handleChannel(newChannel ssh.NewChannel, conn *ssh.ServerConn) {
	channel, requests, err := newChannel.Accept()
	if err != nil {
		s.logger.Warn("Could not accept channel", map[string]interface{}{
			"error": err.Error(),
		})
		return
	}
	defer channel.Close()

	for req := range requests {
		switch req.Type {
		case "exec":
			// Exec payload format: first 4 bytes are uint32 length prefix, followed by command string.
			// Validate length to prevent slice out-of-bounds panic from malicious short payloads.
			if len(req.Payload) < 4 {
				fmt.Fprintf(channel.Stderr(), "invalid exec payload\n")
				if req.WantReply {
					req.Reply(false, nil)
				}
				channel.Close()
				continue
			}
			cmdStr := string(req.Payload[4:])
			go func(req *ssh.Request) {
				exitCode := s.handleExec(channel, conn, cmdStr)
				if req.WantReply {
					req.Reply(true, nil)
				}
				// Send exit-status message so the client sees the real code.
				exitPayload := ssh.Marshal(struct{ Code uint32 }{uint32(exitCode)})
				channel.SendRequest("exit-status", false, exitPayload)
				channel.Close()
			}(req)
		case "shell":
			// No interactive shell; reply with a friendly banner.
			if req.WantReply {
				req.Reply(true, nil)
			}
			username := conn.Permissions.Extensions["username"]
			fmt.Fprintf(channel, "Hi %s! You've successfully authenticated, but iforge does not provide shell access.\r\n", username)
			exitPayload := ssh.Marshal(struct{ Code uint32 }{0})
			channel.SendRequest("exit-status", false, exitPayload)
			channel.Close()
		default:
			if req.WantReply {
				req.Reply(false, nil)
			}
		}
	}
}

// handleExec parses the git command, checks permissions, and runs the git
// subprocess bridged to the SSH channel. Returns the process exit code.
func (s *SSHServer) handleExec(channel ssh.Channel, conn *ssh.ServerConn, cmdStr string) int {
	parts := splitShellArgs(cmdStr)
	if len(parts) == 0 {
		fmt.Fprintf(channel.Stderr(), "no command provided\n")
		return 1
	}

	gitCmd := parts[0]
	if gitCmd != "git-upload-pack" && gitCmd != "git-receive-pack" {
		// Some clients send "git upload-pack ..." as two tokens.
		if len(parts) >= 2 && gitCmd == "git" && (parts[1] == "upload-pack" || parts[1] == "receive-pack") {
			gitCmd = "git-" + parts[1]
			parts = parts[1:]
		} else {
			fmt.Fprintf(channel.Stderr(), "unsupported command: %s\n", gitCmd)
			return 1
		}
	}
	if len(parts) < 2 {
		fmt.Fprintf(channel.Stderr(), "missing repository path\n")
		return 1
	}

	repoPath := strings.TrimSuffix(parts[1], ".git")
	owner, repoName, ok := parseRepoPath(repoPath)
	if !ok {
		fmt.Fprintf(channel.Stderr(), "invalid repository path: %s\n", repoPath)
		return 1
	}

	repo, err := s.repoService.GetRepository(owner, repoName)
	if err != nil {
		fmt.Fprintf(channel.Stderr(), "repository not found: %s/%s\n", owner, repoName)
		return 1
	}

	username := conn.Permissions.Extensions["username"]
	isAdmin := conn.Permissions.Extensions["is_admin"] == "true"
	authType := conn.Permissions.Extensions["auth_type"]

	// Deploy key authentication
	if authType == "deploy_key" {
		allowWrite := conn.Permissions.Extensions["allow_write"] == "true"
		deployRepo := conn.Permissions.Extensions["repository"]

		// Check if deploy key is for this repository
		if deployRepo != repo.RepositoryName {
			fmt.Fprintf(channel.Stderr(), "permission denied: deploy key is not for this repository\n")
			return 1
		}

		// Check write permission
		if gitCmd == "git-receive-pack" && !allowWrite {
			fmt.Fprintf(channel.Stderr(), "permission denied: deploy key does not have write access\n")
			return 1
		}
	} else {
		// User authentication
		if gitCmd == "git-receive-pack" {
			if !s.canPush(username, isAdmin, owner, repo, "") {
				fmt.Fprintf(channel.Stderr(), "permission denied: you cannot push to %s/%s\n", owner, repoName)
				return 1
			}
		} else {
			if !s.canPull(username, isAdmin, owner, repo) {
				fmt.Fprintf(channel.Stderr(), "permission denied: you cannot access %s/%s\n", owner, repoName)
				return 1
			}
		}
	}

	diskPath := s.gitClient.RepositoryPath(owner, repoName)
	if _, err := os.Stat(diskPath); os.IsNotExist(err) {
		fmt.Fprintf(channel.Stderr(), "repository not found on disk\n")
		return 1
	}

	s.logger.Info("SSH git command", map[string]interface{}{
		"command": gitCmd,
		"repo":    owner + "/" + repoName,
		"user":    username,
	})

	subCmd := strings.TrimPrefix(gitCmd, "git-")
	// Use context timeout to prevent slow clone/push from blocking goroutines
	ctx, cancel := context.WithTimeout(context.Background(), sshGitTimeout)
	defer cancel()
	cmd := exec.CommandContext(ctx, "git", subCmd, diskPath)
	cmd.Stdin = channel
	cmd.Stdout = channel
	cmd.Stderr = channel.Stderr()

	var oldRefs map[string]string
	if gitCmd == "git-receive-pack" {
		oldRefs = s.snapshotRefs(owner, repoName)
	}

	if err := cmd.Run(); err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			code := exitErr.ExitCode()
			if code < 0 {
				code = 1
			}
			if gitCmd == "git-receive-pack" {
				s.postPush(owner, repoName, username, oldRefs)
			}
			return code
		}
		fmt.Fprintf(channel.Stderr(), "git command failed: %v\n", err)
		return 1
	}

	if gitCmd == "git-receive-pack" {
		s.postPush(owner, repoName, username, oldRefs)
	}
	return 0
}

func (s *SSHServer) snapshotRefs(owner, repoName string) map[string]string {
	r, err := s.gitClient.OpenRepository(owner, repoName)
	if err != nil {
		return map[string]string{}
	}
	refs := make(map[string]string)
	iter, err := r.References()
	if err != nil {
		return refs
	}
	iter.ForEach(func(ref *plumbing.Reference) error {
		refs[ref.Name().String()] = ref.Hash().String()
		return nil
	})
	return refs
}

func diffRefs(oldRefs, newRefs map[string]string) []gitsvc.RefUpdate {
	var updates []gitsvc.RefUpdate
	for refName, newSHA := range newRefs {
		oldSHA, existed := oldRefs[refName]
		if !existed {
			oldSHA = "0000000000000000000000000000000000000000"
		}
		if oldSHA == newSHA {
			continue
		}
		branchName := strings.TrimPrefix(refName, "refs/heads/")
		branchName = strings.TrimPrefix(branchName, "refs/tags/")
		updates = append(updates, gitsvc.RefUpdate{
			RefName:     refName,
			BranchName:  branchName,
			OldSHA:      oldSHA,
			NewSHA:      newSHA,
			IsNewBranch: !existed,
			IsDeleted:   newSHA == "0000000000000000000000000000000000000000",
		})
	}
	for refName, oldSHA := range oldRefs {
		if _, exists := newRefs[refName]; !exists {
			branchName := strings.TrimPrefix(refName, "refs/heads/")
			branchName = strings.TrimPrefix(branchName, "refs/tags/")
			updates = append(updates, gitsvc.RefUpdate{
				RefName:    refName,
				BranchName: branchName,
				OldSHA:     oldSHA,
				NewSHA:     "0000000000000000000000000000000000000000",
				IsDeleted:  true,
			})
		}
	}
	return updates
}

// postPush mirrors the cache-invalidation and activity recording done by the
// HTTP receive-pack handler (see git_http_handler.go receivePack).
func (s *SSHServer) postPush(owner, repoName, pusherName string, oldRefs map[string]string) {
	// Invalidate all caches for this repository (per-repo, not global)
	gitsvc.InvalidateRepoCaches(owner, repoName)

	newRefs := s.snapshotRefs(owner, repoName)
	refUpdates := diffRefs(oldRefs, newRefs)

	var additionalInfo *string
	if s.gitClient != nil {
		for i, update := range refUpdates {
			if update.IsDeleted {
				refUpdates[i].Commits = []gitsvc.PushCommitInfo{}
				continue
			}
			commits, err := s.gitClient.ListCommitsBetween(owner, repoName, update.OldSHA, update.NewSHA, 50)
			if err == nil {
				refUpdates[i].Commits = commits
			}
		}
		if len(refUpdates) > 0 {
			pushDetail := gitsvc.PushDetail{RefUpdates: refUpdates}
			if jsonBytes, err := json.Marshal(pushDetail); err == nil {
				jsonStr := string(jsonBytes)
				additionalInfo = &jsonStr
			}
		}
	}

	// Publish push_batch event: ActivitySubscriber records "push" activity.
	// WebhookSubscriber does not subscribe to this event type (SSH push has
	// no webhook delivery historically); only ActivitySubscriber handles it.
	if s.eventBus != nil {
		var addInfoStr string
		if additionalInfo != nil {
			addInfoStr = *additionalInfo
		}
		repo := &model.Repository{UserName: owner, RepositoryName: repoName}
		sender := &model.Account{UserName: pusherName}
		s.eventBus.Publish(event.NewPushBatchEvent(repo, sender, addInfoStr))

		// Publish branch_pushed events for Scrum-layer subscribers (auto-status
		// transition + commit activity logging). Mirrors the HTTP receivePack
		// handler: one event per branch ref update.
		repoFullName := owner + "/" + repoName
		for _, update := range refUpdates {
			if update.IsDeleted {
				continue
			}
			// Branch push: publish branch_pushed event (Scrum subscriber)
			if strings.HasPrefix(update.RefName, "refs/heads/") {
				s.eventBus.Publish(event.NewBranchPushedEvent(
					repo, sender,
					repoFullName, update.BranchName, pusherName,
					update.Commits, update.IsNewBranch,
				))
				continue
			}
			// Tag push: publish tag_pushed event (CICDSubscriber subscribes, triggers release pipeline)
			if strings.HasPrefix(update.RefName, "refs/tags/") {
				s.eventBus.Publish(event.NewTagPushedEvent(
					repo, sender,
					repoFullName, update.BranchName, pusherName,
					update.Commits, update.IsNewBranch,
				))
			}
		}
	}
	s.logger.Info("SSH push completed", map[string]interface{}{
		"repo": owner + "/" + repoName,
		"user": pusherName,
	})
}

func (s *SSHServer) canPush(username string, isAdmin bool, owner string, repo *model.Repository, branch string) bool {
	if username == owner {
		return true
	}
	if isAdmin {
		return true
	}
	role, err := s.collaboratorService.GetCollaboratorRole(owner, repo.RepositoryName, username)
	if err != nil {
		return false
	}
	// Write access requires admin or member (was previously "ADMIN"/"WRITE" string
	// literal which mismatched the canonical DEVELOPER constant — bug fix).
	hasWriteAccess := model.HasRoleAtLeast(role, model.RoleMember)
	if !hasWriteAccess {
		return false
	}
	// Check branch protection
	if branch != "" {
		canPush, err := s.repoService.CanPushToBranch(owner, repo.RepositoryName, branch, username)
		if err != nil || !canPush {
			return false
		}
	}
	return true
}

func (s *SSHServer) canPull(username string, isAdmin bool, owner string, repo *model.Repository) bool {
	if !repo.IsPrivate {
		return true
	}
	if username == owner {
		return true
	}
	if isAdmin {
		return true
	}
	role, err := s.collaboratorService.GetCollaboratorRole(owner, repo.RepositoryName, username)
	if err != nil {
		return false
	}
	// Read access requires at least viewer (was previously "ADMIN"/"WRITE"/"READ"
	// string literal — bug fix, now uses canonical role constants).
	return model.HasRoleAtLeast(role, model.RoleViewer)
}

// loadOrCreateHostKey loads an ed25519 host key from path, generating and
// persisting a new one if the file does not exist.
func (s *SSHServer) loadOrCreateHostKey(path string) (ssh.Signer, error) {
	if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		return nil, err
	}

	data, err := os.ReadFile(path)
	if err == nil {
		signer, err := ssh.ParsePrivateKey(data)
		if err != nil {
			return nil, fmt.Errorf("failed to parse host key: %w", err)
		}
		return signer, nil
	}

	if !os.IsNotExist(err) {
		return nil, err
	}

	pub, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		return nil, fmt.Errorf("failed to generate host key: %w", err)
	}
	_ = pub

	pemBlock, err := ssh.MarshalPrivateKey(priv, "")
	if err != nil {
		return nil, fmt.Errorf("failed to marshal host key: %w", err)
	}
	if err := os.WriteFile(path, pem.EncodeToMemory(pemBlock), 0600); err != nil {
		return nil, fmt.Errorf("failed to write host key: %w", err)
	}
	s.logger.Info("Generated new SSH host key", map[string]interface{}{
		"path": path,
	})

	signer, err := ssh.ParsePrivateKey(pem.EncodeToMemory(pemBlock))
	if err != nil {
		return nil, err
	}
	return signer, nil
}

// parseRepoPath splits "owner/repo" into parts and validates that neither
// segment contains path traversal or invalid characters.
func parseRepoPath(p string) (owner, repo string, ok bool) {
	p = strings.TrimSpace(p)
	p = strings.Trim(p, "/'")
	p = strings.Trim(p, "\"")
	if p == "" || strings.Contains(p, "..") || strings.Contains(p, "\\") {
		return "", "", false
	}
	parts := strings.Split(p, "/")
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		return "", "", false
	}
	owner, repo = parts[0], parts[1]
	if !isValidName(owner) || !isValidName(repo) {
		return "", "", false
	}
	return owner, repo, true
}

func isValidName(name string) bool {
	if name == "" {
		return false
	}
	for _, r := range name {
		switch {
		case r >= 'a' && r <= 'z':
		case r >= 'A' && r <= 'Z':
		case r >= '0' && r <= '9':
		case r == '_' || r == '-' || r == '.':
		default:
			return false
		}
	}
	return true
}

// splitShellArgs splits a command string the way an SSH server would, honoring
// single and double quotes. This mirrors OpenSSH's handling of the "exec"
// request payload.
func splitShellArgs(s string) []string {
	var args []string
	var current strings.Builder
	inSingle, inDouble := false, false
	active := false

	for i := 0; i < len(s); i++ {
		c := s[i]
		switch {
		case c == '\'' && !inDouble:
			inSingle = !inSingle
			active = true
		case c == '"' && !inSingle:
			inDouble = !inDouble
			active = true
		case (c == ' ' || c == '\t' || c == '\n') && !inSingle && !inDouble:
			if active {
				args = append(args, current.String())
				current.Reset()
				active = false
			}
		case c == '\\' && !inSingle && i+1 < len(s):
			next := s[i+1]
			if next == '\'' || next == '"' || next == '\\' || next == ' ' {
				current.WriteByte(next)
				active = true
				i++
			} else {
				current.WriteByte(c)
				active = true
			}
		default:
			current.WriteByte(c)
			active = true
		}
	}
	if active {
		args = append(args, current.String())
	}
	return args
}
