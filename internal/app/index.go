package app

import (
	"errors"

	"github.com/liurui/codex-session-manager/internal/model"
	"github.com/liurui/codex-session-manager/internal/store"
)

func (a *App) LoadIndexEntries() ([]model.SessionIndexEntry, error) {
	exists, err := store.Exists(a.Paths.SessionIndexFile)
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, errors.New("session_index.jsonl 不存在，请先执行 csm scan")
	}

	return store.ReadJSONLines[model.SessionIndexEntry](a.Paths.SessionIndexFile)
}
