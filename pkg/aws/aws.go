package aws

import (
	"context"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/aws/aws-sdk-go/service/pricing"
	"go.uber.org/zap"

	ec2Waste "github.com/cloudwaste/cloudwaste/pkg/aws/ec2"
	"github.com/cloudwaste/cloudwaste/pkg/aws/util"
)

func AnalyzeWaste(log *zap.SugaredLogger) {
	sess := session.Must(session.NewSession())

	region := "us-east-1"

	ec2Client := &ec2Waste.Client{
		EC2:     ec2.New(sess, aws.NewConfig().WithRegion(region)),
		Pricing: pricing.New(sess, aws.NewConfig().WithRegion(region)),
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
	}

	for _, r := range wastedResources {
		log.Infof("%s - %s: $%f/%s\n", r.Resource.R.Type(), r.Resource.R.ID(), r.Price.Rate, r.Price.Unit)
	}
}
