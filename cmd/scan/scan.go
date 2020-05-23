package scan

import (
	"github.com/cloudwaste/cloudwaste/pkg/aws"
	"github.com/cloudwaste/cloudwaste/pkg/aws/util"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"go.uber.org/zap"
)

// Cmd runs the scan command
func Cmd(log *zap.SugaredLogger) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "scan",
		Short: "Scan your cloud accounts for unused resources",
		Run: func(_ *cobra.Command, _ []string) {
			main(log)
		},
	}
	cmd.PersistentFlags().String(util.FlagRegion, "", "The AWS region you wish to scan. AWS_REGION env var and AWS shared config file are also supported.")
	err := viper.BindPFlag(util.FlagRegion, cmd.PersistentFlags().Lookup(util.FlagRegion))
	if err != nil {
		log.Fatalf("couldn't bind PFlag", err)
	}

	return cmd
}

func main(log *zap.SugaredLogger) {
	aws.AnalyzeWaste(log)
}
