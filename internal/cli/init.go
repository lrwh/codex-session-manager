package cli

import (
	"fmt"

	"github.com/liurui/codex-session-manager/internal/app"
	"github.com/spf13/cobra"
)

func newInitCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "init",
		Short: "初始化 CSM 本地工作目录",
		RunE: func(cmd *cobra.Command, args []string) error {
			application, err := app.New()
			if err != nil {
				return err
			}
			if err := application.Init(); err != nil {
				return err
			}

			fmt.Fprintf(cmd.OutOrStdout(), "CSM 已初始化: %s\n", application.Paths.HomeDir)
			return nil
		},
	}
}
