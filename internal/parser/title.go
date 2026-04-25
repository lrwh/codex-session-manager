package parser

import (
	"strings"
	"unicode"
	"unicode/utf8"
)

var genericAliasMessages = map[string]struct{}{
	"继续":     {},
	"继续继续":   {},
	"继续吧":    {},
	"继续处理":   {},
	"继续修复":   {},
	"继续做":    {},
	"可以":     {},
	"可以了":    {},
	"不对":     {},
	"好的":     {},
	"行":      {},
	"开始":     {},
	"然后":     {},
	"重试":     {},
}

var aliasRejectTerms = []string{
	"怎么",
	"如何",
	"哪里",
	"为什么",
	"报错",
	"错误",
	"无法",
	"失败",
	"检查",
	"输出",
	"继续",
	"启动",
	"运行",
	"没有",
	"还是",
	"现在",
	"这个",
	"那个",
	"帮我",
	"请",
}

func selectSessionTitle(messages []string) string {
	cleaned := make([]string, 0, len(messages))
	for _, message := range messages {
		message = normalizeWhitespace(message)
		if message == "" || isMetadataMessage(message) {
			continue
		}
		cleaned = append(cleaned, message)
	}
	if len(cleaned) == 0 {
		return ""
	}

	if alias := findSessionAlias(cleaned); alias != "" {
		return alias
	}

	return buildSessionTitle(cleaned[0])
}

func findSessionAlias(messages []string) string {
	for _, message := range messages {
		if _, ok := aliasCandidateScore(message); ok {
			return message
		}
	}
	return ""
}

func aliasCandidateScore(message string) (int, bool) {
	message = normalizeWhitespace(message)
	if message == "" || strings.Contains(message, "\n") {
		return 0, false
	}

	length := utf8.RuneCountInString(message)
	if length < 2 || length > 18 {
		return 0, false
	}

	lower := strings.ToLower(message)
	if _, exists := genericAliasMessages[lower]; exists {
		return 0, false
	}

	if strings.Contains(lower, "http://") || strings.Contains(lower, "https://") {
		return 0, false
	}

	for _, term := range aliasRejectTerms {
		if strings.Contains(message, term) {
			return 0, false
		}
	}

	digitCount := 0
	spaceCount := 0
	punctuationCount := 0
	for _, r := range message {
		switch {
		case unicode.IsDigit(r):
			digitCount++
		case unicode.IsSpace(r):
			spaceCount++
		case unicode.IsPunct(r) || unicode.IsSymbol(r):
			switch r {
			case '-', '_':
				// allow lightweight project-like aliases
			default:
				punctuationCount++
			}
		}
	}

	if digitCount > 4 || spaceCount > 3 || punctuationCount > 0 {
		return 0, false
	}

	score := 100 - length
	if length >= 4 && length <= 12 {
		score += 20
	}
	if spaceCount == 0 {
		score += 10
	}
	return score, true
}

func isMetadataMessage(value string) bool {
	return strings.HasPrefix(value, "<environment_context>") ||
		strings.HasPrefix(value, "<turn_aborted>") ||
		strings.HasPrefix(value, "# AGENTS.md instructions")
}
