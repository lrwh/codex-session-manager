package cli

import (
	"fmt"

	"github.com/liurui/codex-session-manager/internal/app"
	"github.com/spf13/cobra"
)

func newSourceCommand() *cobra.Command {
	command := &cobra.Command{
		Use:   "source",
		Short: "管理 CSM 数据源",
	}

	command.AddCommand(newSourceAddCommand())
	command.AddCommand(newSourceListCommand())
	return command
}

func newSourceAddCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "add <path>",
		Short: "添加一个 Codex 数据目录",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			application, err := app.New()
			if err != nil {
				return err
			}

			source, err := application.AddSource(args[0])
			if err != nil {
				return err
			}

			fmt.Fprintf(
				cmd.OutOrStdout(),
				"已添加数据源: id=%s path=%s enabled=%t\n",
				source.ID,
				source.Path,
				source.Enabled,
			)
			return nil
		},
	}
}

func newSourceListCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "列出当前已配置的数据源",
		RunE: func(cmd *cobra.Command, args []string) error {
			application, err := app.New()
			if err != nil {
				return err
			}

			sources, err := application.LoadSources()
			if err != nil {
				return err
			}

			if len(sources) == 0 {
				fmt.Fprintln(cmd.OutOrStdout(), "当前没有已配置的数据源")
				return nil
			}

			fmt.Fprintln(cmd.OutOrStdout(), "ID\tENABLED\tPATH")
			for _, source := range sources {
				fmt.Fprintf(
					cmd.OutOrStdout(),
					"%s\t%t\t%s\n",
					source.ID,
					source.Enabled,
					source.Path,
				)
			}
			return nil
		},
	}
}
