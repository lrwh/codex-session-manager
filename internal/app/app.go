package app

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/liurui/codex-session-manager/internal/config"
	"github.com/liurui/codex-session-manager/internal/model"
	"github.com/liurui/codex-session-manager/internal/store"
)

type App struct {
	Paths config.Paths
}

func New() (*App, error) {
	paths, err := config.ResolvePaths()
	if err != nil {
		return nil, err
	}
	return &App{Paths: paths}, nil
}

func (a *App) Init() error {
	if err := os.MkdirAll(a.Paths.HomeDir, 0o755); err != nil {
		return err
	}

	exists, err := store.Exists(a.Paths.ConfigFile)
	if err != nil {
		return err
	}
	if !exists {
		if err := store.WriteJSON(a.Paths.ConfigFile, model.DefaultConfig()); err != nil {
			return err
		}
	}

	exists, err = store.Exists(a.Paths.SourcesFile)
	if err != nil {
		return err
	}
	if !exists {
		if err := store.WriteJSON(a.Paths.SourcesFile, []model.Source{}); err != nil {
			return err
		}
	}

	return nil
}

func (a *App) LoadSources() ([]model.Source, error) {
	exists, err := store.Exists(a.Paths.SourcesFile)
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, errors.New("sources.json 不存在，请先执行 csm init")
	}

	var sources []model.Source
	if err := store.ReadJSON(a.Paths.SourcesFile, &sources); err != nil {
		return nil, err
	}
	return sources, nil
}

func (a *App) SaveSources(sources []model.Source) error {
	return store.WriteJSON(a.Paths.SourcesFile, sources)
}

func (a *App) AddSource(sourcePath string) (model.Source, error) {
	if strings.TrimSpace(sourcePath) == "" {
		return model.Source{}, errors.New("source path 不能为空")
	}

	absolutePath, err := filepath.Abs(sourcePath)
	if err != nil {
		return model.Source{}, err
	}
	absolutePath = filepath.Clean(absolutePath)

	info, err := os.Stat(absolutePath)
	if err != nil {
		return model.Source{}, err
	}
	if !info.IsDir() {
		return model.Source{}, fmt.Errorf("source path 不是目录: %s", absolutePath)
	}

	sources, err := a.LoadSources()
	if err != nil {
		return model.Source{}, err
	}

	for _, source := range sources {
		if source.Path == absolutePath {
			return source, nil
		}
	}

	source := model.Source{
		ID:      nextSourceID(absolutePath, sources),
		Path:    absolutePath,
		Enabled: true,
	}
	sources = append(sources, source)

	if err := a.SaveSources(sources); err != nil {
		return model.Source{}, err
	}

	return source, nil
}

func nextSourceID(sourcePath string, existing []model.Source) string {
	base := filepath.Base(sourcePath)
	base = strings.ToLower(base)

	var builder strings.Builder
	for _, ch := range base {
		switch {
		case ch >= 'a' && ch <= 'z':
			builder.WriteRune(ch)
		case ch >= '0' && ch <= '9':
			builder.WriteRune(ch)
		default:
			builder.WriteRune('-')
		}
	}

	id := strings.Trim(builder.String(), "-")
	if id == "" {
		id = "source"
	}

	candidate := id
	index := 2
	for hasSourceID(candidate, existing) {
		candidate = fmt.Sprintf("%s-%d", id, index)
		index++
	}

	return candidate
}

func hasSourceID(id string, existing []model.Source) bool {
	for _, source := range existing {
		if source.ID == id {
			return true
		}
	}
	return false
}
