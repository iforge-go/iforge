package util

import "regexp"

// mentionRegex matches @username (letter first, then letters/digits/_/-).
// The leading non-word boundary (?:^|[^a-zA-Z0-9_]) prevents matching emails
// (foo@bar.com) and code paths (file@version).
var mentionRegex = regexp.MustCompile(`(?:^|[^a-zA-Z0-9_])@([a-zA-Z][a-zA-Z0-9_-]*)`)

// ParseMentions extracts all @mentioned usernames from content (deduplicated,
// preserving first-seen order).
// Example: "Hi @alice and @bob, cc @alice" -> ["alice", "bob"]
func ParseMentions(content string) []string {
	matches := mentionRegex.FindAllStringSubmatch(content, -1)
	if len(matches) == 0 {
		return nil
	}
	seen := make(map[string]bool)
	var result []string
	for _, m := range matches {
		username := m[1]
		if !seen[username] {
			seen[username] = true
			result = append(result, username)
		}
	}
	return result
}
