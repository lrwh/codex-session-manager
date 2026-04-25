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
	}).Parse(pageTemplate))
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
          <p>列表固定展示最新时间、标题、恢复命令、摘要、cwd、用户消息数、总消息数和路径。</p>
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
              <div class="session-title">{{.Title}}</div>
              <div class="command-row">
                <div class="command-box">{{resumeCommand .}}</div>
                <button type="button" class="copy-btn" data-copy="{{resumeCommand .}}">复制</button>
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
