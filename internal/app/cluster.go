package app

import (
	"errors"
	"sort"
	"strings"
	"time"

	"github.com/liurui/codex-session-manager/internal/model"
	"github.com/liurui/codex-session-manager/internal/store"
)

type ClusterRebuildResult struct {
	ClusterCount int
	OutputFile   string
}

func (a *App) RebuildClusters() (ClusterRebuildResult, error) {
	exists, err := store.Exists(a.Paths.SessionIndexFile)
	if err != nil {
		return ClusterRebuildResult{}, err
	}
	if !exists {
		return ClusterRebuildResult{}, errors.New("session_index.jsonl 不存在，请先执行 csm scan")
	}

	entries, err := store.ReadJSONLines[model.SessionIndexEntry](a.Paths.SessionIndexFile)
	if err != nil {
		return ClusterRebuildResult{}, err
	}
	if len(entries) == 0 {
		clusterFile := model.ClusterFile{
			Version:     1,
			GeneratedAt: time.Now().Format(time.RFC3339),
			Clusters:    []model.Cluster{},
		}
		if err := store.WriteJSON(a.Paths.ClustersFile, clusterFile); err != nil {
			return ClusterRebuildResult{}, err
		}
		return ClusterRebuildResult{ClusterCount: 0, OutputFile: a.Paths.ClustersFile}, nil
	}

	buckets := make(map[string][]int)
	for index, entry := range entries {
		bucketKey := clusterBucket(entry)
		buckets[bucketKey] = append(buckets[bucketKey], index)
	}

	union := newDisjointSet(len(entries))
	for _, indexes := range buckets {
		for i := 0; i < len(indexes); i++ {
			for j := i + 1; j < len(indexes); j++ {
				left := indexes[i]
				right := indexes[j]
				if clusterSimilarity(entries[left], entries[right]) >= 0.34 {
					union.Union(left, right)
				}
			}
		}
	}

	grouped := make(map[int][]model.SessionIndexEntry)
	for index, entry := range entries {
		root := union.Find(index)
		grouped[root] = append(grouped[root], entry)
	}

	clusters := make([]model.Cluster, 0, len(grouped))
	for _, group := range grouped {
		cluster := buildCluster(group)
		clusters = append(clusters, cluster)
	}

	sort.Slice(clusters, func(i, j int) bool {
		if clusters[i].SessionCount == clusters[j].SessionCount {
			return clusters[i].LatestStartedAt > clusters[j].LatestStartedAt
		}
		return clusters[i].SessionCount > clusters[j].SessionCount
	})

	clusterFile := model.ClusterFile{
		Version:     1,
		GeneratedAt: time.Now().Format(time.RFC3339),
		Clusters:    clusters,
	}
	if err := store.WriteJSON(a.Paths.ClustersFile, clusterFile); err != nil {
		return ClusterRebuildResult{}, err
	}

	return ClusterRebuildResult{
		ClusterCount: len(clusters),
		OutputFile:   a.Paths.ClustersFile,
	}, nil
}

func (a *App) LoadClusters() ([]model.Cluster, error) {
	exists, err := store.Exists(a.Paths.ClustersFile)
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, errors.New("clusters.json 不存在，请先执行 csm cluster rebuild")
	}

	var clusterFile model.ClusterFile
	if err := store.ReadJSON(a.Paths.ClustersFile, &clusterFile); err != nil {
		return nil, err
	}

	tags, err := a.LoadTags()
	if err != nil {
		return nil, err
	}

	indexEntries, err := a.LoadIndexEntries()
	if err != nil {
		return nil, err
	}

	clusters := applyManualMerges(clusterFile.Clusters, tags.ManualMerges)
	clusters = applyManualSplits(clusters, tags.ManualSplits, indexEntries)
	for index := range clusters {
		if name, ok := tags.ClusterNames[clusters[index].ClusterID]; ok {
			clusters[index].Name = name
		}
	}
	return clusters, nil
}

func clusterBucket(entry model.SessionIndexEntry) string {
	if len(entry.Projects) > 0 && entry.Projects[0] != "" {
		return "project:" + strings.ToLower(entry.Projects[0])
	}

	cwdProject := strings.ToLower(projectFromCWD(entry.CWD))
	if cwdProject != "" {
		return "cwd:" + cwdProject
	}

	tokens := splitQuery(strings.ToLower(entry.Title))
	if len(tokens) > 0 {
		return "title:" + tokens[0]
	}

	return "misc"
}

