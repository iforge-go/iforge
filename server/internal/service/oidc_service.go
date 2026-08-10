package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"sync"
	"time"

	"github.com/coreos/go-oidc/v3/oidc"
	"golang.org/x/oauth2"
)

// OIDCService handles OpenID Connect authentication
type OIDCService struct {
	settingsService *SystemSettingsService
	provider        *oidc.Provider
	verifier        *oidc.IDTokenVerifier
	oauth2Config    *oauth2.Config

	// stateStore keeps pending OIDC state values to validate callbacks.
	// Stored as state -> expiry time. Used when the cookie-based flow is
	// unavailable (e.g., frontend and backend on different ports/origins).
	stateStore map[string]time.Time
	stateMu    sync.Mutex
}

// stateTTL is how long a pending OIDC state remains valid.
const stateTTL = 5 * time.Minute

// NewOIDCService creates a new OIDCService
func NewOIDCService(settingsService *SystemSettingsService) *OIDCService {
	return &OIDCService{
		settingsService: settingsService,
		stateStore:      make(map[string]time.Time),
	}
}

// SaveState stores a pending OIDC state with the configured TTL.
func (s *OIDCService) SaveState(state string) {
	if state == "" {
		return
	}
	s.stateMu.Lock()
	defer s.stateMu.Unlock()
	// Garbage collect expired entries opportunistically
	now := time.Now()
	for k, exp := range s.stateStore {
		if !exp.After(now) {
			delete(s.stateStore, k)
		}
	}
	s.stateStore[state] = now.Add(stateTTL)
}

// ConsumeState validates and removes a pending OIDC state.
// Returns true if the state was valid and not yet consumed/expired.
func (s *OIDCService) ConsumeState(state string) bool {
	if state == "" {
		return false
	}
	s.stateMu.Lock()
	defer s.stateMu.Unlock()
	exp, ok := s.stateStore[state]
	if !ok {
		return false
	}
	delete(s.stateStore, state)
	return exp.After(time.Now())
}

// OIDCSettings represents OIDC configuration
type OIDCSettings struct {
	Enabled      bool   `json:"enabled"`
	Issuer       string `json:"issuer"`
	ClientID     string `json:"clientId"`
	ClientSecret string `json:"clientSecret"`
	RedirectURL  string `json:"redirectUrl"`
}

// OIDCUser represents a user from OIDC provider
type OIDCUser struct {
	Subject       string `json:"sub"`
	Email         string `json:"email"`
	EmailVerified bool   `json:"email_verified"`
	Name          string `json:"name"`
	PreferredName string `json:"preferred_username"`
}

// GetOIDCSettings gets OIDC settings
func (s *OIDCService) GetOIDCSettings() (*OIDCSettings, error) {
	settings, err := s.settingsService.GetAllSettings()
	if err != nil {
		return nil, err
	}

	return &OIDCSettings{
		Enabled:      settings["oidc.enabled"] == "true",
		Issuer:       settings["oidc.issuer"],
		ClientID:     settings["oidc.clientId"],
		ClientSecret: settings["oidc.clientSecret"],
		RedirectURL:  settings["oidc.redirectUrl"],
	}, nil
}

// SetOIDCSettings sets OIDC settings
func (s *OIDCService) SetOIDCSettings(oidcSettings *OIDCSettings) error {
	settings := map[string]string{
		"oidc.enabled":      boolToString(oidcSettings.Enabled),
		"oidc.issuer":       oidcSettings.Issuer,
		"oidc.clientId":     oidcSettings.ClientID,
		"oidc.clientSecret": oidcSettings.ClientSecret,
		"oidc.redirectUrl":  oidcSettings.RedirectURL,
	}

	for key, value := range settings {
		if err := s.settingsService.SetSetting(key, value); err != nil {
			return err
		}
	}

	return nil
}

// Initialize initializes the OIDC provider and verifier
func (s *OIDCService) Initialize() error {
	settings, err := s.GetOIDCSettings()
	if err != nil {
		return fmt.Errorf("failed to get OIDC settings: %w", err)
	}

	if !settings.Enabled {
		return errors.New("OIDC is not enabled")
	}

	if settings.Issuer == "" || settings.ClientID == "" {
		return errors.New("OIDC issuer and client ID are required")
	}

	// Create OIDC provider
	ctx := context.Background()
	provider, err := oidc.NewProvider(ctx, settings.Issuer)
	if err != nil {
		return fmt.Errorf("failed to create OIDC provider: %w", err)
	}

	s.provider = provider
	s.verifier = provider.Verifier(&oidc.Config{
		ClientID: settings.ClientID,
	})

	s.oauth2Config = &oauth2.Config{
		ClientID:     settings.ClientID,
		ClientSecret: settings.ClientSecret,
		RedirectURL:  settings.RedirectURL,
		Endpoint:     provider.Endpoint(),
		Scopes:       []string{oidc.ScopeOpenID, "profile", "email"},
	}

	return nil
}

// GetAuthURL returns the OAuth2 authorization URL
func (s *OIDCService) GetAuthURL(state string) (string, error) {
	if s.oauth2Config == nil {
		if err := s.Initialize(); err != nil {
			return "", err
		}
	}

	return s.oauth2Config.AuthCodeURL(state), nil
}

// Exchange exchanges an authorization code for tokens and returns user info
func (s *OIDCService) Exchange(code string) (*OIDCUser, error) {
	if s.oauth2Config == nil || s.verifier == nil {
		if err := s.Initialize(); err != nil {
			return nil, err
		}
	}

	ctx := context.Background()

	// Exchange code for token
	token, err := s.oauth2Config.Exchange(ctx, code)
	if err != nil {
		return nil, fmt.Errorf("failed to exchange code: %w", err)
	}

	// Extract ID token
	rawIDToken, ok := token.Extra("id_token").(string)
	if !ok {
		return nil, errors.New("no id_token in token response")
	}

	// Verify ID token
	idToken, err := s.verifier.Verify(ctx, rawIDToken)
	if err != nil {
		return nil, fmt.Errorf("failed to verify ID token: %w", err)
	}

	// Extract claims
	var claims OIDCUser
	if err := idToken.Claims(&claims); err != nil {
		return nil, fmt.Errorf("failed to parse claims: %w", err)
	}

	return &claims, nil
}

// TestConnection tests the OIDC connection
func (s *OIDCService) TestConnection() error {
	settings, err := s.GetOIDCSettings()
	if err != nil {
		return fmt.Errorf("failed to get OIDC settings: %w", err)
	}

	if !settings.Enabled {
		return errors.New("OIDC is not enabled")
	}

	// Try to discover the provider
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	_, err = oidc.NewProvider(ctx, settings.Issuer)
	if err != nil {
		return fmt.Errorf("failed to connect to OIDC provider: %w", err)
	}

	return nil
}

// GetUserInfo gets user info from the OIDC provider
func (s *OIDCService) GetUserInfo(accessToken string) (*OIDCUser, error) {
	if s.provider == nil {
		if err := s.Initialize(); err != nil {
			return nil, err
		}
	}

	userInfoEndpoint := s.provider.UserInfoEndpoint()
	if userInfoEndpoint == "" {
		return nil, errors.New("userinfo endpoint not available")
	}

	client := &http.Client{Timeout: 10 * time.Second}
	req, err := http.NewRequest("GET", userInfoEndpoint, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+accessToken)

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to get user info: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("user info request failed with status: %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	var userInfo OIDCUser
	if err := json.Unmarshal(body, &userInfo); err != nil {
		return nil, fmt.Errorf("failed to parse user info: %w", err)
	}

	return &userInfo, nil
}
