package config

import (
	"os"
	"path/filepath"
)

const EnvHome = "CSM_HOME"

type Paths struct {
	HomeDir          string
	ConfigFile       string
	SourcesFile      string
	SessionIndexFile string
	ClustersFile     string
	TagsFile         string
}

func ResolvePaths() (Paths, error) {
	homeDir := os.Getenv(EnvHome)
	if homeDir == "" {
		userConfigDir, err := os.UserConfigDir()
		if err != nil {
			return Paths{}, err
		}
		homeDir = filepath.Join(userConfigDir, "csm")
	}

	return Paths{
		HomeDir:          homeDir,
		ConfigFile:       filepath.Join(homeDir, "config.json"),
		SourcesFile:      filepath.Join(homeDir, "sources.json"),
		SessionIndexFile: filepath.Join(homeDir, "session_index.jsonl"),
		ClustersFile:     filepath.Join(homeDir, "clusters.json"),
		TagsFile:         filepath.Join(homeDir, "tags.json"),
	}, nil
}
