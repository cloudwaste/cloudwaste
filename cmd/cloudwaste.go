package main

import (
	"os"

	"github.com/cloudwaste/cloudwaste/cmd/scan"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"go.uber.org/zap"
)

func main() {
	var (
		rootCmd = &cobra.Command{
			Use:   "cloudwaste",
			Short: "Cloudwaste finds wasted resources in your cloud",
		}
	)

	l, err := zap.NewDevelopment()
	if err != nil {
		panic("failed to create logger")
	}
	logger := l.Sugar()

	viper.SetEnvPrefix("cloudwaste")
	viper.AutomaticEnv()

	rootCmd.AddCommand(scan.Cmd(logger))
	if err := rootCmd.Execute(); err != nil {
		logger.Errorf("failed to execute root command", zap.Error(err))
		os.Exit(1)
	}
}
