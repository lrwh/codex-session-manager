package app

import "github.com/liurui/codex-session-manager/internal/model"

func (a *App) PrepareSessions() error {
	if err := a.Init(); err != nil {
		return err
	}
	if err := a.ensureDefaultSource(); err != nil {
		return err
	}
	_, err := a.Scan()
	return err
}

func (a *App) ListSessions(limit int) ([]model.SessionIndexEntry, error) {
	if err := a.PrepareSessions(); err != nil {
		return nil, err
	}

	entries, err := a.LoadIndexEntries()
	if err != nil {
		return nil, err
	}
	if limit > 0 && len(entries) > limit {
		entries = entries[:limit]
	}
	return entries, nil
}
