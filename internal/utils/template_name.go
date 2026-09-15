package utils

import (
	"strings"
	"unicode"
	"unicode/utf8"
)

// SanitizeTemplateName sanitizes a rendered folder or file name and drops redundant segments.
func SanitizeTemplateName(name string) string {
	return CollapseTemplateName(SanitizeFilename(name))
}

// CollapseTemplateName drops empty, placeholder-year, duplicate, and prefix-overlapped " - " segments.
func CollapseTemplateName(name string) string {
	parts := strings.Split(name, " - ")
	cleaned := make([]string, 0, len(parts))

	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" || isPlaceholderYear(part) {
			continue
		}

		cleaned = appendCollapsedPart(cleaned, part)
	}

	return strings.Join(cleaned, " - ")
}

func appendCollapsedPart(cleaned []string, part string) []string {
	if len(cleaned) == 0 {
		return append(cleaned, part)
	}

	prev := cleaned[len(cleaned)-1]

	switch {
	case strings.EqualFold(prev, part):
		return cleaned
	case isRedundantNamePrefix(prev, part):
		cleaned[len(cleaned)-1] = part
		return cleaned
	case isRedundantNamePrefix(part, prev):
		return cleaned
	default:
		return append(cleaned, part)
	}
}

func isPlaceholderYear(value string) bool {
	return value == "0000" || value == "0"
}

func isRedundantNamePrefix(prefix, full string) bool {
	prefix = strings.TrimSpace(prefix)

	full = strings.TrimSpace(full)
	if prefix == "" || full == "" {
		return false
	}

	if strings.EqualFold(prefix, full) {
		return true
	}

	if !hasPrefixFold(full, prefix) {
		return false
	}

	rest := full[len(prefix):]
	if rest == "" {
		return true
	}

	next, _ := utf8.DecodeRuneInString(rest)

	return unicode.IsSpace(next) || strings.ContainsRune(`.,:;!?—–-«»"'([{`, next)
}

func hasPrefixFold(value, prefix string) bool {
	if strings.HasPrefix(value, prefix) {
		return true
	}

	return strings.HasPrefix(strings.ToLower(value), strings.ToLower(prefix))
}
