package cli

import (
	"fmt"

	"github.com/liurui/codex-session-manager/internal/app"
	"github.com/spf13/cobra"
)

func newFindCommand() *cobra.Command {
	var limit int

	command := &cobra.Command{
		Use:   "find <query>",
		Short: "搜索历史 session",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			application, err := app.New()
			if err != nil {
				return err
			}

			results, err := application.Find(args[0], limit)
			if err != nil {
				return err
			}

			if len(results) == 0 {
				fmt.Fprintln(cmd.OutOrStdout(), "没有找到匹配的 session")
				return nil
			}

			for index, result := range results {
				entry := result.Entry
				fmt.Fprintf(
					cmd.OutOrStdout(),
					"[%d] score=%d session=%s source=%s time=%s\n",
					index+1,
					result.Score,
					entry.SessionID,
					entry.SourceID,
					entry.StartedAt,
				)
				fmt.Fprintf(cmd.OutOrStdout(), "title: %s\n", entry.Title)
				if entry.CWD != "" {
					fmt.Fprintf(cmd.OutOrStdout(), "cwd: %s\n", entry.CWD)
				}
				if entry.Preview != "" {
					fmt.Fprintf(cmd.OutOrStdout(), "preview: %s\n", entry.Preview)
				}
				fmt.Fprintf(cmd.OutOrStdout(), "file: %s\n\n", entry.FilePath)
			}
			return nil
		},
	}

	command.Flags().IntVarP(&limit, "limit", "n", 10, "返回结果数量")
	return command
}
