package model

type Config struct {
	Version         int           `json:"version"`
	MaxPreviewChars int           `json:"max_preview_chars"`
	ScanConcurrency int           `json:"scan_concurrency"`
	Cluster         ClusterConfig `json:"cluster"`
}

type ClusterConfig struct {
	MinSimilarity float64 `json:"min_similarity"`
	TimeDecayDays int     `json:"time_decay_days"`
}

func DefaultConfig() Config {
	return Config{
		Version:         1,
		MaxPreviewChars: 200,
		ScanConcurrency: 4,
		Cluster: ClusterConfig{
			MinSimilarity: 0.42,
			TimeDecayDays: 14,
		},
	}
}
