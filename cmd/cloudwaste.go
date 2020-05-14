package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/timmyers/cloudwaste/cmd/scan"
)

func main() {
	var (
		rootCmd = &cobra.Command{
			Use:   "cloudwaste",
			Short: "Cloudwaste finds wasted resources in your cloud",
		}
	)

	rootCmd.AddCommand(scan.Cmd())
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}