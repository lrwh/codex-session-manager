package parser

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/liurui/codex-session-manager/internal/model"
)

type detailEnvelope struct {
	Timestamp string          `json:"timestamp"`
	Type      string          `json:"type"`
	Payload   json.RawMessage `json:"payload"`
}

type functionCallPayload struct {
	Name      string `json:"name"`
	Arguments string `json:"arguments"`
}

type functionCallOutputPayload struct {
	CallID string `json:"call_id"`
	Output string `json:"output"`
}

type turnContextPayload struct {
	CWD          string `json:"cwd"`
	CurrentDate  string `json:"current_date"`
	Timezone     string `json:"timezone"`
	Model        string `json:"model"`
	Effort       string `json:"effort"`
	UserSettings string `json:"user_instructions"`
}

func ParseSessionDetail(path string) (model.SessionDetail, error) {
	file, err := os.Open(path)
	if err != nil {
		return model.SessionDetail{}, err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	scanner.Buffer(make([]byte, 1024), 16*1024*1024)

	detail := model.SessionDetail{
		FilePath: path,
		Events:   make([]model.SessionDetailEvent, 0, 128),
	}

	for scanner.Scan() {
		line := scanner.Bytes()
		if len(line) == 0 {
			continue
		}

		var envelope detailEnvelope
		if err := json.Unmarshal(line, &envelope); err != nil {
			continue
		}

		switch envelope.Type {
		case "session_meta":
			var payload sessionMetaPayload
			if err := json.Unmarshal(envelope.Payload, &payload); err != nil {
				continue
			}
			detail.SessionID = payload.ID
			detail.StartedAt = payload.Timestamp
			detail.CWD = payload.CWD
		case "response_item":
			event, ok := buildResponseDetailEvent(envelope)
			if ok {
				addDetailEvent(&detail.Events, event)
			}
		case "event_msg":
			event, ok := buildEventMessageDetailEvent(envelope)
			if ok {
				addDetailEvent(&detail.Events, event)
			}
		case "turn_context":
			event, ok := buildTurnContextDetailEvent(envelope)
			if ok {
				addDetailEvent(&detail.Events, event)
			}
		}
	}

	if err := scanner.Err(); err != nil {
		return model.SessionDetail{}, err
	}

	for index := range detail.Events {
		detail.Events[index].Index = index + 1
	}
	return detail, nil
}

func buildResponseDetailEvent(envelope detailEnvelope) (model.SessionDetailEvent, bool) {
	var header struct {
		Type string `json:"type"`
		Role string `json:"role"`
		Name string `json:"name"`
	}
	if err := json.Unmarshal(envelope.Payload, &header); err != nil {
		return model.SessionDetailEvent{}, false
	}

	switch header.Type {
	case "message":
		text := extractResponseMessageText(envelope.Payload, header.Role)
		if strings.TrimSpace(text) == "" {
			return model.SessionDetailEvent{}, false
		}
		title := ""
		if header.Role == "developer" {
			title = "开发者指令"
		}
		return model.SessionDetailEvent{
			Timestamp: envelope.Timestamp,
			Role:      normalizeRole(header.Role),
			Kind:      "message",
			Title:     title,
			Content:   text,
		}, true
	case "function_call":
		var payload functionCallPayload
		if err := json.Unmarshal(envelope.Payload, &payload); err != nil {
			return model.SessionDetailEvent{}, false
		}
		content := strings.TrimSpace(payload.Arguments)
		if content == "" {
			content = "无参数"
		}
		return model.SessionDetailEvent{
			Timestamp: envelope.Timestamp,
			Role:      "tool",
			Kind:      "function_call",
			Title:     payload.Name,
			Content:   content,
		}, true
	case "function_call_output":
		var payload functionCallOutputPayload
		if err := json.Unmarshal(envelope.Payload, &payload); err != nil {
			return model.SessionDetailEvent{}, false
		}
		content := strings.TrimSpace(payload.Output)
		if content == "" {
			content = "无输出"
		}
		return model.SessionDetailEvent{
			Timestamp: envelope.Timestamp,
			Role:      "tool",
			Kind:      "function_call_output",
			Title:     "工具输出",
			Content:   content,
		}, true
	case "reasoning":
		return model.SessionDetailEvent{
			Timestamp: envelope.Timestamp,
			Role:      "system",
			Kind:      "reasoning",
			Title:     "Reasoning",
			Content:   "内部 reasoning 事件，原始内容已加密。",
		}, true
	default:
		return model.SessionDetailEvent{}, false
	}
}

func buildEventMessageDetailEvent(envelope detailEnvelope) (model.SessionDetailEvent, bool) {
	var payload eventMessagePayload
	if err := json.Unmarshal(envelope.Payload, &payload); err != nil {
		return model.SessionDetailEvent{}, false
	}

	switch payload.Type {
	case "thread_name_updated":
		if payload.ThreadName == "" {
			return model.SessionDetailEvent{}, false
		}
		return model.SessionDetailEvent{
			Timestamp: envelope.Timestamp,
			Role:      "system",
			Kind:      "event_msg",
			Title:     "会话名称更新",
			Content:   payload.ThreadName,
		}, true
	case "task_started":
		return model.SessionDetailEvent{
			Timestamp: envelope.Timestamp,
			Role:      "system",
			Kind:      "event_msg",
			Title:     "任务开始",
			Content:   "新一轮任务已开始。",
		}, true
	case "token_count":
		return model.SessionDetailEvent{
			Timestamp: envelope.Timestamp,
			Role:      "system",
			Kind:      "event_msg",
			Title:     "Token 统计",
			Content:   "本轮产生了 token 统计信息。",
		}, true
	default:
		return model.SessionDetailEvent{}, false
	}
}

func buildTurnContextDetailEvent(envelope detailEnvelope) (model.SessionDetailEvent, bool) {
	var payload turnContextPayload
	if err := json.Unmarshal(envelope.Payload, &payload); err != nil {
		return model.SessionDetailEvent{}, false
	}

	parts := make([]string, 0, 5)
	if payload.CWD != "" {
		parts = append(parts, "cwd: "+payload.CWD)
	}
	if payload.Model != "" {
		parts = append(parts, "model: "+payload.Model)
	}
	if payload.Effort != "" {
		parts = append(parts, "effort: "+payload.Effort)
	}
	if payload.CurrentDate != "" {
		parts = append(parts, "date: "+payload.CurrentDate)
	}
	if payload.Timezone != "" {
		parts = append(parts, "timezone: "+payload.Timezone)
	}
	if len(parts) == 0 {
		return model.SessionDetailEvent{}, false
	}

	return model.SessionDetailEvent{
		Timestamp: envelope.Timestamp,
		Role:      "system",
		Kind:      "turn_context",
		Title:     "Turn Context",
		Content:   strings.Join(parts, "\n"),
	}, true
}

func extractResponseMessageText(raw json.RawMessage, role string) string {
	var payload responseItemPayload
	if err := json.Unmarshal(raw, &payload); err != nil {
		return ""
	}
	if payload.Type != "message" || payload.Role != role {
		return ""
	}

	parts := make([]string, 0, len(payload.Content))
	for _, item := range payload.Content {
		switch item.Type {
		case "input_text", "output_text":
			text := strings.TrimSpace(item.Text)
			if text != "" {
				parts = append(parts, text)
			}
		}
	}

	return strings.TrimSpace(strings.Join(parts, "\n\n"))
}

func normalizeRole(role string) string {
	switch strings.ToLower(strings.TrimSpace(role)) {
	case "user":
		return "user"
	case "assistant":
		return "assistant"
	case "developer":
		return "developer"
	case "system":
		return "system"
	default:
		return "system"
	}
}

func addDetailEvent(events *[]model.SessionDetailEvent, event model.SessionDetailEvent) {
	event.Content = strings.TrimSpace(event.Content)
	if event.Content == "" {
		return
	}

	current := *events
	if len(current) > 0 {
		last := current[len(current)-1]
		if last.Role == event.Role && last.Title == event.Title && last.Content == event.Content {
			return
		}
	}
	*events = append(*events, event)
}

func FormatDetailSearchText(event model.SessionDetailEvent) string {
	return strings.ToLower(strings.Join([]string{
		event.Role,
		event.Kind,
		event.Title,
		event.Content,
		fmt.Sprintf("%d", event.Index),
	}, "\n"))
}
