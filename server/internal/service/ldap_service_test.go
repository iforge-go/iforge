package service

import (
	"testing"
)

func TestReplacePlaceholder(t *testing.T) {
	tests := []struct {
		name        string
		input       string
		placeholder string
		value       string
		expected    string
	}{
		{
			name:        "simple replacement",
			input:       "(uid={username})",
			placeholder: "{username}",
			value:       "john",
			expected:    "(uid=john)",
		},
		{
			name:        "multiple occurrences",
			input:       "{username} and {username}",
			placeholder: "{username}",
			value:       "jane",
			expected:    "jane and jane",
		},
		{
			name:        "no match",
			input:       "(uid={email})",
			placeholder: "{username}",
			value:       "john",
			expected:    "(uid={email})",
		},
		{
			name:        "empty placeholder",
			input:       "test string",
			placeholder: "",
			value:       "value",
			expected:    "test string",
		},
		{
			name:        "empty value",
			input:       "(uid={username})",
			placeholder: "{username}",
			value:       "",
			expected:    "(uid=)",
		},
		{
			name:        "complex filter",
			input:       "(&(objectClass=user)(sAMAccountName={username}))",
			placeholder: "{username}",
			value:       "admin",
			expected:    "(&(objectClass=user)(sAMAccountName=admin))",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := replacePlaceholder(tt.input, tt.placeholder, tt.value)
			if result != tt.expected {
				t.Errorf("replacePlaceholder() = %q; want %q", result, tt.expected)
			}
		})
	}
}

func TestLDAPService_NewLDAPService(t *testing.T) {
	settingsService := &SystemSettingsService{}
	service := NewLDAPService(settingsService)

	if service == nil {
		t.Fatal("Expected LDAPService to be created")
	}
	if service.settingsService != settingsService {
		t.Error("Expected settingsService to be set")
	}
}

func TestLDAPUser_Fields(t *testing.T) {
	user := &LDAPUser{
		Username: "john.doe",
		Email:    "john.doe@example.com",
		FullName: "John Doe",
	}

	if user.Username != "john.doe" {
		t.Errorf("Expected Username 'john.doe', got %q", user.Username)
	}
	if user.Email != "john.doe@example.com" {
		t.Errorf("Expected Email 'john.doe@example.com', got %q", user.Email)
	}
	if user.FullName != "John Doe" {
		t.Errorf("Expected FullName 'John Doe', got %q", user.FullName)
	}
}
