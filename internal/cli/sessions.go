package cli

import (
	"encoding/json"
	"fmt"
	"strings"
	"text/tabwriter"
	"unicode/utf8"

	"github.com/liurui/codex-session-manager/internal/app"
	"github.com/liurui/codex-session-manager/internal/model"
	"github.com/spf13/cobra"
)

func newSessionsCommand() *cobra.Command {
	var limit int
	var asJSON bool
	var verbose bool

	command := &cobra.Command{
		Use:   "sessions",
		Short: "列出全部 session 信息",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runSessions(cmd, limit, asJSON, verbose)
		},
	}

	command.Flags().IntVarP(&limit, "limit", "n", 0, "限制返回数量，默认 0 表示全部")
	command.Flags().BoolVar(&asJSON, "json", false, "以 JSON 输出")
	command.Flags().BoolVar(&verbose, "verbose", false, "输出详细信息")
	return command
}

func runSessions(cmd *cobra.Command, limit int, asJSON, verbose bool) error {
	application, err := app.New()
	if err != nil {
		return err
	}

	sessions, err := application.ListSessions(limit)
	if err != nil {
		return err
	}

	if asJSON {
		encoder := json.NewEncoder(cmd.OutOrStdout())
		for _, session := range sessions {
			if err := encoder.Encode(session); err != nil {
				return err
			}
		}
		return nil
	}

	if verbose {
		renderSessionsVerbose(cmd, sessions)
		return nil
	}

	renderSessionsTable(cmd, sessions)
	return nil
}

func renderSessionsTable(cmd *cobra.Command, sessions []model.SessionIndexEntry) {
	if len(sessions) == 0 {
		fmt.Fprintln(cmd.OutOrStdout(), "当前没有 session")
		return
	}

	writer := tabwriter.NewWriter(cmd.OutOrStdout(), 0, 0, 2, ' ', 0)
	fmt.Fprintln(writer, "TIME\tSESSION_ID\tTITLE\tCWD")
	for index, session := range sessions {
		_ = index
		fmt.Fprintf(
			writer,
			"%s\t%s\t%s\t%s\n",
			session.StartedAt,
			session.SessionID,
			shortenCell(session.Title, 60),
			shortenCell(session.CWD, 36),
		)
	}
	_ = writer.Flush()
}

func renderSessionsVerbose(cmd *cobra.Command, sessions []model.SessionIndexEntry) {
	if len(sessions) == 0 {
		fmt.Fprintln(cmd.OutOrStdout(), "当前没有 session")
		return
	}

	for index, session := range sessions {
		fmt.Fprintf(cmd.OutOrStdout(), "[%d] %s\n", index+1, session.StartedAt)
		fmt.Fprintf(cmd.OutOrStdout(), "session_id: %s\n", session.SessionID)
		fmt.Fprintf(cmd.OutOrStdout(), "title: %s\n", session.Title)
		fmt.Fprintf(cmd.OutOrStdout(), "user_messages: %d\n", session.UserMessageCount)
		fmt.Fprintf(cmd.OutOrStdout(), "total_messages: %d\n", session.TotalMessageCount)
		if session.CWD != "" {
			fmt.Fprintf(cmd.OutOrStdout(), "cwd: %s\n", session.CWD)
		}
		fmt.Fprintf(cmd.OutOrStdout(), "file: %s\n", session.FilePath)
		if session.Preview != "" {
			fmt.Fprintf(cmd.OutOrStdout(), "preview: %s\n", session.Preview)
		}
		fmt.Fprintln(cmd.OutOrStdout())
	}
}

func shortenCell(value string, limit int) string {
	value = strings.TrimSpace(value)
	if limit <= 0 || value == "" {
		return value
	}
	if utf8.RuneCountInString(value) <= limit {
		return value
	}
	runes := []rune(value)
	return string(runes[:limit]) + "..."
}
