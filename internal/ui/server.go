package ui

import (
	"html/template"
	"net"
	"net/http"
	"net/url"
	"os/exec"
	"runtime"
	"strings"

	"github.com/liurui/codex-session-manager/internal/app"
	"github.com/liurui/codex-session-manager/internal/model"
)

type Server struct {
	App *app.App
}

func New(application *app.App) *Server {
	return &Server{App: application}
}

func (s *Server) ListenAndServe(addr string, openBrowser bool) (string, error) {
	mux := http.NewServeMux()
	mux.HandleFunc("/", s.handleHome)
	mux.HandleFunc("/session", s.handleSessionDetail)
	mux.HandleFunc("/refresh", s.handleRefresh)
	mux.HandleFunc("/cluster/tag", s.handleTagSet)
	mux.HandleFunc("/cluster/reset", s.handleClusterReset)

	listener, err := net.Listen("tcp", addr)
	if err != nil {
		return "", err
	}

	url := "http://" + listener.Addr().String()
	if openBrowser {
		_ = open(url)
	}

	server := &http.Server{Handler: mux}
	go server.Serve(listener)
	return url, nil
}

func (s *Server) handleHome(w http.ResponseWriter, r *http.Request) {
	query := strings.TrimSpace(r.URL.Query().Get("q"))
	message := strings.TrimSpace(r.URL.Query().Get("message"))
	errorMessage := strings.TrimSpace(r.URL.Query().Get("error"))

	indexEntries, err := s.App.LoadIndexEntries()
	if err != nil && errorMessage == "" {
		errorMessage = err.Error()
	}

	sessions := filterSessions(indexEntries, query)

	data := pageData{
		Query:                query,
		Sessions:             sessions,
		TotalSessions:        len(indexEntries),
		VisibleCount:         len(sessions),
		VisibleUserMessages:  sumUserMessages(sessions),
		VisibleTotalMessages: sumTotalMessages(sessions),
		Message:              message,
		Error:                errorMessage,
	}

	tpl := template.Must(template.New("page").Funcs(template.FuncMap{
		"joinCommands":  joinCommands,
		"resumeCommand": resumeCommand,
		"sessionLink":   sessionLink,
	}).Parse(pageTemplate))
	_ = tpl.Execute(w, data)
}

func (s *Server) handleSessionDetail(w http.ResponseWriter, r *http.Request) {
	sessionID := strings.TrimSpace(r.URL.Query().Get("id"))
	query := strings.TrimSpace(r.URL.Query().Get("q"))
	if sessionID == "" {
		http.Redirect(w, r, "/?error="+url.QueryEscape("缺少 session id"), http.StatusSeeOther)
		return
	}

	entry, detail, err := s.App.GetSessionDetail(sessionID)
	if err != nil {
		http.Redirect(w, r, "/?error="+url.QueryEscape(err.Error()), http.StatusSeeOther)
		return
	}
	filtered := filterDetailEvents(detail.Events, query)

	data := detailPageData{
		Query:        query,
		Entry:        entry,
		Detail:       detail,
		Events:       filtered,
		TotalEvents:  len(detail.Events),
		VisibleCount: len(filtered),
	}

	tpl := template.Must(template.New("detail").Funcs(template.FuncMap{
		"resumeCommand": resumeCommand,
		"roleLabel":     roleLabel,
		"barPercent":    barPercent,
		"percentValue":  percentValue,
	}).Parse(detailTemplate))
	_ = tpl.Execute(w, data)
}

func (s *Server) handleRefresh(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Redirect(w, r, "/", http.StatusSeeOther)
		return
	}

	if err := s.App.PrepareData(); err != nil {
		http.Redirect(w, r, "/?error="+url.QueryEscape(err.Error()), http.StatusSeeOther)
		return
	}
	http.Redirect(w, r, "/?message="+url.QueryEscape("数据已刷新"), http.StatusSeeOther)
}

func (s *Server) handleTagSet(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Redirect(w, r, "/", http.StatusSeeOther)
		return
	}
	clusterID := r.FormValue("cluster_id")
	name := r.FormValue("name")
	if err := s.App.SetClusterName(clusterID, name); err != nil {
		http.Redirect(w, r, "/?cluster="+url.QueryEscape(clusterID)+"&error="+url.QueryEscape(err.Error()), http.StatusSeeOther)
		return
	}
	http.Redirect(w, r, "/?cluster="+url.QueryEscape(clusterID)+"&message="+url.QueryEscape("名称已更新"), http.StatusSeeOther)
}

func (s *Server) handleClusterReset(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Redirect(w, r, "/", http.StatusSeeOther)
		return
	}
	clusterID := r.FormValue("cluster_id")
	if err := s.App.ResetCluster(clusterID); err != nil {
		http.Redirect(w, r, "/?cluster="+url.QueryEscape(clusterID)+"&error="+url.QueryEscape(err.Error()), http.StatusSeeOther)
		return
	}
	http.Redirect(w, r, "/?message="+url.QueryEscape("cluster 已重置"), http.StatusSeeOther)
}

type pageData struct {
	Query                string
	Sessions             []model.SessionIndexEntry
	TotalSessions        int
	VisibleCount         int
	VisibleUserMessages  int
	VisibleTotalMessages int
	Message              string
	Error                string
}

