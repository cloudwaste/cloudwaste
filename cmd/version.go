package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(versionCmd)
}

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print the version number of cloudwaste",
	Long:  `All software has versions. This is Cloudwaste's`,
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("Cloudwaste v0.0.1")
	},
}
