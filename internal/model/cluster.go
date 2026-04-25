package model

type ClusterFile struct {
	Version     int       `json:"version"`
	GeneratedAt string    `json:"generated_at"`
	Clusters    []Cluster `json:"clusters"`
}

type Cluster struct {
	ClusterID           string   `json:"cluster_id"`
	Name                string   `json:"name,omitempty"`
	RepresentativeTitle string   `json:"representative_title"`
	SessionIDs          []string `json:"session_ids"`
	TopKeywords         []string `json:"top_keywords,omitempty"`
	Projects            []string `json:"projects,omitempty"`
	SessionCount        int      `json:"session_count"`
	LatestStartedAt     string   `json:"latest_started_at,omitempty"`
}