type detailPageData struct {
	Query        string
	Entry        model.SessionIndexEntry
	Detail       model.SessionDetail
	Events       []model.SessionDetailEvent
	TotalEvents  int
	VisibleCount int
}

func filterSessions(entries []model.SessionIndexEntry, query string) []model.SessionIndexEntry {
	query = strings.TrimSpace(strings.ToLower(query))
	if query == "" {
		return entries
	}

	filtered := make([]model.SessionIndexEntry, 0, len(entries))
	for _, entry := range entries {
		if sessionMatches(entry, query) {
			filtered = append(filtered, entry)
		}
	}
	return filtered
}

func filterDetailEvents(events []model.SessionDetailEvent, query string) []model.SessionDetailEvent {
	query = strings.TrimSpace(strings.ToLower(query))
	if query == "" {
		return events
	}

	filtered := make([]model.SessionDetailEvent, 0, len(events))
	for _, event := range events {
		if strings.Contains(strings.ToLower(eventSearchText(event)), query) {
			filtered = append(filtered, event)
		}
	}
	return filtered
}

func sessionMatches(entry model.SessionIndexEntry, query string) bool {
	values := []string{
		entry.SessionID,
		entry.StartedAt,
		entry.Title,
		entry.Preview,
		entry.CWD,
		entry.FilePath,
		strings.Join(entry.Commands, "\n"),
		strings.Join(entry.Keywords, " "),
		strings.Join(entry.Projects, " "),
	}

	for _, value := range values {
		if strings.Contains(strings.ToLower(value), query) {
			return true
		}
	}
	return false
}

func joinCommands(commands []string) string {
	trimmed := make([]string, 0, len(commands))
	for _, command := range commands {
		command = strings.TrimSpace(command)
		if command == "" {
			continue
		}
		trimmed = append(trimmed, command)
		if len(trimmed) >= 2 {
			break
		}
	}
	if len(trimmed) == 0 {
		return "暂未解析"
	}
	return strings.Join(trimmed, "\n")
}

func sumUserMessages(entries []model.SessionIndexEntry) int {
	total := 0
	for _, entry := range entries {
		total += entry.UserMessageCount
	}
	return total
}

func sumTotalMessages(entries []model.SessionIndexEntry) int {
	total := 0
	for _, entry := range entries {
		total += entry.TotalMessageCount
	}
	return total
}

func resumeCommand(entry model.SessionIndexEntry) string {
	return "codex resume " + entry.SessionID
}

func sessionLink(entry model.SessionIndexEntry) string {
	return "/session?id=" + url.QueryEscape(entry.SessionID)
}

func roleLabel(role string) string {
	switch strings.ToLower(strings.TrimSpace(role)) {
	case "user":
		return "用户"
	case "assistant":
		return "助手"
	case "developer":
		return "开发者"
	case "tool":
		return "工具"
	default:
		return "系统"
	}
}

func eventSearchText(event model.SessionDetailEvent) string {
	return strings.Join([]string{
		event.Timestamp,
		event.Role,
		event.Kind,
		event.Title,
		event.Content,
	}, "\n")
}

func percentValue(value, total int) int {
	if total <= 0 || value <= 0 {
		return 0
	}
	result := value * 100 / total
	if result > 100 {
		return 100
	}
	return result
}

func barPercent(value, total int) int {
	result := percentValue(value, total)
	if result == 0 {
		return 0
	}
	if result < 6 {
		return 6
	}
	return result
}

func open(target string) error {
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "darwin":
		cmd = exec.Command("open", target)
	case "windows":
		cmd = exec.Command("rundll32", "url.dll,FileProtocolHandler", target)
	default:
		cmd = exec.Command("xdg-open", target)
	}
	return cmd.Start()
}

