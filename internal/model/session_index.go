package model

type SessionIndexEntry struct {
	SessionID         string   `json:"session_id"`
	SourceID          string   `json:"source_id"`
	FilePath          string   `json:"file_path"`
	StartedAt         string   `json:"started_at,omitempty"`
	CWD               string   `json:"cwd,omitempty"`
	Title             string   `json:"title,omitempty"`
	Preview           string   `json:"preview,omitempty"`
	Commands          []string `json:"commands,omitempty"`
	Keywords          []string `json:"keywords,omitempty"`
	Projects          []string `json:"projects,omitempty"`
	UserMessageCount  int      `json:"user_message_count"`
	TotalMessageCount int      `json:"total_message_count"`
	ContentHash       string   `json:"content_hash,omitempty"`
}
