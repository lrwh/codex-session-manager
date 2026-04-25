package cli

import (
	"fmt"

	"github.com/liurui/codex-session-manager/internal/app"
	"github.com/spf13/cobra"
)

func newScanCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "scan",
		Short: "扫描 Codex session 并生成轻量索引",
		RunE: func(cmd *cobra.Command, args []string) error {
			application, err := app.New()
			if err != nil {
				return err
			}

			result, err := application.Scan()
			if err != nil {
				return err
			}

			fmt.Fprintf(
				cmd.OutOrStdout(),
				"扫描完成: sources=%d sessions=%d output=%s\n",
				result.SourceCount,
				result.SessionCount,
				result.OutputFile,
			)
			return nil
		},
	}
}