const pageTemplate = `<!doctype html>
<html lang="zh-CN">
<head>
  <meta charset="utf-8">
  <meta name="viewport" content="width=device-width,initial-scale=1">
  <title>CSM</title>
  <style>
    :root {
      --bg: #f7f9ff;
      --bg-2: #eef4ff;
      --hero: #f4f8ff;
      --hero-2: #e9f1ff;
      --panel: rgba(255, 255, 255, 0.94);
      --panel-strong: #ffffff;
      --sidebar: rgba(232, 240, 255, 0.96);
      --sidebar-card: rgba(255, 255, 255, 0.05);
      --sidebar-line: rgba(53, 92, 255, 0.12);
      --ink: #1d2b57;
      --muted: #6d7898;
      --line: #dbe5fb;
      --line-strong: #c5d3f4;
      --accent: #3f5cff;
      --accent-strong: #2f49ea;
      --accent-soft: #edf2ff;
      --warm: #ff7a1a;
      --warm-soft: #fff1e5;
      --shadow: 0 20px 60px rgba(62, 92, 187, 0.10);
    }
    * { box-sizing: border-box; }
    body {
      margin: 0;
      font-family: "IBM Plex Sans", "Noto Sans SC", sans-serif;
      color: var(--ink);
      background:
        radial-gradient(circle at top left, rgba(255,255,255,0.96) 0, transparent 24%),
        radial-gradient(circle at top right, rgba(63,92,255,0.12) 0, transparent 18%),
        linear-gradient(180deg, var(--bg) 0%, var(--bg-2) 100%);
    }
    .shell {
      max-width: 1480px;
      margin: 0 auto;
      padding: 24px 24px 40px;
    }
    .hero, .panel, .stat {
      background: var(--panel);
      border: 1px solid var(--line);
      border-radius: 24px;
      box-shadow: var(--shadow);
      backdrop-filter: blur(8px);
    }
    .hero {
      padding: 24px 24px 22px;
      margin-bottom: 20px;
      color: var(--ink);
      background:
        radial-gradient(circle at top right, rgba(63,92,255,0.16) 0, transparent 26%),
        radial-gradient(circle at bottom left, rgba(99,160,255,0.12) 0, transparent 24%),
        linear-gradient(135deg, var(--hero) 0%, var(--hero-2) 100%);
      border-color: rgba(63,92,255,0.10);
    }
    h1 {
      margin: 0;
      font-size: 42px;
      letter-spacing: -0.05em;
      line-height: 1;
    }
    h2, h3 {
      margin: 0;
      letter-spacing: -0.03em;
    }
    .headline {
      display: flex;
      align-items: flex-start;
      justify-content: space-between;
      gap: 24px;
      flex-wrap: wrap;
    }
    .eyebrow {
      display: inline-flex;
      align-items: center;
      gap: 8px;
      margin-bottom: 8px;
      padding: 6px 12px;
      border-radius: 999px;
      font-size: 13px;
      font-weight: 700;
      color: var(--accent);
      background: rgba(63,92,255,0.10);
      border: 1px solid rgba(63,92,255,0.10);
    }
    .subcopy {
      max-width: none;
      margin-top: 6px;
      color: var(--muted);
      font-size: 14px;
      line-height: 1.5;
    }
    .muted { color: var(--muted); }
    .stats {
      display: grid;
      grid-template-columns: repeat(3, minmax(0, 1fr));
      gap: 18px;
      margin: 18px 0 0;
    }
    .stat {
      padding: 20px 22px;
      background: linear-gradient(180deg, rgba(255,255,255,0.96), rgba(248,251,255,0.96));
      border-color: var(--line);
      min-height: 92px;
    }
    .stat-label {
      color: var(--muted);
      font-size: 12px;
      letter-spacing: 0.08em;
    }
    .stat-value {
      margin-top: 8px;
      font-size: 30px;
      font-weight: 700;
      letter-spacing: -0.04em;
      color: var(--ink);
    }
    .stat:nth-child(1) {
      background:
        linear-gradient(180deg, rgba(255,255,255,0.98), rgba(241,246,255,0.98)),
        radial-gradient(circle at right bottom, rgba(64,96,255,0.08), transparent 42%);
    }
    .stat:nth-child(1) .stat-value { color: #2840ba; }
    .stat:nth-child(2) {
      background:
        linear-gradient(180deg, rgba(255,255,255,0.98), rgba(237,251,246,0.98)),
        radial-gradient(circle at right bottom, rgba(16,185,129,0.10), transparent 42%);
    }
    .stat:nth-child(2) .stat-value { color: #0ea86e; }
    .stat:nth-child(3) {
      background:
        linear-gradient(180deg, rgba(255,255,255,0.98), rgba(245,239,255,0.98)),
        radial-gradient(circle at right bottom, rgba(139,92,246,0.10), transparent 42%);
    }
    .stat:nth-child(3) .stat-value { color: #7c3aed; }
    .stat:nth-child(3) {
      background:
        linear-gradient(180deg, rgba(255,255,255,0.98), rgba(255,244,235,0.98)),
        radial-gradient(circle at right bottom, rgba(255,122,26,0.10), transparent 42%);
    }
    .stat:nth-child(3) .stat-value { color: #ff6a00; }
    .stat:nth-child(3) .stat-label { color: #8a653e; }
    .toolbar {
      display: flex;
      gap: 16px;
      margin-top: 16px;
      flex-wrap: wrap;
      align-items: flex-start;
    }
    input[type="text"] {
      flex: 1;
      min-width: 260px;
      padding: 14px 16px;
      border-radius: 14px;
      border: 1px solid var(--line);
      background: rgba(255,255,255,0.92);
      font: inherit;
      color: var(--ink);
      box-shadow: inset 0 1px 0 rgba(255,255,255,0.5);
    }
    input[type="text"]::placeholder {
      color: #96a1bd;
    }
    button, .link-btn {
      border: none;
      border-radius: 14px;
      padding: 13px 18px;
      font: inherit;
      font-weight: 600;
      cursor: pointer;
      background: linear-gradient(135deg, var(--accent) 0%, var(--accent-strong) 100%);
      color: white;
      text-decoration: none;
      display: inline-flex;
      align-items: center;
      justify-content: center;
      box-shadow: 0 12px 28px rgba(47, 73, 234, 0.22);
    }
    .subtle {
      background: rgba(255,255,255,0.92);
      color: var(--accent);
      box-shadow: none;
      border: 1px solid var(--line);
    }
    .grid {
      display: grid;
      grid-template-columns: 360px minmax(0, 1fr);
      gap: 18px;
    }
    .panel {
      padding: 24px;
      min-width: 0;
    }
    .panel-head {
      display: flex;
      align-items: flex-start;
      justify-content: space-between;
      gap: 12px;
      margin-bottom: 16px;
    }
    .panel-head p {
      margin: 6px 0 0;
      color: var(--muted);
      font-size: 14px;
    }
    .sidebar-panel {
      background: linear-gradient(180deg, rgba(24,50,43,0.98), rgba(17,35,30,0.98));
      color: #ecf7f0;
      border-color: rgba(255,255,255,0.08);
    }
    .sidebar-panel .panel-head p,
    .sidebar-panel .muted {
      color: rgba(236,247,240,0.66);
    }
    .main-panel {
      background:
        linear-gradient(180deg, rgba(255,255,255,0.98), rgba(248,251,255,0.96));
    }
    .list {
      display: flex;
      flex-direction: column;
      gap: 12px;
      max-height: 72vh;
      overflow: auto;
      padding-right: 6px;
    }
    .cluster {
      display: block;
      text-decoration: none;
      color: inherit;
      padding: 16px;
      border-radius: 18px;
      border: 1px solid var(--sidebar-line);
      background:
        linear-gradient(180deg, rgba(255,255,255,0.05), rgba(255,255,255,0.03));
      transition: transform 120ms ease, border-color 120ms ease, box-shadow 120ms ease;
    }
    .cluster:hover,
    .cluster.active {
      border-color: rgba(42,160,111,0.60);
      transform: translateY(-1px);
      box-shadow: 0 14px 28px rgba(0, 0, 0, 0.18);
    }
    .cluster.active {
      background:
        linear-gradient(180deg, rgba(42,160,111,0.18), rgba(255,255,255,0.04));
    }
    .title { font-weight: 700; line-height: 1.4; }
    .meta {
      margin-top: 10px;
      color: rgba(236,247,240,0.66);
      font-size: 13px;
      line-height: 1.5;
    }
    .mini-chips {
      display: flex;
      gap: 8px;
      flex-wrap: wrap;
      margin-top: 10px;
    }
    .mini-chip {
      border-radius: 999px;
      padding: 4px 9px;
      font-size: 12px;
      color: #d7f7e6;
      background: rgba(42,160,111,0.16);
      border: 1px solid rgba(42,160,111,0.24);
    }
    .stream {
      display: grid;
      grid-template-columns: 1fr;
      gap: 14px;
    }
    .card {
      border: 1px solid var(--line);
      border-radius: 22px;
      padding: 18px;
      background:
        linear-gradient(180deg, rgba(255,255,255,0.99), rgba(247,250,255,0.96));
      box-shadow: 0 10px 30px rgba(62, 92, 187, 0.06);
    }
    .stack > .card:nth-child(odd) {
      background:
        linear-gradient(180deg, rgba(255,255,255,0.99), rgba(236,243,255,0.99));
    }
    .stack > .card:nth-child(even) {
      background:
        linear-gradient(180deg, rgba(255,255,255,0.99), rgba(244,238,255,0.99));
    }
    .session {
      border-top: 1px solid var(--line);
      padding: 16px 0;
    }
    .session:first-child { border-top: none; padding-top: 0; }
    .session-title {
      font-size: 17px;
      font-weight: 700;
      line-height: 1.45;
      margin-bottom: 8px;
    }
    .session-link {
      color: inherit;
      text-decoration: none;
    }
    .session-link:hover {
      color: var(--accent-strong);
    }
    .session-grid {
      display: grid;
      gap: 8px;
    }
    .session-meta {
      display: flex;
      gap: 10px;
      flex-wrap: wrap;
      align-items: center;
      color: var(--muted);
      font-size: 13px;
    }
    .mono {
      display: inline-flex;
      align-items: center;
      padding: 6px 10px;
      border-radius: 11px;
      background: #f2f6ff;
      border: 1px solid var(--line);
      color: var(--accent-strong);
      font-family: "IBM Plex Mono", "SFMono-Regular", monospace;
      font-size: 12px;
      line-height: 1.4;
      word-break: break-all;
    }
    .command-row {
      display: flex;
      gap: 10px;
      align-items: stretch;
      flex-wrap: wrap;
    }
    .command-box {
      flex: 1;
      min-width: 260px;
      padding: 12px 14px;
      border-radius: 16px;
      border: 1px solid var(--line);
      background: linear-gradient(180deg, #f5f8ff 0%, #eef3ff 100%);
      color: var(--accent-strong);
      font-family: "IBM Plex Mono", "SFMono-Regular", monospace;
      font-size: 12px;
      line-height: 1.6;
      word-break: break-all;
    }
    .copy-btn {
      min-width: 88px;
      padding: 0 14px;
      border: 1px solid var(--line);
      border-radius: 14px;
      background: #ffffff;
      color: var(--ink);
      box-shadow: none;
    }
    .copy-btn:hover {
      border-color: var(--accent);
      color: var(--accent-strong);
      background: #eef3ff;
    }
    .detail-link {
      min-width: 108px;
      padding: 0 16px;
      box-shadow: none;
    }
    .cwd-block {
      padding: 12px 14px;
      border-radius: 16px;
      border: 1px solid var(--line);
      background: #f6f9ff;
      color: #485579;
      font-size: 13px;
      line-height: 1.6;
      word-break: break-word;
    }
    .path {
      color: var(--muted);
      font-size: 13px;
      word-break: break-all;
    }
    .preview {
      margin: 0;
      color: #56627f;
      line-height: 1.65;
    }
    .empty {
      padding: 26px 18px;
      border: 1px dashed var(--line-strong);
      border-radius: 18px;
      color: var(--muted);
      background: rgba(255,255,255,0.5);
    }
    .session {
      border-top: 1px solid var(--line);
      padding: 16px 0;
    }
    .session:first-child { border-top: none; padding-top: 0; }
    .chips {
      display: flex;
      gap: 8px;
      flex-wrap: wrap;
      margin: 14px 0 18px;
    }
    .chip {
      border-radius: 999px;
      padding: 6px 10px;
      background: var(--warm-soft);
      color: #7a4715;
      font-size: 12px;
    }
    .flash {
      margin-top: 14px;
      padding: 12px 14px;
      border-radius: 14px;
      font-size: 14px;
      transition: opacity 220ms ease, transform 220ms ease;
    }
    .flash.hide {
      opacity: 0;
      transform: translateY(-6px);
    }
    .ok {
      background: #e8f5eb;
      color: #14532d;
      border: 1px solid #cce4d1;
    }
    .err {
      background: #fdecec;
      color: #991b1b;
      border: 1px solid #f4c7c7;
    }
    .actions {
      display: flex;
      gap: 10px;
      flex-wrap: wrap;
      margin: 14px 0 18px;
    }
    form.inline { display: inline-flex; gap: 10px; flex-wrap: wrap; width: 100%; }
    .section-label {
      margin: 0 0 14px;
      font-size: 14px;
      font-weight: 700;
      color: var(--muted);
      text-transform: uppercase;
      letter-spacing: 0.08em;
    }
    .stack {
      display: flex;
      flex-direction: column;
      gap: 14px;
    }
    @media (max-width: 980px) {
      .grid { grid-template-columns: 1fr; }
      .list { max-height: none; }
      .stream { grid-template-columns: 1fr; }
      .stats { grid-template-columns: 1fr; }
    }
    @media (max-width: 640px) {
      .shell {
        padding: 18px 14px 28px;
      }
      .hero, .panel, .stat {
        border-radius: 20px;
      }
      h1 {
        font-size: 34px;
      }
    }
  </style>
</head>
<body>
  <div class="shell">
    <section class="hero">
      <div class="headline">
        <div>
          <div class="eyebrow">CSM / Codex Session Manager</div>
          <h1>Session 统计</h1>
          <div class="subcopy">本地统一查看 Codex 历史会话，支持检索、恢复指定 session，并快速判断会话规模。</div>
        </div>
      </div>
      {{if .Message}}<div class="flash ok">{{.Message}}</div>{{end}}
      {{if .Error}}<div class="flash err">{{.Error}}</div>{{end}}
      <div class="stats">
        <div class="stat">
          <div class="stat-label">会话数</div>
          <div class="stat-value">{{.TotalSessions}}</div>
        </div>
        <div class="stat">
          <div class="stat-label">用户消息数</div>
          <div class="stat-value">{{.VisibleUserMessages}}</div>
        </div>
        <div class="stat">
          <div class="stat-label">总消息数</div>
          <div class="stat-value">{{.VisibleTotalMessages}}</div>
        </div>
      </div>
      <div class="toolbar">
        <form action="/" method="get" class="inline">
          <input type="text" name="q" value="{{.Query}}" placeholder="搜索标题、session_id、摘要、cwd、路径">
          <button type="submit">搜索</button>
          <button class="subtle" type="submit" formaction="/refresh" formmethod="post">刷新数据</button>
        </form>
      </div>
    </section>

    <section class="panel main-panel">
        <div class="panel-head">
        <div>
          <h3>{{if .Query}}过滤后的 Session{{else}}全部 Session{{end}}</h3>
          <p>点击某个 session 可进入详情页，查看完整会话内容，并支持在详情页内继续搜索。</p>
        </div>
      </div>
      <div class="stack">
        {{range .Sessions}}
          <article class="card">
            <div class="session-grid">
              <div class="session-meta">
                <span>{{.StartedAt}}</span>
                <span>用户 {{.UserMessageCount}}</span>
                <span>总计 {{.TotalMessageCount}}</span>
                <span>{{.SourceID}}</span>
              </div>
              <div class="session-title"><a class="session-link" href="{{sessionLink .}}">{{.Title}}</a></div>
              <div class="command-row">
                <div class="command-box">{{resumeCommand .}}</div>
                <button type="button" class="copy-btn" data-copy="{{resumeCommand .}}">复制</button>
                <a class="link-btn subtle detail-link" href="{{sessionLink .}}">查看详情</a>
              </div>
              {{if .Preview}}<p class="preview">{{.Preview}}</p>{{end}}
              {{if .CWD}}<div class="cwd-block">{{.CWD}}</div>{{end}}
              <div class="path">{{.FilePath}}</div>
            </div>
          </article>
        {{else}}
          <div class="empty">当前没有可展示的 session。</div>
        {{end}}
      </div>
    </section>
  </div>
  <script>
    document.addEventListener("DOMContentLoaded", function() {
      var flashes = document.querySelectorAll(".flash");
      if (flashes.length === 0) {
        return;
      }

      setTimeout(function() {
        flashes.forEach(function(node) {
          node.classList.add("hide");
        });

        setTimeout(function() {
          flashes.forEach(function(node) {
            node.remove();
          });
        }, 240);
      }, 3000);

      var url = new URL(window.location.href);
      if (url.searchParams.has("message") || url.searchParams.has("error")) {
        url.searchParams.delete("message");
        url.searchParams.delete("error");
        window.history.replaceState({}, "", url.toString());
      }
    });

    document.addEventListener("click", async function(event) {
      var target = event.target;
      if (!(target instanceof HTMLElement) || !target.classList.contains("copy-btn")) {
        return;
      }

      var text = target.getAttribute("data-copy") || "";
      if (!text) {
        return;
      }

      try {
        await navigator.clipboard.writeText(text);
        var original = target.textContent;
        target.textContent = "已复制";
        setTimeout(function() {
          target.textContent = original || "复制";
        }, 1200);
      } catch (error) {
        target.textContent = "复制失败";
        setTimeout(function() {
          target.textContent = "复制";
        }, 1200);
      }
    });
  </script>
</body>
</html>`

