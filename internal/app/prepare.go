package app

import (
	"errors"
	"os"
	"path/filepath"
)

func (a *App) PrepareData() error {
	if err := a.Init(); err != nil {
		return err
	}
	if err := a.ensureDefaultSource(); err != nil {
		return err
	}
	if _, err := a.Scan(); err != nil {
		return err
	}
	if _, err := a.RebuildClusters(); err != nil {
		return err
	}
	return nil
}

func (a *App) ensureDefaultSource() error {
	sources, err := a.LoadSources()
	if err != nil {
		return err
	}
	if len(sources) > 0 {
		return nil
	}

	homeDir, err := os.UserHomeDir()
	if err != nil {
		return err
	}
	defaultSource := filepath.Join(homeDir, ".codex")

	info, err := os.Stat(defaultSource)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return errors.New("没有可用的数据源，请先执行 csm source add <path>")
		}
		return err
	}
	if !info.IsDir() {
		return errors.New("默认数据源不是目录: " + defaultSource)
	}

	_, err = a.AddSource(defaultSource)
	return err
}
