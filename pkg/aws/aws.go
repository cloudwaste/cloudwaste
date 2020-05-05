package aws

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/ec2"

	ec2Waste "github.com/timmyers/cloudwaste/pkg/aws/ec2"
	"github.com/timmyers/cloudwaste/pkg/aws/util"
)

func AnalyzeWaste() {
	sess := session.Must(session.NewSession())

	client := &ec2Waste.EC2{
		Client: ec2.New(sess, aws.NewConfig().WithRegion("us-east-1")),
	}

	var unusedResources []util.AWSResourceObject

	// Run all the checks
	unusedAddresses, err := client.GetUnusedElasticIPAddresses(context.TODO())
	if err == nil {
		unusedResources = append(unusedResources, unusedAddresses...)
	}
	unusedGateways, err := client.GetUnusedNATGateways(context.TODO())
	if err == nil {
		unusedResources = append(unusedResources, unusedGateways...)
	}
	unusedVolumes, err := client.GetUnusedEBSVolumes(context.TODO())
	if err == nil {
		unusedResources = append(unusedResources, unusedVolumes...)
	}

	for _, r := range unusedResources {
		fmt.Printf("%s - %s\n", r.R.Type(), r.R.ID())
	}
}