func clusterSimilarity(left, right model.SessionIndexEntry) float64 {
	leftTokens := clusterTokens(left)
	rightTokens := clusterTokens(right)
	if len(leftTokens) == 0 || len(rightTokens) == 0 {
		return 0
	}

	leftSet := make(map[string]struct{}, len(leftTokens))
	for _, token := range leftTokens {
		leftSet[token] = struct{}{}
	}

	rightSet := make(map[string]struct{}, len(rightTokens))
	for _, token := range rightTokens {
		rightSet[token] = struct{}{}
	}

	intersection := 0
	for token := range leftSet {
		if _, ok := rightSet[token]; ok {
			intersection++
		}
	}

	unionSize := len(leftSet) + len(rightSet) - intersection
	if unionSize == 0 {
		return 0
	}

	similarity := float64(intersection) / float64(unionSize)
	if shareProject(left, right) {
		similarity += 0.18
	}
	if sameCWDProject(left, right) {
		similarity += 0.12
	}
	if similarity > 1 {
		return 1
	}
	return similarity
}

func clusterTokens(entry model.SessionIndexEntry) []string {
	seen := make(map[string]struct{})
	tokens := make([]string, 0, 24)

	add := func(values ...string) {
		for _, value := range values {
			value = strings.TrimSpace(strings.ToLower(value))
			if len(value) < 2 {
				continue
			}
			if _, exists := seen[value]; exists {
				continue
			}
			seen[value] = struct{}{}
			tokens = append(tokens, value)
		}
	}

	add(entry.Projects...)
	add(projectFromCWD(entry.CWD))
	add(entry.Keywords...)
	add(splitQuery(strings.ToLower(entry.Title))...)
	return tokens
}

func shareProject(left, right model.SessionIndexEntry) bool {
	leftSet := make(map[string]struct{}, len(left.Projects))
	for _, project := range left.Projects {
		leftSet[strings.ToLower(project)] = struct{}{}
	}
	for _, project := range right.Projects {
		if _, ok := leftSet[strings.ToLower(project)]; ok {
			return true
		}
	}
	return false
}

func sameCWDProject(left, right model.SessionIndexEntry) bool {
	return strings.EqualFold(projectFromCWD(left.CWD), projectFromCWD(right.CWD))
}

func buildCluster(entries []model.SessionIndexEntry) model.Cluster {
	sort.Slice(entries, func(i, j int) bool {
		return entries[i].StartedAt > entries[j].StartedAt
	})

	sessionIDs := make([]string, 0, len(entries))
	keywordCount := make(map[string]int)
	projectCount := make(map[string]int)
	minSessionID := entries[0].SessionID

	for _, entry := range entries {
		sessionIDs = append(sessionIDs, entry.SessionID)
		if entry.SessionID < minSessionID {
			minSessionID = entry.SessionID
		}
		for _, keyword := range entry.Keywords {
			keywordCount[keyword]++
		}
		for _, project := range entry.Projects {
			projectCount[project]++
		}
	}
	sort.Strings(sessionIDs)

	return model.Cluster{
		ClusterID:           "cluster-" + minSessionID,
		RepresentativeTitle: entries[0].Title,
		SessionIDs:          sessionIDs,
		TopKeywords:         topKeys(keywordCount, 6),
		Projects:            topKeys(projectCount, 4),
		SessionCount:        len(entries),
		LatestStartedAt:     entries[0].StartedAt,
	}
}

func applyManualMerges(clusters []model.Cluster, rules []model.ClusterMerge) []model.Cluster {
	if len(rules) == 0 {
		return clusters
	}

	clusterMap := make(map[string]model.Cluster, len(clusters))
	for _, cluster := range clusters {
		clusterMap[cluster.ClusterID] = cluster
	}

	for _, rule := range rules {
		target, ok := clusterMap[rule.Target]
		if !ok {
			continue
		}

		mergedSessions := append([]string{}, target.SessionIDs...)
		mergedKeywords := append([]string{}, target.TopKeywords...)
		mergedProjects := append([]string{}, target.Projects...)
		latest := target.LatestStartedAt
		sessionCount := target.SessionCount

		for _, sourceID := range rule.Sources {
			source, ok := clusterMap[sourceID]
			if !ok {
				continue
			}
			mergedSessions = append(mergedSessions, source.SessionIDs...)
			mergedKeywords = append(mergedKeywords, source.TopKeywords...)
			mergedProjects = append(mergedProjects, source.Projects...)
			if source.LatestStartedAt > latest {
				latest = source.LatestStartedAt
			}
			sessionCount += source.SessionCount
			delete(clusterMap, sourceID)
		}

		target.SessionIDs = uniqueStrings(mergedSessions)
		sort.Strings(target.SessionIDs)
		target.TopKeywords = uniqueStrings(mergedKeywords)
		if len(target.TopKeywords) > 6 {
			target.TopKeywords = target.TopKeywords[:6]
		}
		target.Projects = uniqueStrings(mergedProjects)
		if len(target.Projects) > 4 {
			target.Projects = target.Projects[:4]
		}
		target.LatestStartedAt = latest
		target.SessionCount = len(target.SessionIDs)
		if target.SessionCount == 0 {
			target.SessionCount = sessionCount
		}
		clusterMap[rule.Target] = target
	}

	merged := make([]model.Cluster, 0, len(clusterMap))
	for _, cluster := range clusterMap {
		merged = append(merged, cluster)
	}
	sort.Slice(merged, func(i, j int) bool {
		if merged[i].SessionCount == merged[j].SessionCount {
			return merged[i].LatestStartedAt > merged[j].LatestStartedAt
		}
		return merged[i].SessionCount > merged[j].SessionCount
	})
	return merged
}

