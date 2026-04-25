package app

import (
	"errors"
	"path/filepath"
	"sort"
	"strings"
	"unicode"

	"github.com/liurui/codex-session-manager/internal/model"
	"github.com/liurui/codex-session-manager/internal/store"
)

type FindResult struct {
	Entry model.SessionIndexEntry
	Score int
}

func (a *App) Find(query string, limit int) ([]FindResult, error) {
	query = strings.TrimSpace(query)
	if query == "" {
		return nil, errors.New("query 不能为空")
	}
	if limit <= 0 {
		limit = 10
	}

	exists, err := store.Exists(a.Paths.SessionIndexFile)
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, errors.New("session_index.jsonl 不存在，请先执行 csm scan")
	}

	entries, err := store.ReadJSONLines[model.SessionIndexEntry](a.Paths.SessionIndexFile)
	if err != nil {
		return nil, err
	}

	lowerQuery := strings.ToLower(query)
	queryTokens := splitQuery(lowerQuery)
	results := make([]FindResult, 0, len(entries))

	for _, entry := range entries {
		score := scoreEntry(entry, lowerQuery, queryTokens)
		if score == 0 {
			continue
		}

		results = append(results, FindResult{
			Entry: entry,
			Score: score,
		})
	}

	sort.Slice(results, func(i, j int) bool {
		if results[i].Score == results[j].Score {
			return results[i].Entry.StartedAt > results[j].Entry.StartedAt
		}
		return results[i].Score > results[j].Score
	})

	if len(results) > limit {
		results = results[:limit]
	}
	return results, nil
}

func splitQuery(query string) []string {
	seen := make(map[string]struct{})
	fields := strings.FieldsFunc(query, func(r rune) bool {
		return unicode.IsSpace(r) || unicode.IsPunct(r) || unicode.IsSymbol(r)
	})

	tokens := make([]string, 0, len(fields)+1)
	if query != "" {
		tokens = append(tokens, query)
		seen[query] = struct{}{}
	}

	for _, field := range fields {
		field = strings.TrimSpace(field)
		if len(field) < 2 {
			continue
		}
		if _, exists := seen[field]; exists {
			continue
		}
		seen[field] = struct{}{}
		tokens = append(tokens, field)
	}

	return tokens
}

func scoreEntry(entry model.SessionIndexEntry, query string, tokens []string) int {
	score := 0

	score += scoreText(strings.ToLower(entry.Title), query, tokens, 20, 8)
	score += scoreText(strings.ToLower(entry.CWD), query, tokens, 12, 4)
	score += scoreText(strings.ToLower(entry.Preview), query, tokens, 8, 2)

	for _, keyword := range entry.Keywords {
		score += scoreText(strings.ToLower(keyword), query, tokens, 6, 3)
	}
	for _, project := range entry.Projects {
		score += scoreText(strings.ToLower(project), query, tokens, 10, 5)
	}

	fileBase := strings.ToLower(filepath.Base(entry.FilePath))
	score += scoreText(fileBase, query, tokens, 4, 2)
	return score
}

func scoreText(text, query string, tokens []string, wholeWeight, tokenWeight int) int {
	if text == "" {
		return 0
	}

	score := 0
	if strings.Contains(text, query) {
		score += wholeWeight
	}

	for _, token := range tokens {
		if token == "" || token == query {
			continue
		}
		if strings.Contains(text, token) {
			score += tokenWeight
		}
	}

	return score
}
