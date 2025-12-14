package gitignore

import (
	"regexp"
	"strings"
)

// Pattern represents a compiled gitignore pattern
type Pattern struct {
	Original string
	Regex    *regexp.Regexp
	IsDir    bool // Pattern ends with /
	HasSlash bool // Pattern contains /
}

// Compile converts a gitignore pattern to a compiled Pattern
func Compile(pattern string) (*Pattern, error) {
	p := &Pattern{
		Original: pattern,
		IsDir:    strings.HasSuffix(pattern, "/"),
		HasSlash: strings.Contains(strings.TrimSuffix(pattern, "/"), "/"),
	}

	regexStr := patternToRegex(pattern)
	regex, err := regexp.Compile(regexStr)
	if err != nil {
		return nil, err
	}
	p.Regex = regex

	return p, nil
}

// patternToRegex converts a gitignore pattern to regex string
func patternToRegex(pattern string) string {
	var result strings.Builder
	i := 0

	for i < len(pattern) {
		ch := pattern[i]

		switch ch {
		case '*':
			if i+1 < len(pattern) && pattern[i+1] == '*' {
				// Handle **
				if i+2 < len(pattern) && pattern[i+2] == '/' {
					// **/ matches zero or more directories
					result.WriteString("(?:.*/)?")
					i += 3
					continue
				} else if i+2 == len(pattern) {
					// ** at end matches everything
					result.WriteString(".*")
					i += 2
					continue
				}
				// ** in middle
				result.WriteString(".*")
				i += 2
			} else {
				// * matches anything except /
				result.WriteString("[^/]*")
				i++
			}
		case '?':
			// ? matches single character except /
			result.WriteString("[^/]")
			i++
		case '[', '.', '+', '^', '$', '{', '}', '(', ')', '|', '\\':
			// Escape regex special characters
			result.WriteByte('\\')
			result.WriteByte(ch)
			i++
		default:
			result.WriteByte(ch)
			i++
		}
	}

	return result.String()
}

// Match checks if a path matches this pattern
func (p *Pattern) Match(relativePath, basename string) bool {
	// Normalize path separators to forward slash
	normalizedPath := strings.ReplaceAll(relativePath, "\\", "/")

	// Handle directory patterns (ending with /)
	if p.IsDir {
		dirPattern := strings.TrimSuffix(p.Original, "/")
		regex := patternToRegex(dirPattern)
		pathRegex, err := regexp.Compile("^" + regex + "(/.*)?$")
		if err != nil {
			return false
		}
		return pathRegex.MatchString(normalizedPath)
	}

	// Pattern with no slash: match against basename or any path component
	if !p.HasSlash {
		regex := patternToRegex(p.Original)
		nameRegex, err := regexp.Compile("^" + regex + "$")
		if err != nil {
			return false
		}

		// Check basename
		if nameRegex.MatchString(basename) {
			return true
		}

		// Check each path component
		parts := strings.Split(normalizedPath, "/")
		for _, part := range parts {
			if nameRegex.MatchString(part) {
				return true
			}
		}

		return false
	}

	// Pattern with slash: match against full relative path
	regex := patternToRegex(p.Original)
	pathRegex, err := regexp.Compile("^" + regex + "$")
	if err != nil {
		return false
	}
	return pathRegex.MatchString(normalizedPath)
}

// Matcher manages multiple patterns
type Matcher struct {
	patterns []*Pattern
}

// NewMatcher creates a new Matcher from pattern strings
func NewMatcher(patterns []string) (*Matcher, error) {
	m := &Matcher{
		patterns: make([]*Pattern, 0, len(patterns)),
	}

	for _, p := range patterns {
		if p == "" {
			continue
		}
		pattern, err := Compile(p)
		if err != nil {
			// Skip invalid patterns
			continue
		}
		m.patterns = append(m.patterns, pattern)
	}

	return m, nil
}

// ShouldIgnore checks if a path should be ignored
func (m *Matcher) ShouldIgnore(relativePath, basename string) bool {
	for _, p := range m.patterns {
		if p.Match(relativePath, basename) {
			return true
		}
	}
	return false
}
