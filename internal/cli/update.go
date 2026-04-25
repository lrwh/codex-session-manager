package cli

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/liurui/codex-session-manager/internal/app"
)

func newUpdateCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "update",
		Short: "更新 CSM 到 GitHub Releases 的最新版本",
		RunE: func(cmd *cobra.Command, args []string) error {
			application, err := app.New()
			if err != nil {
				return err
			}

			result, err := application.Update(Version)
			if err != nil {
				return err
			}

			if !result.Updated {
				fmt.Fprintf(cmd.OutOrStdout(), "当前已经是最新版本: %s\n", result.CurrentVersion)
				return nil
			}

			fmt.Fprintf(cmd.OutOrStdout(), "更新完成: %s -> %s (%s)\n", result.CurrentVersion, result.LatestVersion, result.AssetName)
			return nil
		},
	}
}