const detailTemplate = `<!doctype html>
<html lang="zh-CN">
<head>
  <meta charset="utf-8">
  <meta name="viewport" content="width=device-width,initial-scale=1">
  <title>{{.Detail.Title}} - CSM</title>
  <style>
    :root {
      --bg: #f7f9ff;
      --bg-2: #eef4ff;
      --panel: rgba(255, 255, 255, 0.95);
      --ink: #1d2b57;
      --muted: #6d7898;
      --line: #dbe5fb;
      --accent: #3f5cff;
      --accent-strong: #2f49ea;
      --shadow: 0 20px 60px rgba(62, 92, 187, 0.10);
    }
    * { box-sizing: border-box; }
    body {
      margin: 0;
      font-family: "IBM Plex Sans", "Noto Sans SC", sans-serif;
      color: var(--ink);
      background: linear-gradient(180deg, var(--bg) 0%, var(--bg-2) 100%);
    }
    .shell {
      max-width: 1360px;
      margin: 0 auto;
      padding: 24px 24px 40px;
    }
    .hero, .panel, .event {
      background: var(--panel);
      border: 1px solid var(--line);
      border-radius: 24px;
      box-shadow: var(--shadow);
    }
    .hero {
      padding: 24px;
      margin-bottom: 20px;
      background:
        radial-gradient(circle at top right, rgba(63,92,255,0.16) 0, transparent 24%),
        linear-gradient(135deg, #f4f8ff 0%, #e9f1ff 100%);
    }
    .eyebrow {
      display: inline-flex;
      align-items: center;
      gap: 8px;
      margin-bottom: 8px;
      padding: 6px 12px;
      border-radius: 999px;
      font-size: 13px;
      font-weight: 700;
      color: var(--accent);
      background: rgba(63,92,255,0.10);
      border: 1px solid rgba(63,92,255,0.10);
    }
    h1, h2, h3, h4 { margin: 0; }
    h1 {
      font-size: 34px;
      line-height: 1.15;
      letter-spacing: -0.05em;
    }
    .subcopy {
      margin-top: 8px;
      color: var(--muted);
      line-height: 1.6;
    }
    .meta {
      display: flex;
      gap: 10px;
      flex-wrap: wrap;
      margin-top: 16px;
      color: var(--muted);
      font-size: 13px;
    }
    .overview {
      display: grid;
      grid-template-columns: repeat(3, minmax(0, 1fr));
      gap: 14px;
      margin-top: 18px;
    }
    .overview-card {
      padding: 16px 18px;
      border-radius: 20px;
      border: 1px solid var(--line);
      background: rgba(255,255,255,0.72);
    }
    .overview-label {
      color: var(--muted);
      font-size: 12px;
      letter-spacing: 0.06em;
    }
    .overview-value {
      margin-top: 8px;
      font-size: 28px;
      font-weight: 700;
      letter-spacing: -0.04em;
    }
    .overview-card:nth-child(1) .overview-value { color: #355cff; }
    .overview-card:nth-child(2) .overview-value { color: #0c9b66; }
    .overview-card:nth-child(3) .overview-value { color: #7c3aed; }
    .overview-track {
      margin-top: 12px;
      height: 9px;
      border-radius: 999px;
      background: rgba(196, 210, 243, 0.48);
      overflow: hidden;
    }
    .overview-fill {
      height: 100%;
      border-radius: inherit;
      background: linear-gradient(90deg, #5b7cff 0%, #2f49ea 100%);
    }
    .overview-card:nth-child(2) .overview-fill {
      background: linear-gradient(90deg, #1ac58b 0%, #0c9b66 100%);
    }
    .overview-card:nth-child(3) .overview-fill {
      background: linear-gradient(90deg, #9a6bff 0%, #7c3aed 100%);
    }
    .overview-note {
      margin-top: 10px;
      color: var(--muted);
      font-size: 12px;
    }
    .mono {
      display: inline-flex;
      align-items: center;
      padding: 6px 10px;
      border-radius: 11px;
      background: #f2f6ff;
      border: 1px solid var(--line);
      color: var(--accent-strong);
      font-family: "IBM Plex Mono", "SFMono-Regular", monospace;
      font-size: 12px;
      line-height: 1.4;
      word-break: break-all;
    }
    .command-row {
      display: flex;
      gap: 10px;
      align-items: stretch;
      flex-wrap: wrap;
      margin-top: 16px;
    }
    .command-box {
      flex: 1;
      min-width: 260px;
      padding: 12px 14px;
      border-radius: 16px;
      border: 1px solid var(--line);
      background: linear-gradient(180deg, #f5f8ff 0%, #eef3ff 100%);
      color: var(--accent-strong);
      font-family: "IBM Plex Mono", "SFMono-Regular", monospace;
      font-size: 12px;
      line-height: 1.6;
      word-break: break-all;
    }
    .toolbar {
      display: flex;
      gap: 16px;
      margin-top: 18px;
      flex-wrap: wrap;
      align-items: center;
    }
    .inline {
      display: flex;
      gap: 12px;
      flex-wrap: wrap;
      width: 100%;
    }
    input[type="text"] {
      flex: 1;
      min-width: 280px;
      padding: 14px 16px;
      border-radius: 14px;
      border: 1px solid var(--line);
      background: rgba(255,255,255,0.92);
      font: inherit;
      color: var(--ink);
    }
    input[type="text"]::placeholder {
      color: #96a1bd;
    }
    button, .link-btn {
      border: none;
      border-radius: 14px;
      padding: 13px 18px;
      font: inherit;
      font-weight: 600;
      cursor: pointer;
      background: linear-gradient(135deg, var(--accent) 0%, var(--accent-strong) 100%);
      color: white;
      text-decoration: none;
      display: inline-flex;
      align-items: center;
      justify-content: center;
      box-shadow: 0 12px 28px rgba(47, 73, 234, 0.22);
    }
    .subtle {
      background: rgba(255,255,255,0.92);
      color: var(--accent);
      box-shadow: none;
      border: 1px solid var(--line);
    }
    .panel {
      padding: 24px;
    }
    .panel-head {
      display: flex;
      align-items: flex-start;
      justify-content: space-between;
      gap: 12px;
      margin-bottom: 16px;
    }
    .panel-head p {
      margin: 6px 0 0;
      color: var(--muted);
      font-size: 14px;
    }
    .events {
      display: flex;
      flex-direction: column;
      gap: 16px;
    }
    .event {
      width: 100%;
      position: relative;
      padding: 18px;
      background:
        linear-gradient(180deg, rgba(255,255,255,0.99), rgba(247,250,255,0.96));
    }
    .event:nth-child(odd) {
      background:
        linear-gradient(180deg, rgba(255,255,255,0.99), rgba(236,243,255,0.99));
    }
    .event-user,
    .event-assistant {
      max-width: min(860px, 78%);
    }
    .event-user {
      align-self: flex-end;
      border-radius: 24px 24px 10px 24px;
      border-color: rgba(63,92,255,0.34);
      background:
        linear-gradient(180deg, rgba(223,234,255,0.99), rgba(206,221,255,0.99));
      box-shadow:
        0 14px 34px rgba(63,92,255,0.14),
        inset 0 0 0 1px rgba(63,92,255,0.06);
    }
    .event-assistant {
      align-self: flex-start;
      border-radius: 24px 24px 24px 10px;
      border-color: rgba(12,155,102,0.22);
      background:
        linear-gradient(180deg, rgba(242,253,248,0.99), rgba(231,248,239,0.99));
      box-shadow:
        0 12px 34px rgba(12,155,102,0.08),
        inset 0 0 0 1px rgba(12,155,102,0.03);
    }
    .event-developer,
    .event-tool,
    .event-system {
      align-self: stretch;
      max-width: 100%;
      border-radius: 22px;
    }
    .event-user .event-title {
      color: #2747d9;
    }
    .event-user .event-content {
      color: #1d3670;
    }
    .event-assistant .event-title {
      color: #12744f;
    }
    .event-assistant .event-content {
      color: #215f4a;
    }
    .event-tool {
      background:
        linear-gradient(180deg, rgba(251,247,255,0.99), rgba(243,235,255,0.99));
      border-color: rgba(124,58,237,0.16);
    }
    .event-developer {
      background:
        linear-gradient(180deg, rgba(255,249,243,0.99), rgba(255,241,231,0.99));
      border-color: rgba(180,83,9,0.18);
    }
    .event-system {
      background:
        linear-gradient(180deg, rgba(248,250,255,0.99), rgba(239,243,251,0.99));
      border-color: rgba(86,98,127,0.16);
    }
    .event-head {
      display: flex;
      gap: 10px;
      flex-wrap: wrap;
      align-items: center;
      margin-bottom: 12px;
      font-size: 13px;
      color: var(--muted);
    }
    .role {
      padding: 5px 10px;
      border-radius: 999px;
      font-weight: 700;
      font-size: 12px;
    }
    .role-user { background: #e8f0ff; color: #355cff; }
    .role-assistant { background: #e9fbf4; color: #0c9b66; }
    .role-developer { background: #fff0e8; color: #b45309; }
    .role-tool { background: #f4ebff; color: #7c3aed; }
    .role-system { background: #eef2f8; color: #56627f; }
    .event-title {
      font-size: 16px;
      font-weight: 700;
      margin-bottom: 10px;
    }
    .event-content {
      margin: 0;
      white-space: pre-wrap;
      word-break: break-word;
      line-height: 1.7;
      color: #44506d;
    }
    .tool-details {
      margin-top: 8px;
      border: 1px solid rgba(124,58,237,0.14);
      border-radius: 16px;
      background: rgba(255,255,255,0.58);
      overflow: hidden;
    }
    .tool-details summary {
      list-style: none;
      cursor: pointer;
      padding: 12px 14px;
      color: #6d5c9b;
      font-weight: 600;
      user-select: none;
    }
    .tool-details summary::-webkit-details-marker {
      display: none;
    }
    .tool-details summary::after {
      content: "展开";
      float: right;
      color: #7c3aed;
      font-size: 12px;
    }
    .tool-details[open] summary::after {
      content: "收起";
    }
    .tool-details .event-content {
      padding: 0 14px 14px;
    }
    .empty {
      padding: 26px 18px;
      border: 1px dashed #c5d3f4;
      border-radius: 18px;
      color: var(--muted);
      background: rgba(255,255,255,0.5);
    }
    .copy-btn {
      min-width: 88px;
      padding: 0 14px;
      border: 1px solid var(--line);
      border-radius: 14px;
      background: #ffffff;
      color: var(--ink);
      box-shadow: none;
    }
    .copy-btn:hover {
      border-color: var(--accent);
      color: var(--accent-strong);
      background: #eef3ff;
    }
    @media (max-width: 640px) {
      .shell { padding: 18px 14px 28px; }
      .hero, .panel, .event { border-radius: 20px; }
      h1 { font-size: 28px; }
      .overview { grid-template-columns: 1fr; }
    }
  </style>
</head>
<body>
  <div class="shell">
    <section class="hero">
      <div class="eyebrow">CSM / Session Detail</div>
      <h1>{{.Detail.Title}}</h1>
      <div class="subcopy">这里展示当前 session 的完整事件流。可按关键词过滤用户消息、助手回复、工具调用和系统事件。</div>
      <div class="meta">
        <span>{{.Detail.StartedAt}}</span>
        <span>{{.Entry.SourceID}}</span>
      </div>
      <div class="overview">
        <div class="overview-card">
          <div class="overview-label">用户消息数</div>
          <div class="overview-value">{{.Entry.UserMessageCount}}</div>
          <div class="overview-track"><div class="overview-fill" style="width: {{barPercent .Entry.UserMessageCount .Entry.TotalMessageCount}}%;"></div></div>
          <div class="overview-note">占总消息 {{percentValue .Entry.UserMessageCount .Entry.TotalMessageCount}}%</div>
        </div>
        <div class="overview-card">
          <div class="overview-label">总消息数</div>
          <div class="overview-value">{{.Entry.TotalMessageCount}}</div>
          <div class="overview-track"><div class="overview-fill" style="width: 100%;"></div></div>
          <div class="overview-note">当前索引统计到的全部消息</div>
        </div>
        <div class="overview-card">
          <div class="overview-label">当前展示事件</div>
          <div class="overview-value">{{.VisibleCount}}</div>
          <div class="overview-track"><div class="overview-fill" style="width: {{barPercent .VisibleCount .TotalEvents}}%;"></div></div>
          <div class="overview-note">共 {{.TotalEvents}} 条事件</div>
        </div>
      </div>
      <div class="meta">
        {{if .Detail.CWD}}<span class="mono">{{.Detail.CWD}}</span>{{end}}
        <span class="mono">{{.Detail.FilePath}}</span>
      </div>
      <div class="command-row">
        <div class="command-box">{{resumeCommand .Entry}}</div>
        <button type="button" class="copy-btn" data-copy="{{resumeCommand .Entry}}">复制</button>
        <a class="link-btn subtle" href="/">返回列表</a>
      </div>
      <div class="toolbar">
        <form action="/session" method="get" class="inline">
          <input type="hidden" name="id" value="{{.Entry.SessionID}}">
          <input type="text" name="q" value="{{.Query}}" placeholder="搜索详情内容、角色、标题、时间">
          <button type="submit">搜索</button>
          <a class="link-btn subtle" href="/session?id={{.Entry.SessionID}}">清空搜索</a>
        </form>
      </div>
    </section>

    <section class="panel">
      <div class="panel-head">
        <div>
          <h3>{{if .Query}}过滤后的会话内容{{else}}会话全部内容{{end}}</h3>
          <p>按原始 session 文件动态解析，不依赖轻量索引缓存全文。</p>
        </div>
      </div>
      <div class="events">
        {{range .Events}}
          <article class="event event-{{.Role}}">
            <div class="event-head">
              <span class="role role-{{.Role}}">{{roleLabel .Role}}</span>
              <span>#{{.Index}}</span>
              <span>{{.Timestamp}}</span>
              <span>{{.Kind}}</span>
            </div>
            {{if .Title}}<div class="event-title">{{.Title}}</div>{{end}}
            {{if eq .Role "tool"}}
              <details class="tool-details">
                <summary>查看工具输出</summary>
                <pre class="event-content">{{.Content}}</pre>
              </details>
            {{else}}
              <pre class="event-content">{{.Content}}</pre>
            {{end}}
          </article>
        {{else}}
          <div class="empty">当前搜索条件下没有匹配到任何会话内容。</div>
        {{end}}
      </div>
    </section>
  </div>
  <script>
    document.addEventListener("click", async function(event) {
      var target = event.target;
      if (!(target instanceof HTMLElement) || !target.classList.contains("copy-btn")) {
        return;
      }

      var text = target.getAttribute("data-copy") || "";
      if (!text) {
        return;
      }

      try {
        await navigator.clipboard.writeText(text);
        var original = target.textContent;
        target.textContent = "已复制";
        setTimeout(function() {
          target.textContent = original || "复制";
        }, 1200);
      } catch (error) {
        target.textContent = "复制失败";
        setTimeout(function() {
          target.textContent = "复制";
        }, 1200);
      }
    });
  </script>
</body>
</html>`
