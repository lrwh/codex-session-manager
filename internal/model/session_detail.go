package model

type SessionDetail struct {
	SessionID string
	StartedAt string
	CWD       string
	Title     string
	FilePath  string
	Events    []SessionDetailEvent
}

type SessionDetailEvent struct {
	Index     int
	Timestamp string
	Role      string
	Kind      string
	Title     string
	Content   string
}