func applyManualSplits(clusters []model.Cluster, rules []model.ClusterSplit, entries []model.SessionIndexEntry) []model.Cluster {
	if len(rules) == 0 {
		return clusters
	}

	entryMap := make(map[string]model.SessionIndexEntry, len(entries))
	for _, entry := range entries {
		entryMap[entry.SessionID] = entry
	}

	clusterMap := make(map[string]model.Cluster, len(clusters))
	for _, cluster := range clusters {
		clusterMap[cluster.ClusterID] = cluster
	}

	for _, rule := range rules {
		source, ok := clusterMap[rule.Source]
		if !ok {
			continue
		}

		sessionSet := make(map[string]struct{}, len(rule.SessionIDs))
		for _, sessionID := range rule.SessionIDs {
			sessionSet[sessionID] = struct{}{}
		}

		remaining := make([]string, 0, len(source.SessionIDs))
		splitted := make([]model.SessionIndexEntry, 0, len(rule.SessionIDs))
		for _, sessionID := range source.SessionIDs {
			if _, shouldSplit := sessionSet[sessionID]; shouldSplit {
				if entry, ok := entryMap[sessionID]; ok {
					splitted = append(splitted, entry)
				}
				continue
			}
			remaining = append(remaining, sessionID)
		}

		if len(splitted) == 0 {
			continue
		}

		source.SessionIDs = remaining
		source.SessionCount = len(remaining)
		if len(remaining) == 0 {
			delete(clusterMap, rule.Source)
		} else {
			clusterMap[rule.Source] = source
		}

		splitCluster := buildCluster(splitted)
		splitCluster.ClusterID = rule.Target
		clusterMap[rule.Target] = splitCluster
	}

	result := make([]model.Cluster, 0, len(clusterMap))
	for _, cluster := range clusterMap {
		result = append(result, cluster)
	}
	sort.Slice(result, func(i, j int) bool {
		if result[i].SessionCount == result[j].SessionCount {
			return result[i].LatestStartedAt > result[j].LatestStartedAt
		}
		return result[i].SessionCount > result[j].SessionCount
	})
	return result
}

func topKeys(counter map[string]int, limit int) []string {
	type item struct {
		Key   string
		Count int
	}

	items := make([]item, 0, len(counter))
	for key, count := range counter {
		if strings.TrimSpace(key) == "" {
			continue
		}
		items = append(items, item{Key: key, Count: count})
	}

	sort.Slice(items, func(i, j int) bool {
		if items[i].Count == items[j].Count {
			return items[i].Key < items[j].Key
		}
		return items[i].Count > items[j].Count
	})

	if len(items) > limit {
		items = items[:limit]
	}

	keys := make([]string, 0, len(items))
	for _, item := range items {
		keys = append(keys, item.Key)
	}
	return keys
}

type disjointSet struct {
	parent []int
	rank   []int
}

func newDisjointSet(size int) *disjointSet {
	parent := make([]int, size)
	rank := make([]int, size)
	for i := 0; i < size; i++ {
		parent[i] = i
	}
	return &disjointSet{parent: parent, rank: rank}
}

func (d *disjointSet) Find(x int) int {
	if d.parent[x] != x {
		d.parent[x] = d.Find(d.parent[x])
	}
	return d.parent[x]
}

func (d *disjointSet) Union(x, y int) {
	rootX := d.Find(x)
	rootY := d.Find(y)
	if rootX == rootY {
		return
	}

	if d.rank[rootX] < d.rank[rootY] {
		d.parent[rootX] = rootY
		return
	}
	if d.rank[rootX] > d.rank[rootY] {
		d.parent[rootY] = rootX
		return
	}

	d.parent[rootY] = rootX
	d.rank[rootX]++
}
