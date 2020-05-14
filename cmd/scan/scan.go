package scan

import (
	"github.com/cloudwaste/cloudwaste/pkg/aws"
	"github.com/spf13/cobra"
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
