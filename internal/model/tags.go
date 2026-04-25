package model

type TagsFile struct {
	ClusterNames map[string]string `json:"cluster_names"`
	ManualMerges []ClusterMerge    `json:"manual_merges,omitempty"`
	ManualSplits []ClusterSplit    `json:"manual_splits,omitempty"`
}

type ClusterMerge struct {
	Target  string   `json:"target"`
	Sources []string `json:"sources"`
}

type ClusterSplit struct {
	Source     string   `json:"source"`
	Target     string   `json:"target"`
	SessionIDs []string `json:"session_ids"`
}

func DefaultTagsFile() TagsFile {
	return TagsFile{
		ClusterNames: map[string]string{},
		ManualMerges: []ClusterMerge{},
		ManualSplits: []ClusterSplit{},
	}
}
