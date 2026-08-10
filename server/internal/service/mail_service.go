package service

import (
	"bytes"
	"crypto/tls"
	"fmt"
	"html/template"
	"net/smtp"
	"strings"
	"sync"
	"time"
)

// SMTPConfig represents SMTP configuration
type SMTPConfig struct {
	Host       string
	Port       int
	Username   string
	Password   string
	From       string
	UseTLS     bool
	UseSSL     bool
	AuthMethod string // "plain", "login", "cram-md5"
}

// EmailMessage represents an email message
type EmailMessage struct {
	To      []string
	Subject string
	Body    string
	IsHTML  bool
}

// MailService handles email sending
type MailService struct {
	config     *SMTPConfig
	mailQueue  chan *EmailMessage
	wg         sync.WaitGroup
	stopChan   chan struct{}
	maxRetries int
}

// NewMailService creates a new MailService
func NewMailService(config *SMTPConfig) *MailService {
	service := &MailService{
		config:     config,
		mailQueue:  make(chan *EmailMessage, 100),
		stopChan:   make(chan struct{}),
		maxRetries: 3,
	}

	// Start background worker
	service.wg.Add(1)
	go service.worker()

	return service
}

// UpdateConfig updates SMTP configuration
func (s *MailService) UpdateConfig(config *SMTPConfig) {
	s.config = config
}

// SendMail sends an email immediately (synchronous)
func (s *MailService) SendMail(msg *EmailMessage) error {
	if s.config == nil || s.config.Host == "" {
		return fmt.Errorf("SMTP not configured")
	}

	// Build email content
	var body strings.Builder
	body.WriteString(fmt.Sprintf("From: %s\r\n", s.config.From))
	body.WriteString(fmt.Sprintf("To: %s\r\n", strings.Join(msg.To, ", ")))
	body.WriteString(fmt.Sprintf("Subject: %s\r\n", msg.Subject))
	body.WriteString("MIME-Version: 1.0\r\n")

	if msg.IsHTML {
		body.WriteString("Content-Type: text/html; charset=UTF-8\r\n")
	} else {
		body.WriteString("Content-Type: text/plain; charset=UTF-8\r\n")
	}

	body.WriteString("\r\n")
	body.WriteString(msg.Body)

	// Connect to SMTP server
	addr := fmt.Sprintf("%s:%d", s.config.Host, s.config.Port)

	var auth smtp.Auth
	if s.config.Username != "" {
		switch s.config.AuthMethod {
		case "plain":
			auth = smtp.PlainAuth("", s.config.Username, s.config.Password, s.config.Host)
		case "login":
			// Login auth is not natively supported, use Plain as fallback
			auth = smtp.PlainAuth("", s.config.Username, s.config.Password, s.config.Host)
		case "cram-md5":
			auth = smtp.CRAMMD5Auth(s.config.Username, s.config.Password)
		default:
			auth = smtp.PlainAuth("", s.config.Username, s.config.Password, s.config.Host)
		}
	}

	var err error
	if s.config.UseTLS || s.config.UseSSL {
		err = s.sendWithTLS(addr, auth, s.config.From, msg.To, []byte(body.String()))
	} else {
		err = smtp.SendMail(addr, auth, s.config.From, msg.To, []byte(body.String()))
	}

	return err
}

// sendWithTLS sends email with TLS/SSL
func (s *MailService) sendWithTLS(addr string, auth smtp.Auth, from string, to []string, msg []byte) error {
	tlsConfig := &tls.Config{
		InsecureSkipVerify: false,
		ServerName:         s.config.Host,
	}

	conn, err := tls.Dial("tcp", addr, tlsConfig)
	if err != nil {
		return fmt.Errorf("failed to connect with TLS: %w", err)
	}
	defer conn.Close()

	client, err := smtp.NewClient(conn, s.config.Host)
	if err != nil {
		return fmt.Errorf("failed to create SMTP client: %w", err)
	}
	defer client.Close()

	if auth != nil {
		if err = client.Auth(auth); err != nil {
			return fmt.Errorf("authentication failed: %w", err)
		}
	}

	if err = client.Mail(from); err != nil {
		return fmt.Errorf("failed to set sender: %w", err)
	}

	for _, addr := range to {
		if err = client.Rcpt(addr); err != nil {
			return fmt.Errorf("failed to set recipient %s: %w", addr, err)
		}
	}

	w, err := client.Data()
	if err != nil {
		return fmt.Errorf("failed to start data: %w", err)
	}

	_, err = w.Write(msg)
	if err != nil {
		return fmt.Errorf("failed to write message: %w", err)
	}

	err = w.Close()
	if err != nil {
		return fmt.Errorf("failed to close data writer: %w", err)
	}

	return client.Quit()
}

