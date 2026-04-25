package cli

import (
	"fmt"

	"github.com/liurui/codex-session-manager/internal/app"
	"github.com/liurui/codex-session-manager/internal/ui"
	"github.com/spf13/cobra"
)

func newUICommand() *cobra.Command {
	var addr string
	var noOpen bool

	command := &cobra.Command{
		Use:     "dashboard",
		Aliases: []string{"ui"},
		Short:   "启动本地 Dashboard，并在需要时自动初始化和准备数据",
		RunE: func(cmd *cobra.Command, args []string) error {
			application, err := app.New()
			if err != nil {
				return err
			}
			if err := application.PrepareData(); err != nil {
				return err
			}

			server := ui.New(application)
			url, err := server.ListenAndServe(addr, !noOpen)
			if err != nil {
				return err
			}

			fmt.Fprintf(cmd.OutOrStdout(), "CSM UI 已启动: %s\n", url)
			select {}
		},
	}

	command.Flags().StringVar(&addr, "addr", "127.0.0.1:7788", "监听地址")
	command.Flags().BoolVar(&noOpen, "no-open", false, "不自动打开浏览器")
	return command
}
