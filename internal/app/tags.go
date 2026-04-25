package app

import (
	"errors"
	"strings"

	"github.com/liurui/codex-session-manager/internal/model"
	"github.com/liurui/codex-session-manager/internal/store"
)

func (a *App) LoadTags() (model.TagsFile, error) {
	exists, err := store.Exists(a.Paths.TagsFile)
	if err != nil {
		return model.TagsFile{}, err
	}
	if !exists {
		return model.DefaultTagsFile(), nil
	}

	var tags model.TagsFile
	if err := store.ReadJSON(a.Paths.TagsFile, &tags); err != nil {
		return model.TagsFile{}, err
	}
	if tags.ClusterNames == nil {
		tags.ClusterNames = map[string]string{}
	}
	return tags, nil
}

func (a *App) SaveTags(tags model.TagsFile) error {
	if tags.ClusterNames == nil {
		tags.ClusterNames = map[string]string{}
	}
	if tags.ManualMerges == nil {
		tags.ManualMerges = []model.ClusterMerge{}
	}
	if tags.ManualSplits == nil {
		tags.ManualSplits = []model.ClusterSplit{}
	}
	return store.WriteJSON(a.Paths.TagsFile, tags)
}

func (a *App) SetClusterName(clusterID, name string) error {
	clusterID = strings.TrimSpace(clusterID)
	name = strings.TrimSpace(name)
	if clusterID == "" {
		return errors.New("cluster id 不能为空")
	}
	if name == "" {
		return errors.New("cluster name 不能为空")
	}

	clusters, err := a.LoadClusters()
	if err != nil {
		return err
	}

	found := false
	for _, cluster := range clusters {
		if cluster.ClusterID == clusterID {
			found = true
			break
		}
	}
	if !found {
		return errors.New("cluster 不存在")
	}

	tags, err := a.LoadTags()
	if err != nil {
		return err
	}
	tags.ClusterNames[clusterID] = name
	return a.SaveTags(tags)
}

func (a *App) RemoveClusterName(clusterID string) error {
	clusterID = strings.TrimSpace(clusterID)
	if clusterID == "" {
		return errors.New("cluster id 不能为空")
	}

	tags, err := a.LoadTags()
	if err != nil {
		return err
	}

	delete(tags.ClusterNames, clusterID)
	return a.SaveTags(tags)
}

func (a *App) AddClusterMerge(target string, sources []string) error {
	target = strings.TrimSpace(target)
	if target == "" {
		return errors.New("target cluster 不能为空")
	}

	cleanSources := make([]string, 0, len(sources))
	seenSources := make(map[string]struct{})
	for _, source := range sources {
		source = strings.TrimSpace(source)
		if source == "" || source == target {
			continue
		}
		if _, exists := seenSources[source]; exists {
			continue
		}
		seenSources[source] = struct{}{}
		cleanSources = append(cleanSources, source)
	}
	if len(cleanSources) == 0 {
		return errors.New("至少需要一个 source cluster")
	}

	clusters, err := a.LoadClusters()
	if err != nil {
		return err
	}
	clusterSet := make(map[string]struct{}, len(clusters))
	for _, cluster := range clusters {
		clusterSet[cluster.ClusterID] = struct{}{}
	}
	if _, ok := clusterSet[target]; !ok {
		return errors.New("target cluster 不存在")
	}
	for _, source := range cleanSources {
		if _, ok := clusterSet[source]; !ok {
			return errors.New("source cluster 不存在: " + source)
		}
	}

	tags, err := a.LoadTags()
	if err != nil {
		return err
	}

	sourceSet := make(map[string]struct{}, len(cleanSources))
	for _, source := range cleanSources {
		sourceSet[source] = struct{}{}
	}

	updated := make([]model.ClusterMerge, 0, len(tags.ManualMerges)+1)
	var targetRule *model.ClusterMerge
	for _, rule := range tags.ManualMerges {
		filtered := make([]string, 0, len(rule.Sources))
		for _, source := range rule.Sources {
			if _, shouldRemove := sourceSet[source]; shouldRemove {
				continue
			}
			if source == target {
				continue
			}
			filtered = append(filtered, source)
		}

		if rule.Target == target {
			targetRule = &model.ClusterMerge{
				Target:  target,
				Sources: append(filtered, cleanSources...),
			}
			continue
		}

		if len(filtered) > 0 {
			rule.Sources = uniqueStrings(filtered)
			updated = append(updated, rule)
		}
	}

	if targetRule == nil {
		targetRule = &model.ClusterMerge{
			Target:  target,
			Sources: cleanSources,
		}
	}
	targetRule.Sources = uniqueStrings(targetRule.Sources)
	updated = append(updated, *targetRule)

	tags.ManualMerges = updated
	return a.SaveTags(tags)
}

