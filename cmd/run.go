package cmd

import (
	"github.com/spf13/cobra"
	"github.com/timmyers/cloudwaste/pkg/aws"
)

func init() {
	rootCmd.AddCommand(runCmd)
}

var runCmd = &cobra.Command{
	Use:   "run",
	Short: "Run cloudwaste analysis",
	Run:   runCmdRun,
}

func runCmdRun(cmd *cobra.Command, args []string) {
	aws.AnalyzeWaste()
	// writer := uilive.New()
	// writer.Start()

	// for i := 0; i <= 100; i++ {
	// 	fmt.Fprintf(writer, "Downloading.. (%d/%d) GB\n", i, 100)
	// 	time.Sleep(time.Millisecond * 5)
	// }

	// fmt.Fprintln(writer, "Finished: Downloaded 100GB")
	// writer.Stop() // flush and stop rendering
}
