package cli

import (
	"fmt"

	"github.com/liurui/codex-session-manager/internal/app"
	"github.com/spf13/cobra"
)

func newClusterCommand() *cobra.Command {
	command := &cobra.Command{
		Use:   "cluster",
		Short: "管理 session 聚类",
	}

	command.AddCommand(newClusterRebuildCommand())
	command.AddCommand(newClusterListCommand())
	command.AddCommand(newClusterMergeCommand())
	command.AddCommand(newClusterSplitCommand())
	command.AddCommand(newClusterResetCommand())
	return command
}

func newClusterRebuildCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "rebuild",
		Short: "重建 clusters.json",
		RunE: func(cmd *cobra.Command, args []string) error {
			application, err := app.New()
			if err != nil {
				return err
			}

			result, err := application.RebuildClusters()
			if err != nil {
				return err
			}

			fmt.Fprintf(
				cmd.OutOrStdout(),
				"聚类完成: clusters=%d output=%s\n",
				result.ClusterCount,
				result.OutputFile,
			)
			return nil
		},
	}
}

func newClusterListCommand() *cobra.Command {
	var limit int

	command := &cobra.Command{
		Use:   "list",
		Short: "列出聚类结果",
		RunE: func(cmd *cobra.Command, args []string) error {
			application, err := app.New()
			if err != nil {
				return err
			}

			clusters, err := application.LoadClusters()
			if err != nil {
				return err
			}

			if limit > 0 && len(clusters) > limit {
				clusters = clusters[:limit]
			}

			if len(clusters) == 0 {
				fmt.Fprintln(cmd.OutOrStdout(), "当前没有聚类结果")
				return nil
			}

			fmt.Fprintln(cmd.OutOrStdout(), "ID\tSESSIONS\tLATEST\tTITLE")
			for _, cluster := range clusters {
				title := cluster.RepresentativeTitle
				if cluster.Name != "" {
					title = cluster.Name
				}
				fmt.Fprintf(
					cmd.OutOrStdout(),
					"%s\t%d\t%s\t%s\n",
					cluster.ClusterID,
					cluster.SessionCount,
					cluster.LatestStartedAt,
					title,
				)
			}
			return nil
		},
	}

	command.Flags().IntVarP(&limit, "limit", "n", 20, "返回结果数量")
	return command
}

func newClusterMergeCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "merge <target-cluster-id> <source-cluster-id...>",
		Short: "把多个 cluster 合并到 target cluster",
		Args:  cobra.MinimumNArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			application, err := app.New()
			if err != nil {
				return err
			}

			if err := application.AddClusterMerge(args[0], args[1:]); err != nil {
				return err
			}

			fmt.Fprintf(
				cmd.OutOrStdout(),
				"已添加 merge 规则: target=%s sources=%v\n",
				args[0],
				args[1:],
			)
			return nil
		},
	}
}

func newClusterSplitCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "split <source-cluster-id> <session-id...>",
		Short: "从一个 cluster 中拆出新的 cluster",
		Args:  cobra.MinimumNArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			application, err := app.New()
			if err != nil {
				return err
			}

			target, err := application.AddClusterSplit(args[0], args[1:])
			if err != nil {
				return err
			}

			fmt.Fprintf(
				cmd.OutOrStdout(),
				"已添加 split 规则: source=%s target=%s sessions=%v\n",
				args[0],
				target,
				args[1:],
			)
			return nil
		},
	}
}

func newClusterResetCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "reset <cluster-id>",
		Short: "清除某个 cluster 相关的人工规则，恢复为自动结果",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			application, err := app.New()
			if err != nil {
				return err
			}

			if err := application.ResetCluster(args[0]); err != nil {
				return err
			}

			fmt.Fprintf(cmd.OutOrStdout(), "已重置 cluster: %s\n", args[0])
			return nil
		},
	}
}