// QueueMail adds an email to the queue for async sending
func (s *MailService) QueueMail(msg *EmailMessage) {
	select {
	case s.mailQueue <- msg:
	default:
		// Queue full: wait up to 1s to avoid unbounded goroutine growth (OOM).
		// If still full after 1s, drop the email with a warning.
		timer := time.NewTimer(1 * time.Second)
		defer timer.Stop()
		select {
		case s.mailQueue <- msg:
		case <-timer.C:
			fmt.Printf("[warn] mail queue full, dropping email to %s (subject: %s)\n",
				strings.Join(msg.To, ","), msg.Subject)
		}
	}
}

// worker processes the mail queue
func (s *MailService) worker() {
	defer s.wg.Done()

	for {
		select {
		case msg := <-s.mailQueue:
			s.sendWithRetry(msg)
		case <-s.stopChan:
			return
		}
	}
}

// sendWithRetry sends email with retry logic
func (s *MailService) sendWithRetry(msg *EmailMessage) {
	for i := 0; i < s.maxRetries; i++ {
		err := s.SendMail(msg)
		if err == nil {
			return
		}

		// Wait before retry (exponential backoff)
		if i < s.maxRetries-1 {
			time.Sleep(time.Duration(i+1) * time.Second)
		}
	}
}

// Stop stops the mail service
func (s *MailService) Stop() {
	close(s.stopChan)
	s.wg.Wait()
}

// TestConnection tests SMTP connection
func (s *MailService) TestConnection() error {
	if s.config == nil || s.config.Host == "" {
		return fmt.Errorf("SMTP not configured")
	}

	addr := fmt.Sprintf("%s:%d", s.config.Host, s.config.Port)

	var auth smtp.Auth
	if s.config.Username != "" {
		auth = smtp.PlainAuth("", s.config.Username, s.config.Password, s.config.Host)
	}

	var c *smtp.Client
	var err error

	if s.config.UseTLS || s.config.UseSSL {
		tlsConfig := &tls.Config{
			InsecureSkipVerify: false,
			ServerName:         s.config.Host,
		}

		conn, err := tls.Dial("tcp", addr, tlsConfig)
		if err != nil {
			return fmt.Errorf("failed to connect with TLS: %w", err)
		}
		defer conn.Close()

		c, err = smtp.NewClient(conn, s.config.Host)
		if err != nil {
			return fmt.Errorf("failed to create SMTP client: %w", err)
		}
	} else {
		c, err = smtp.Dial(addr)
		if err != nil {
			return fmt.Errorf("failed to connect: %w", err)
		}
	}
	defer c.Close()

	if auth != nil {
		if err = c.Auth(auth); err != nil {
			return fmt.Errorf("authentication failed: %w", err)
		}
	}

	return c.Quit()
}

// renderTemplate renders an HTML email template with the given data.
func renderTemplate(tmpl string, data interface{}) string {
	t, err := template.New("email").Parse(tmpl)
	if err != nil {
		return ""
	}
	var buf bytes.Buffer
	if err := t.Execute(&buf, data); err != nil {
		return ""
	}
	return buf.String()
}

const issueTemplate = `<h2>New Issue Created</h2>
<p><strong>Repository:</strong> {{.Owner}}/{{.Repo}}</p>
<p><strong>Issue:</strong> #{{.IssueID}}</p>
<p><strong>Title:</strong> {{.Title}}</p>
<p><a href="{{.BaseURL}}/{{.Owner}}/{{.Repo}}/issues/{{.IssueID}}">View Issue</a></p>`

const commentTemplate = `<h2>New Comment</h2>
<p><strong>Repository:</strong> {{.Owner}}/{{.Repo}}</p>
<p><strong>Issue:</strong> #{{.IssueID}}</p>
<p><strong>Author:</strong> {{.Author}}</p>
<p><strong>Comment:</strong></p>
<blockquote>{{.Content}}</blockquote>
<p><a href="{{.BaseURL}}/{{.Owner}}/{{.Repo}}/issues/{{.IssueID}}">View Comment</a></p>`

const mentionTemplate = `<h2>You were mentioned</h2>
<p><strong>Repository:</strong> {{.Owner}}/{{.Repo}}</p>
<p><strong>Issue:</strong> #{{.IssueID}}</p>
<p><strong>Mentioned by:</strong> {{.Mentioner}}</p>
<p><a href="{{.BaseURL}}/{{.Owner}}/{{.Repo}}/issues/{{.IssueID}}">View Issue</a></p>`

