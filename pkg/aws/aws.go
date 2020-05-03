package aws

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/ec2"

	ec2Waste "github.com/timmyers/cloudwaste/pkg/aws/ec2"
)

type AWSResource interface {
	Type() string
	ID() string
}

func AnalyzeWaste() {
	sess := session.Must(session.NewSession())

	client := &ec2Waste.EC2{
		Client: ec2.New(sess, aws.NewConfig().WithRegion("us-east-1")),
	}

	var unusedResources []AWSResource
	unusedAddresses, err := client.GetUnusedElasticIPAddresses(context.TODO())
	if err == nil {
		for _, address := range unusedAddresses {
			unusedResources = append(unusedResources, address)
		}
	}

	unusedGateways, err := client.GetUnusedNATGateways(context.TODO())
	if err != nil {
		for _, gateway := range unusedGateways {
			unusedResources = append(unusedResources, gateway)
		}
	}

	for _, r := range unusedResources {
		fmt.Printf("%s - %s\n", r.Type(), r.ID())
	}
}
