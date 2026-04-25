package app

import (
	"errors"
	"sort"

	"github.com/liurui/codex-session-manager/internal/model"
)

type ClusterView struct {
	Cluster  model.Cluster
	Sessions []model.SessionIndexEntry
}

func (a *App) ShowCluster(clusterID string) (ClusterView, error) {
	clusters, err := a.LoadClusters()
	if err != nil {
		return ClusterView{}, err
	}

	var target *model.Cluster
	for i := range clusters {
		if clusters[i].ClusterID == clusterID {
			target = &clusters[i]
			break
		}
	}
	if target == nil {
		return ClusterView{}, errors.New("cluster 不存在")
	}

	entries, err := a.LoadIndexEntries()
	if err != nil {
		return ClusterView{}, err
	}

	sessionSet := make(map[string]struct{}, len(target.SessionIDs))
	for _, sessionID := range target.SessionIDs {
		sessionSet[sessionID] = struct{}{}
	}

	sessions := make([]model.SessionIndexEntry, 0, len(target.SessionIDs))
	for _, entry := range entries {
		if _, ok := sessionSet[entry.SessionID]; ok {
			sessions = append(sessions, entry)
		}
	}

	sort.Slice(sessions, func(i, j int) bool {
		return sessions[i].StartedAt > sessions[j].StartedAt
	})

	return ClusterView{
		Cluster:  *target,
		Sessions: sessions,
	}, nil
}
