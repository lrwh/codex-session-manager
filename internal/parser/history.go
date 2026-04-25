package parser

import (
	"bufio"
	"encoding/json"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"unicode"
	"unicode/utf8"
)

type HistorySummary struct {
	Title            string
	Preview          string
	Keywords         []string
	Projects         []string
	UserMessageCount int
}

type historyRecord struct {
	SessionID string `json:"session_id"`
	TS        int64  `json:"ts"`
	Text      string `json:"text"`
}

type historyGroup struct {
	messages []historyRecord
}

func ParseHistoryFile(path string) (map[string]HistorySummary, error) {
	file, err := os.Open(path)
	if err != nil {
		if os.IsNotExist(err) {
			return map[string]HistorySummary{}, nil
		}
		return nil, err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	scanner.Buffer(make([]byte, 1024), 16*1024*1024)

	groups := make(map[string]*historyGroup)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}

		var record historyRecord
		if err := json.Unmarshal([]byte(line), &record); err != nil {
			continue
		}
		if strings.TrimSpace(record.SessionID) == "" {
			continue
		}

		group := groups[record.SessionID]
		if group == nil {
			group = &historyGroup{}
			groups[record.SessionID] = group
		}
		group.messages = append(group.messages, record)
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}

	summaries := make(map[string]HistorySummary, len(groups))
	for sessionID, group := range groups {
		sort.Slice(group.messages, func(i, j int) bool {
			return group.messages[i].TS < group.messages[j].TS
		})

		texts := make([]string, 0, len(group.messages))
		for _, message := range group.messages {
			text := normalizeWhitespace(message.Text)
			if text != "" {
				texts = append(texts, text)
			}
		}

		title := selectSessionTitle(texts)

		preview := ""
		if len(texts) > 0 {
			preview = shorten(strings.Join(texts, "\n"), 200)
		}

		combinedText := strings.Join(texts, "\n")
		summaries[sessionID] = HistorySummary{
			Title:            title,
			Preview:          preview,
			Keywords:         collectKeywords(combinedText),
			Projects:         collectProjects(combinedText),
			UserMessageCount: len(group.messages),
		}
	}

	return summaries, nil
}

func normalizeWhitespace(value string) string {
	return strings.Join(strings.Fields(strings.TrimSpace(value)), " ")
}

func shorten(value string, limit int) string {
	if limit <= 0 || value == "" {
		return ""
	}

	if utf8.RuneCountInString(value) <= limit {
		return value
	}

	runes := []rune(value)
	return string(runes[:limit]) + "..."
}

func buildSessionTitle(value string) string {
	value = normalizeWhitespace(value)
	if value == "" {
		return ""
	}

	parts := strings.FieldsFunc(value, func(r rune) bool {
		switch r {
		case '\n', '\r', '。', '！', '？', '；', ';', '：', ':', '，', ',', '、':
			return true
		default:
			return false
		}
	})

	for _, part := range parts {
		part = normalizeWhitespace(part)
		if utf8.RuneCountInString(part) >= 4 {
			return shorten(part, 32)
		}
	}

	return shorten(value, 32)
}

func collectKeywords(text string) []string {
	seen := make(map[string]struct{})
	fields := strings.FieldsFunc(strings.ToLower(text), func(r rune) bool {
		return unicode.IsSpace(r) || unicode.IsPunct(r) || unicode.IsSymbol(r)
	})

	keywords := make([]string, 0, 12)
	for _, field := range fields {
		field = strings.TrimSpace(field)
		if len(field) < 2 || len(field) > 48 {
			continue
		}
		if _, exists := seen[field]; exists {
			continue
		}
		seen[field] = struct{}{}
		keywords = append(keywords, field)
		if len(keywords) >= 12 {
			break
		}
	}

	return keywords
}

func collectProjects(text string) []string {
	seen := make(map[string]struct{})
	projects := make([]string, 0, 8)

	for _, field := range strings.Fields(text) {
		if strings.HasPrefix(field, "/") {
			base := filepath.Base(field)
			base = strings.TrimSpace(base)
			if base == "" || base == "." || base == "/" {
				continue
			}
			if _, exists := seen[base]; exists {
				continue
			}
			seen[base] = struct{}{}
			projects = append(projects, base)
			if len(projects) >= 8 {
				break
			}
		}
	}

	return projects
}
