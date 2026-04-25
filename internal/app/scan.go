package app

import (
	"errors"
	"fmt"
	"io/fs"
	"path/filepath"
	"sort"

	"github.com/liurui/codex-session-manager/internal/model"
	"github.com/liurui/codex-session-manager/internal/parser"
	"github.com/liurui/codex-session-manager/internal/store"
)

type ScanResult struct {
	SourceCount  int
	SessionCount int
	OutputFile   string
}

func (a *App) Scan() (ScanResult, error) {
	if err := a.Init(); err != nil {
		return ScanResult{}, err
	}

	sources, err := a.LoadSources()
	if err != nil {
		return ScanResult{}, err
	}

	enabledSources := make([]model.Source, 0, len(sources))
	for _, source := range sources {
		if source.Enabled {
			enabledSources = append(enabledSources, source)
		}
	}
	if len(enabledSources) == 0 {
		return ScanResult{}, errors.New("没有可用的数据源，请先执行 csm source add <path>")
	}

	entries := make([]model.SessionIndexEntry, 0, 128)
	for _, source := range enabledSources {
		historyPath := filepath.Join(source.Path, "history.jsonl")
		historyBySession, err := parser.ParseHistoryFile(historyPath)
		if err != nil {
			return ScanResult{}, fmt.Errorf("解析 history 失败 (%s): %w", source.Path, err)
		}

		threadIndexPath := filepath.Join(source.Path, "session_index.jsonl")
		threadIndexBySession, err := parser.ParseThreadIndexFile(threadIndexPath)
		if err != nil {
			return ScanResult{}, fmt.Errorf("解析 session_index 失败 (%s): %w", source.Path, err)
		}

		sessionsRoot := filepath.Join(source.Path, "sessions")
		if err := filepath.WalkDir(sessionsRoot, func(path string, d fs.DirEntry, walkErr error) error {
			if walkErr != nil {
				return walkErr
			}
			if d.IsDir() {
				return nil
			}
			if filepath.Ext(path) != ".jsonl" {
				return nil
			}

			sessionSummary, err := parser.ParseSessionFile(path)
			if err != nil {
				return err
			}
			if sessionSummary.SessionID == "" {
				return nil
			}

			historySummary := historyBySession[sessionSummary.SessionID]
			projects := mergeUnique(append(mergeUnique(historySummary.Projects...), sessionSummary.Projects...)...)
			projects = mergeUnique(append(projects, projectFromCWD(sessionSummary.CWD))...)

			keywords := mergeUnique(append(mergeUnique(historySummary.Keywords...), sessionSummary.Keywords...)...)
			keywords = mergeUnique(append(keywords, projectFromCWD(sessionSummary.CWD))...)

			title := threadIndexBySession[sessionSummary.SessionID].ThreadName
			if title == "" {
				title = sessionSummary.ThreadName
			}
			if title == "" {
				title = historySummary.Title
			}
			if title == "" {
				title = sessionSummary.Title
			}
			if title == "" {
				title = filepath.Base(path)
			}

			preview := historySummary.Preview
			if preview == "" {
				preview = sessionSummary.Preview
			}

			userMessageCount := historySummary.UserMessageCount
			if userMessageCount == 0 {
				userMessageCount = sessionSummary.UserMessageCount
			}

			entries = append(entries, model.SessionIndexEntry{
				SessionID:         sessionSummary.SessionID,
				SourceID:          source.ID,
				FilePath:          path,
				StartedAt:         sessionSummary.StartedAt,
				CWD:               sessionSummary.CWD,
				Title:             title,
				Preview:           preview,
				Commands:          sessionSummary.Commands,
				Keywords:          keywords,
				Projects:          projects,
				UserMessageCount:  userMessageCount,
				TotalMessageCount: sessionSummary.TotalMessageCount,
				ContentHash:       sessionSummary.ContentHash,
			})
			return nil
		}); err != nil {
			return ScanResult{}, fmt.Errorf("扫描 session 失败 (%s): %w", source.Path, err)
		}
	}

	sort.Slice(entries, func(i, j int) bool {
		if entries[i].StartedAt == entries[j].StartedAt {
			if entries[i].SourceID == entries[j].SourceID {
				return entries[i].FilePath < entries[j].FilePath
			}
			return entries[i].SourceID < entries[j].SourceID
		}
		return entries[i].StartedAt > entries[j].StartedAt
	})

	if err := store.WriteJSONLines(a.Paths.SessionIndexFile, entries); err != nil {
		return ScanResult{}, err
	}

	return ScanResult{
		SourceCount:  len(enabledSources),
		SessionCount: len(entries),
		OutputFile:   a.Paths.SessionIndexFile,
	}, nil
}

func projectFromCWD(cwd string) string {
	cwd = filepath.Clean(cwd)
	base := filepath.Base(cwd)
	parent := filepath.Base(filepath.Dir(cwd))

	switch base {
	case "", ".", "/":
		return ""
	case parent:
		return ""
	case "home", "Users":
		return ""
	default:
		if parent == "home" || parent == "Users" {
			return ""
		}
		return base
	}
}

func mergeUnique(values ...string) []string {
	seen := make(map[string]struct{})
	merged := make([]string, 0, len(values))

	for _, value := range values {
		if value == "" {
			continue
		}
		if _, exists := seen[value]; exists {
			continue
		}
		seen[value] = struct{}{}
		merged = append(merged, value)
	}

	return merged
}
