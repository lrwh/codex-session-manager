package cli

import (
	"github.com/spf13/cobra"
)

var Version = "dev"
var rootSessionLimit int
var rootSessionJSON bool
var rootSessionVerbose bool

var rootCmd = &cobra.Command{
	Use:          "csm",
	Short:        "CSM 是一个轻量的 Codex session 管理工具",
	SilenceUsage: true,
	Version:      Version,
	RunE: func(cmd *cobra.Command, args []string) error {
		return runSessions(cmd, rootSessionLimit, rootSessionJSON, rootSessionVerbose)
	},
}

func Execute() error {
	return rootCmd.Execute()
}

func SetVersion(version string) {
	Version = version
	rootCmd.Version = version
}

func init() {
	rootCmd.Flags().IntVarP(&rootSessionLimit, "limit", "n", 0, "限制返回数量，默认 0 表示全部")
	rootCmd.Flags().BoolVar(&rootSessionJSON, "json", false, "以 JSON 输出")
	rootCmd.Flags().BoolVar(&rootSessionVerbose, "verbose", false, "输出详细信息")
	rootCmd.AddCommand(newClusterCommand())
	rootCmd.AddCommand(newFindCommand())
	rootCmd.AddCommand(newInitCommand())
	rootCmd.AddCommand(newScanCommand())
	rootCmd.AddCommand(newSessionsCommand())
	rootCmd.AddCommand(newShowCommand())
	rootCmd.AddCommand(newSourceCommand())
	rootCmd.AddCommand(newTagCommand())
	rootCmd.AddCommand(newUICommand())
}