const mergeRequestTemplate = `<h2>New Merge Request Created</h2>
<p><strong>Repository:</strong> {{.Owner}}/{{.Repo}}</p>
<p><strong>Merge Request:</strong> #{{.MRID}}</p>
<p><strong>Title:</strong> {{.Title}}</p>
<p><a href="{{.BaseURL}}/{{.Owner}}/{{.Repo}}/merge-requests/{{.MRID}}">View Merge Request</a></p>`

const pipelineTemplate = `<h2>Pipeline {{.StatusText}}</h2>
<p><strong>Repository:</strong> {{.Owner}}/{{.Repo}}</p>
<p><strong>Pipeline:</strong> #{{.PipelineID}}</p>
<p><strong>Branch:</strong> {{.Branch}}</p>
<p><strong>Commit:</strong> {{.Commit}}</p>
<p><strong>Commit Message:</strong> {{.Message}}</p>
<p><strong>Status:</strong> {{.StatusText}}</p>
<p><a href="{{.BaseURL}}/{{.Owner}}/{{.Repo}}/pipelines/{{.PipelineID}}">View Pipeline</a></p>`

// SendIssueNotification sends notification for issue creation
func (s *MailService) SendIssueNotification(to []string, owner, repo string, issueID int, title string) {
	subject := fmt.Sprintf("[%s/%s] Issue #%d: %s", owner, repo, issueID, title)
	body := renderTemplate(issueTemplate, map[string]interface{}{
		"Owner": owner, "Repo": repo, "IssueID": issueID, "Title": title,
		"BaseURL": s.baseURL(),
	})

	msg := &EmailMessage{To: to, Subject: subject, Body: body, IsHTML: true}
	s.QueueMail(msg)
}

// SendCommentNotification sends notification for comment
func (s *MailService) SendCommentNotification(to []string, owner, repo string, issueID int, commentAuthor, commentContent string) {
	subject := fmt.Sprintf("Re: [%s/%s] Issue #%d", owner, repo, issueID)
	body := renderTemplate(commentTemplate, map[string]interface{}{
		"Owner": owner, "Repo": repo, "IssueID": issueID,
		"Author": commentAuthor, "Content": commentContent,
		"BaseURL": s.baseURL(),
	})

	msg := &EmailMessage{To: to, Subject: subject, Body: body, IsHTML: true}
	s.QueueMail(msg)
}

// SendMentionNotification sends notification for @mention
func (s *MailService) SendMentionNotification(to []string, owner, repo string, issueID int, mentioner string) {
	subject := fmt.Sprintf("[%s/%s] You were mentioned in #%d", owner, repo, issueID)
	body := renderTemplate(mentionTemplate, map[string]interface{}{
		"Owner": owner, "Repo": repo, "IssueID": issueID, "Mentioner": mentioner,
		"BaseURL": s.baseURL(),
	})

	msg := &EmailMessage{To: to, Subject: subject, Body: body, IsHTML: true}
	s.QueueMail(msg)
}

// SendMergeRequestNotification sends notification for MR creation
func (s *MailService) SendMergeRequestNotification(to []string, owner, repo string, mrID int, title string) {
	subject := fmt.Sprintf("[%s/%s] Merge Request #%d: %s", owner, repo, mrID, title)
	body := renderTemplate(mergeRequestTemplate, map[string]interface{}{
		"Owner": owner, "Repo": repo, "MRID": mrID, "Title": title,
		"BaseURL": s.baseURL(),
	})

	msg := &EmailMessage{To: to, Subject: subject, Body: body, IsHTML: true}
	s.QueueMail(msg)
}

// SendPipelineNotification sends notification for pipeline completion (success/failed)
// status: "success" or "failed"
func (s *MailService) SendPipelineNotification(to []string, owner, repo string, pipelineID int, branch, commit, status, message string) {
	statusText := "succeeded"
	if status == "failed" {
		statusText = "failed"
	}
	subject := fmt.Sprintf("[%s/%s] Pipeline #%d %s", owner, repo, pipelineID, statusText)
	body := renderTemplate(pipelineTemplate, map[string]interface{}{
		"Owner": owner, "Repo": repo, "PipelineID": pipelineID,
		"Branch": branch, "Commit": commit, "Message": message,
		"StatusText": statusText,
		"BaseURL":    s.baseURL(),
	})

	msg := &EmailMessage{To: to, Subject: subject, Body: body, IsHTML: true}
	s.QueueMail(msg)
}

// baseURL returns the configured base URL for email links.
func (s *MailService) baseURL() string {
	if s.config != nil && s.config.From != "" {
		// Use a sensible default; in production this should come from SystemSettings.
		return "http://localhost:3001"
	}
	return "http://localhost:3001"
}
