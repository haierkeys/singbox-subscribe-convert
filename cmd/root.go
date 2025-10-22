package cmd

import (
	"fmt"
	"os"

	"github.com/haierkeys/singbox-subscribe-convert/global"
	"github.com/spf13/cobra"
)

var (
	rootCmd = &cobra.Command{
		Use:   "singbox-subscribe-convert",
		Short: "Singbox Subscribe Convert - A configuration server for sing-box",
		Long: `Singbox Subscribe Convert is a configuration management server that fetches
remote templates and node data, then serves generated configurations for sing-box.`,
		Version: global.Version,
	}
	configDefault string
)

// Execute 执行根命令
func Execute(c string) {
	configDefault = c
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
