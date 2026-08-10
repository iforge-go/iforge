package service

import (
	"crypto/tls"
	"errors"
	"fmt"

	"github.com/go-ldap/ldap/v3"
)

// LDAPService handles LDAP authentication
type LDAPService struct {
	settingsService *SystemSettingsService
}

// NewLDAPService creates a new LDAPService
func NewLDAPService(settingsService *SystemSettingsService) *LDAPService {
	return &LDAPService{
		settingsService: settingsService,
	}
}

// LDAPUser represents a user from LDAP
type LDAPUser struct {
	Username string
	Email    string
	FullName string
}

// Authenticate authenticates a user against LDAP
func (s *LDAPService) Authenticate(username, password string) (*LDAPUser, error) {
	settings, err := s.settingsService.GetLDAPSettings()
	if err != nil {
		return nil, fmt.Errorf("failed to get LDAP settings: %w", err)
	}

	if !settings.Enabled {
		return nil, errors.New("LDAP authentication is not enabled")
	}

	// Connect to LDAP server
	var conn *ldap.Conn
	if settings.TLS {
		conn, err = ldap.DialTLS("tcp", fmt.Sprintf("%s:%d", settings.Host, settings.Port), &tls.Config{
			InsecureSkipVerify: true,
		})
	} else {
		conn, err = ldap.Dial("tcp", fmt.Sprintf("%s:%d", settings.Host, settings.Port))
	}
	if err != nil {
		return nil, fmt.Errorf("failed to connect to LDAP server: %w", err)
	}
	defer conn.Close()

	// Bind with service account
	if settings.BindDN != "" {
		err = conn.Bind(settings.BindDN, settings.BindPassword)
		if err != nil {
			return nil, fmt.Errorf("failed to bind with service account: %w", err)
		}
	}

	// Search for user
	userFilter := settings.UserFilter
	if userFilter == "" {
		userFilter = "(uid={username})"
	}
	userFilter = replacePlaceholder(userFilter, "{username}", username)

	searchRequest := ldap.NewSearchRequest(
		settings.BaseDN,
		ldap.ScopeWholeSubtree, ldap.NeverDerefAliases, 0, 0, false,
		userFilter,
		[]string{settings.EmailAttr, settings.NameAttr, "uid", "cn"},
		nil,
	)

	sr, err := conn.Search(searchRequest)
	if err != nil {
		return nil, fmt.Errorf("failed to search for user: %w", err)
	}

	if len(sr.Entries) != 1 {
		return nil, errors.New("user not found or multiple users found")
	}

	userDN := sr.Entries[0].DN
	email := sr.Entries[0].GetAttributeValue(settings.EmailAttr)
	fullName := sr.Entries[0].GetAttributeValue(settings.NameAttr)
	if fullName == "" {
		fullName = sr.Entries[0].GetAttributeValue("cn")
	}

	// Bind as user to verify password
	err = conn.Bind(userDN, password)
	if err != nil {
		return nil, errors.New("invalid credentials")
	}

	return &LDAPUser{
		Username: username,
		Email:    email,
		FullName: fullName,
	}, nil
}

// replacePlaceholder replaces a placeholder in a string
func replacePlaceholder(s, placeholder, value string) string {
	if placeholder == "" {
		return s
	}
	result := ""
	for i := 0; i < len(s); i++ {
		if i+len(placeholder) <= len(s) && s[i:i+len(placeholder)] == placeholder {
			result += value
			i += len(placeholder) - 1
		} else {
			result += string(s[i])
		}
	}
	return result
}

// TestConnection tests the LDAP connection
func (s *LDAPService) TestConnection() error {
	settings, err := s.settingsService.GetLDAPSettings()
	if err != nil {
		return fmt.Errorf("failed to get LDAP settings: %w", err)
	}

	// Connect to LDAP server
	var conn *ldap.Conn
	if settings.TLS {
		conn, err = ldap.DialTLS("tcp", fmt.Sprintf("%s:%d", settings.Host, settings.Port), &tls.Config{
			InsecureSkipVerify: true,
		})
	} else {
		conn, err = ldap.Dial("tcp", fmt.Sprintf("%s:%d", settings.Host, settings.Port))
	}
	if err != nil {
		return fmt.Errorf("failed to connect to LDAP server: %w", err)
	}
	defer conn.Close()

	// Bind with service account if provided
	if settings.BindDN != "" {
		err = conn.Bind(settings.BindDN, settings.BindPassword)
		if err != nil {
			return fmt.Errorf("failed to bind with service account: %w", err)
		}
	}

	return nil
}