func (a *App) AddClusterSplit(source string, sessionIDs []string) (string, error) {
	source = strings.TrimSpace(source)
	if source == "" {
		return "", errors.New("source cluster 不能为空")
	}

	cleanSessionIDs := uniqueStrings(sessionIDs)
	if len(cleanSessionIDs) == 0 {
		return "", errors.New("至少需要一个 session id")
	}

	view, err := a.ShowCluster(source)
	if err != nil {
		return "", err
	}

	sessionSet := make(map[string]struct{}, len(view.Cluster.SessionIDs))
	for _, sessionID := range view.Cluster.SessionIDs {
		sessionSet[sessionID] = struct{}{}
	}
	for _, sessionID := range cleanSessionIDs {
		if _, ok := sessionSet[sessionID]; !ok {
			return "", errors.New("session 不属于 source cluster: " + sessionID)
		}
	}

	tags, err := a.LoadTags()
	if err != nil {
		return "", err
	}

	for _, rule := range tags.ManualSplits {
		for _, existing := range rule.SessionIDs {
			for _, sessionID := range cleanSessionIDs {
				if existing == sessionID {
					return "", errors.New("session 已经存在于 split 规则中: " + sessionID)
				}
			}
		}
	}

	target := "split-" + cleanSessionIDs[0]
	tags.ManualSplits = append(tags.ManualSplits, model.ClusterSplit{
		Source:     source,
		Target:     target,
		SessionIDs: cleanSessionIDs,
	})

	if err := a.SaveTags(tags); err != nil {
		return "", err
	}
	return target, nil
}

func (a *App) ResetCluster(clusterID string) error {
	clusterID = strings.TrimSpace(clusterID)
	if clusterID == "" {
		return errors.New("cluster id 不能为空")
	}

	tags, err := a.LoadTags()
	if err != nil {
		return err
	}

	delete(tags.ClusterNames, clusterID)

	updatedMerges := make([]model.ClusterMerge, 0, len(tags.ManualMerges))
	for _, rule := range tags.ManualMerges {
		if rule.Target == clusterID {
			continue
		}

		filteredSources := make([]string, 0, len(rule.Sources))
		for _, source := range rule.Sources {
			if source == clusterID {
				continue
			}
			filteredSources = append(filteredSources, source)
		}

		if len(filteredSources) == 0 {
			continue
		}

		rule.Sources = uniqueStrings(filteredSources)
		updatedMerges = append(updatedMerges, rule)
	}
	tags.ManualMerges = updatedMerges

	updatedSplits := make([]model.ClusterSplit, 0, len(tags.ManualSplits))
	for _, rule := range tags.ManualSplits {
		if rule.Source == clusterID || rule.Target == clusterID {
			continue
		}
		updatedSplits = append(updatedSplits, rule)
	}
	tags.ManualSplits = updatedSplits

	return a.SaveTags(tags)
}

func uniqueStrings(values []string) []string {
	seen := make(map[string]struct{}, len(values))
	result := make([]string, 0, len(values))
	for _, value := range values {
		if value == "" {
			continue
		}
		if _, exists := seen[value]; exists {
			continue
		}
		seen[value] = struct{}{}
		result = append(result, value)
	}
	return result
}
