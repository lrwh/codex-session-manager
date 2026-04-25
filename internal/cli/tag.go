package cli

import (
	"fmt"
	"strings"

	"github.com/liurui/codex-session-manager/internal/app"
	"github.com/spf13/cobra"
)

func newTagCommand() *cobra.Command {
	command := &cobra.Command{
		Use:   "tag",
		Short: "管理 cluster 标签和名称",
	}

	command.AddCommand(newTagSetCommand())
	command.AddCommand(newTagRemoveCommand())
	return command
}

func newTagSetCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "set <cluster-id> <name>",
		Short: "给 cluster 设置人工名称",
		Args:  cobra.MinimumNArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			application, err := app.New()
			if err != nil {
				return err
			}

			name := strings.Join(args[1:], " ")
			if err := application.SetClusterName(args[0], name); err != nil {
				return err
			}

			fmt.Fprintf(cmd.OutOrStdout(), "已设置 cluster 名称: %s -> %s\n", args[0], name)
			return nil
		},
	}
}

func newTagRemoveCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "remove <cluster-id>",
		Short: "删除 cluster 的人工名称",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			application, err := app.New()
			if err != nil {
				return err
			}

			if err := application.RemoveClusterName(args[0]); err != nil {
				return err
			}

			fmt.Fprintf(cmd.OutOrStdout(), "已删除 cluster 名称: %s\n", args[0])
			return nil
		},
	}
}
