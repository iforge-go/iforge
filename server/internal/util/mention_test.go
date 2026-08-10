package util

import (
	"reflect"
	"testing"
)

func TestParseMentions(t *testing.T) {
	tests := []struct {
		name     string
		content  string
		expected []string
	}{
		{
			name:     "no mentions",
			content:  "Hello world",
			expected: nil,
		},
		{
			name:     "single mention",
			content:  "Hello @alice",
			expected: []string{"alice"},
		},
		{
			name:     "multiple mentions",
			content:  "Hi @alice and @bob",
			expected: []string{"alice", "bob"},
		},
		{
			name:     "duplicate mentions",
			content:  "@alice @bob @alice",
			expected: []string{"alice", "bob"},
		},
		{
			name:     "mention with underscore",
			content:  "CC @alice_smith",
			expected: []string{"alice_smith"},
		},
		{
			name:     "mention with hyphen",
			content:  "CC @alice-smith",
			expected: []string{"alice-smith"},
		},
		{
			name:     "mention at start",
			content:  "@alice hello",
			expected: []string{"alice"},
		},
		{
			name:     "mention in middle",
			content:  "Hello @alice world",
			expected: []string{"alice"},
		},
		{
			name:     "mention at end",
			content:  "Hello @alice",
			expected: []string{"alice"},
		},
		{
			name:     "email should not match",
			content:  "Email alice@example.com",
			expected: nil,
		},
		{
			name:     "code path should not match",
			content:  "File file@version",
			expected: nil,
		},
		{
			name:     "mention must start with letter",
			content:  "@123alice should not match",
			expected: nil,
		},
		{
			name:     "multiple mentions with text",
			content:  "Hi @alice, please review. CC @bob and @charlie",
			expected: []string{"alice", "bob", "charlie"},
		},
		{
			name:     "mention after punctuation",
			content:  "Hello! @alice",
			expected: []string{"alice"},
		},
		{
			name:     "mention after newline",
			content:  "Hello\n@alice",
			expected: []string{"alice"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ParseMentions(tt.content)
			if !reflect.DeepEqual(result, tt.expected) {
				t.Errorf("ParseMentions(%q) = %v, want %v", tt.content, result, tt.expected)
			}
		})
	}
}

func BenchmarkParseMentions(b *testing.B) {
	content := "Hi @alice, please review this. CC @bob and @charlie. Thanks @alice!"
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		ParseMentions(content)
	}
}
