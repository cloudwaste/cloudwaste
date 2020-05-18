package scan

import (
	"github.com/cloudwaste/cloudwaste/pkg/aws"
	"github.com/spf13/cobra"
	"go.uber.org/zap"
)

// Cmd runs the scan command
func Cmd(log *zap.SugaredLogger) *cobra.Command {
	return &cobra.Command{
		Use:   "scan",
		Short: "Scan your cloud accounts for unused resources",
		Run: func(_ *cobra.Command, _ []string) {
			main(log)
		},
	}
}

func main(log *zap.SugaredLogger) {
	aws.AnalyzeWaste(log)
}
