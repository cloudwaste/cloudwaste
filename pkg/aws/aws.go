package aws

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/aws/aws-sdk-go/service/pricing"

	ec2Waste "github.com/timmyers/cloudwaste/pkg/aws/ec2"
	"github.com/timmyers/cloudwaste/pkg/aws/util"
)

func AnalyzeWaste() {
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
		fmt.Println(err)
	}
	wastedEBSVolumes, err := ec2Client.AnalyzeEBSVolumeWaste(context.TODO(), region)
	if err == nil {
		wastedResources = append(wastedResources, wastedEBSVolumes...)
	} else {
		fmt.Println(err)
	}
	wastedElasticIPAddresses, err := ec2Client.AnalyzeElasticIPAddressWaste(context.TODO(), region)
	if err == nil {
		wastedResources = append(wastedResources, wastedElasticIPAddresses...)
	} else {
		fmt.Println(err)
	}

	for _, r := range wastedResources {
		fmt.Printf("%s - %s: $%f/%s\n", r.Resource.R.Type(), r.Resource.R.ID(), r.Price.Rate, r.Price.Unit)
	}
}
