package app

import (
	"errors"

	"github.com/liurui/codex-session-manager/internal/model"
	"github.com/liurui/codex-session-manager/internal/parser"
)

func (a *App) GetSessionDetail(sessionID string) (model.SessionIndexEntry, model.SessionDetail, error) {
	entries, err := a.LoadIndexEntries()
	if err != nil {
		return model.SessionIndexEntry{}, model.SessionDetail{}, err
	}

	for _, entry := range entries {
		if entry.SessionID != sessionID {
			continue
		}

		detail, err := parser.ParseSessionDetail(entry.FilePath)
		if err != nil {
			return model.SessionIndexEntry{}, model.SessionDetail{}, err
		}
		if detail.SessionID == "" {
			detail.SessionID = entry.SessionID
		}
		if detail.StartedAt == "" {
			detail.StartedAt = entry.StartedAt
		}
		if detail.CWD == "" {
			detail.CWD = entry.CWD
		}
		if detail.Title == "" {
			detail.Title = entry.Title
		}
		detail.FilePath = entry.FilePath
		return entry, detail, nil
	}

	return model.SessionIndexEntry{}, model.SessionDetail{}, errors.New("未找到对应的 session")
}
