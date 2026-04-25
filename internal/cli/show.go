package cli

import (
	"fmt"

	"github.com/liurui/codex-session-manager/internal/app"
	"github.com/spf13/cobra"
)

func newShowCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "show <cluster-id>",
		Short: "查看某个 cluster 下的 session",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			application, err := app.New()
			if err != nil {
				return err
			}

			view, err := application.ShowCluster(args[0])
			if err != nil {
				return err
			}

			fmt.Fprintf(cmd.OutOrStdout(), "cluster: %s\n", view.Cluster.ClusterID)
			fmt.Fprintf(cmd.OutOrStdout(), "sessions: %d\n", view.Cluster.SessionCount)
			fmt.Fprintf(cmd.OutOrStdout(), "latest: %s\n", view.Cluster.LatestStartedAt)
			title := view.Cluster.RepresentativeTitle
			if view.Cluster.Name != "" {
				title = view.Cluster.Name
			}
			fmt.Fprintf(cmd.OutOrStdout(), "title: %s\n", title)
			if len(view.Cluster.Projects) > 0 {
				fmt.Fprintf(cmd.OutOrStdout(), "projects: %v\n", view.Cluster.Projects)
			}
			if len(view.Cluster.TopKeywords) > 0 {
				fmt.Fprintf(cmd.OutOrStdout(), "keywords: %v\n", view.Cluster.TopKeywords)
			}
			fmt.Fprintln(cmd.OutOrStdout())

			for index, session := range view.Sessions {
				fmt.Fprintf(
					cmd.OutOrStdout(),
					"[%d] %s %s\n",
					index+1,
					session.StartedAt,
					session.SessionID,
				)
				fmt.Fprintf(cmd.OutOrStdout(), "title: %s\n", session.Title)
				if session.CWD != "" {
					fmt.Fprintf(cmd.OutOrStdout(), "cwd: %s\n", session.CWD)
				}
				fmt.Fprintf(cmd.OutOrStdout(), "file: %s\n\n", session.FilePath)
			}
			return nil
		},
	}
}
