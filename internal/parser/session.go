package parser

import (
	"bufio"
	"crypto/sha1"
	"encoding/hex"
	"encoding/json"
	"io"
	"os"
	"strings"
)

type SessionSummary struct {
	SessionID         string
	StartedAt         string
	CWD               string
	ThreadName        string
	Title             string
	Preview           string
	Commands          []string
	Keywords          []string
	Projects          []string
	UserMessageCount  int
	TotalMessageCount int
	ContentHash       string
}

type sessionEnvelope struct {
	Type    string          `json:"type"`
	Payload json.RawMessage `json:"payload"`
}

type sessionMetaPayload struct {
	ID        string `json:"id"`
	Timestamp string `json:"timestamp"`
	CWD       string `json:"cwd"`
}

type responseItemPayload struct {
	Type      string                `json:"type"`
	Role      string                `json:"role"`
	Name      string                `json:"name"`
	Arguments string                `json:"arguments"`
	Content   []responseContentItem `json:"content"`
}

type responseContentItem struct {
	Type string `json:"type"`
	Text string `json:"text"`
}

type eventMessagePayload struct {
	Type       string `json:"type"`
	Message    string `json:"message"`
	ThreadName string `json:"thread_name"`
}

type execCommandArguments struct {
	Cmd string `json:"cmd"`
}

func ParseSessionFile(path string) (SessionSummary, error) {
	file, err := os.Open(path)
	if err != nil {
		return SessionSummary{}, err
	}
	defer file.Close()

	hasher := sha1.New()
	reader := io.TeeReader(file, hasher)

	scanner := bufio.NewScanner(reader)
	scanner.Buffer(make([]byte, 1024), 16*1024*1024)

	var summary SessionSummary
	eventUserMessages := make([]string, 0, 8)
	eventAgentMessages := make([]string, 0, 8)
	responseUserMessages := make([]string, 0, 8)
	responseAssistantMessages := make([]string, 0, 8)
	commands := make([]string, 0, 8)
	for scanner.Scan() {
		line := scanner.Bytes()
		if len(line) == 0 {
			continue
		}

		var envelope sessionEnvelope
		if err := json.Unmarshal(line, &envelope); err != nil {
			continue
		}

		if envelope.Type != "session_meta" {
			switch envelope.Type {
			case "response_item":
				userText := extractResponseUserMessage(envelope.Payload)
				if userText != "" {
					responseUserMessages = appendMessage(responseUserMessages, userText)
				}
				assistantText := extractResponseAssistantMessage(envelope.Payload)
				if assistantText != "" {
					responseAssistantMessages = appendMessage(responseAssistantMessages, assistantText)
				}
				command := extractFunctionCallCommand(envelope.Payload)
				if command != "" {
					commands = appendMessage(commands, command)
				}
			case "event_msg":
				userText := extractEventMessageByType(envelope.Payload, "user_message")
				if userText != "" {
					eventUserMessages = appendMessage(eventUserMessages, userText)
				}
				agentText := extractEventMessageByType(envelope.Payload, "agent_message")
				if agentText != "" {
					eventAgentMessages = appendMessage(eventAgentMessages, agentText)
				}
				threadName := extractThreadName(envelope.Payload)
				if threadName != "" {
					summary.ThreadName = threadName
				}
			}
			continue
		}

		var payload sessionMetaPayload
		if err := json.Unmarshal(envelope.Payload, &payload); err != nil {
			continue
		}

		summary.SessionID = payload.ID
		summary.StartedAt = payload.Timestamp
		summary.CWD = payload.CWD
	}
	if err := scanner.Err(); err != nil {
		return SessionSummary{}, err
	}

	messages := eventUserMessages
	if len(messages) == 0 {
		messages = responseUserMessages
	}

	if len(messages) > 0 {
		combined := strings.Join(messages, "\n")
		summary.Title = selectSessionTitle(messages)
		summary.Preview = shorten(combined, 200)
		summary.Keywords = collectKeywords(combined)
		summary.Projects = collectProjects(combined)
		summary.UserMessageCount = len(messages)
	}

	if len(eventUserMessages) > 0 || len(eventAgentMessages) > 0 {
		summary.TotalMessageCount = len(eventUserMessages) + len(eventAgentMessages)
	} else {
		summary.TotalMessageCount = len(responseUserMessages) + len(responseAssistantMessages)
	}
	if summary.TotalMessageCount < summary.UserMessageCount {
		summary.TotalMessageCount = summary.UserMessageCount
	}
	if len(commands) > 6 {
		commands = commands[:6]
	}
	summary.Commands = commands
	summary.ContentHash = hex.EncodeToString(hasher.Sum(nil))
	return summary, nil
}

func extractResponseUserMessage(raw json.RawMessage) string {
	var payload responseItemPayload
	if err := json.Unmarshal(raw, &payload); err != nil {
		return ""
	}
	if payload.Type != "message" || payload.Role != "user" {
		return ""
	}

	parts := make([]string, 0, len(payload.Content))
	for _, item := range payload.Content {
		if item.Type == "input_text" && strings.TrimSpace(item.Text) != "" {
			parts = append(parts, strings.TrimSpace(item.Text))
		}
	}

	return normalizeWhitespace(strings.Join(parts, " "))
}

func extractResponseAssistantMessage(raw json.RawMessage) string {
	var payload responseItemPayload
	if err := json.Unmarshal(raw, &payload); err != nil {
		return ""
	}
	if payload.Type != "message" || payload.Role != "assistant" {
		return ""
	}

	parts := make([]string, 0, len(payload.Content))
	for _, item := range payload.Content {
		if item.Type == "output_text" && strings.TrimSpace(item.Text) != "" {
			parts = append(parts, strings.TrimSpace(item.Text))
		}
	}

	return normalizeWhitespace(strings.Join(parts, " "))
}

func extractFunctionCallCommand(raw json.RawMessage) string {
	var payload responseItemPayload
	if err := json.Unmarshal(raw, &payload); err != nil {
		return ""
	}
	if payload.Type != "function_call" || payload.Name != "exec_command" {
		return ""
	}

	var arguments execCommandArguments
	if err := json.Unmarshal([]byte(payload.Arguments), &arguments); err != nil {
		return ""
	}
	return normalizeWhitespace(arguments.Cmd)
}

func extractEventMessageByType(raw json.RawMessage, eventType string) string {
	var payload eventMessagePayload
	if err := json.Unmarshal(raw, &payload); err != nil {
		return ""
	}
	if payload.Type != eventType {
		return ""
	}
	return normalizeWhitespace(payload.Message)
}

func extractThreadName(raw json.RawMessage) string {
	var payload eventMessagePayload
	if err := json.Unmarshal(raw, &payload); err != nil {
		return ""
	}
	if payload.Type != "thread_name_updated" {
		return ""
	}
	return normalizeWhitespace(payload.ThreadName)
}

func appendMessage(messages []string, text string) []string {
	if text == "" {
		return messages
	}
	if len(messages) > 0 && messages[len(messages)-1] == text {
		return messages
	}
	return append(messages, text)
}
