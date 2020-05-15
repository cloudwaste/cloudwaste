package aws

import (
	"context"
	"os"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/aws/aws-sdk-go/service/pricing"
	"github.com/spf13/viper"
	"go.uber.org/zap"

	ec2Waste "github.com/cloudwaste/cloudwaste/pkg/aws/ec2"
	"github.com/cloudwaste/cloudwaste/pkg/aws/util"
)

func AnalyzeWaste(log *zap.SugaredLogger) {
	sess := session.Must(session.NewSession())

	region := viper.GetString(util.FlagRegion)

	ec2Client := &ec2Waste.Client{
		Logger:  log,
		EC2:     ec2.New(sess, aws.NewConfig().WithRegion(region)),
		Pricing: pricing.New(sess, aws.NewConfig().WithRegion("us-east-1")),
	}

	var wastedResources []util.AWSWastedResource

	// Run all the checks
	wastedNATGateways, err := ec2Client.AnalyzeNATGatewayWaste(context.TODO(), region)
	if err == nil {
		wastedResources = append(wastedResources, wastedNATGateways...)
	} else {
		log.Errorf("failed to analyze NAT Gateways: %v", err)
	}
	wastedEBSVolumes, err := ec2Client.AnalyzeEBSVolumeWaste(context.TODO(), region)
	if err == nil {
		wastedResources = append(wastedResources, wastedEBSVolumes...)
	} else {
		log.Errorf("failed to analyze EBS Volumes: %v", err)
	}
	wastedElasticIPAddresses, err := ec2Client.AnalyzeElasticIPAddressWaste(context.TODO(), region)
	if err == nil {
		wastedResources = append(wastedResources, wastedElasticIPAddresses...)
	} else {
		log.Errorf("failed to analyze Elastic IP Addresses: %v", err)
		_, err := ec2Client.AnalyzeNATGatewayWaste(context.TODO(), region)
		if err != nil {
			os.Exit(-1)
		}
	}

	for _, r := range wastedResources {
		log.Infof("%s - %s: $%f/%s", r.Resource.R.Type(), r.Resource.R.ID(), r.Price.Rate, r.Price.Unit)
	}
}
