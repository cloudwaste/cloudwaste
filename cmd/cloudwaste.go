package main

import (
	"os"

	"github.com/cloudwaste/cloudwaste/cmd/scan"
	"github.com/spf13/cobra"
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
	defer func() {
		err := l.Sync()
		if err != nil {
			panic("failed to flush logs")
		}
	}()
	logger := l.Sugar()

	rootCmd.AddCommand(scan.Cmd(logger))
	if err := rootCmd.Execute(); err != nil {
		logger.Errorf("failed to execute root command", zap.Error(err))
		os.Exit(1)
	}
}
