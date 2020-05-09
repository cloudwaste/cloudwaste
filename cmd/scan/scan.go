package scan

import (
	"github.com/spf13/cobra"
	"github.com/timmyers/cloudwaste/pkg/aws"
)

// Cmd runs the scan command
func Cmd() *cobra.Command {
	return &cobra.Command{
		Use:   "scan",
		Short: "Scan your cloud accounts for unused resources",
		Run: func(_ *cobra.Command, _ []string) {
			main()
		},
	}
}

func main() {
	aws.AnalyzeWaste()
}
