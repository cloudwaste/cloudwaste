package aws

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/cloudwatch"
	"github.com/aws/aws-sdk-go/service/dynamodb"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/aws/aws-sdk-go/service/pricing"

	dynamoWaste "github.com/timmyers/cloudwaste/pkg/aws/dynamodb"
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

	dynamoClient := &dynamoWaste.Client{
		DynamoDB:   dynamodb.New(sess, aws.NewConfig().WithRegion(region)),
		Cloudwatch: cloudwatch.New(sess, aws.NewConfig().WithRegion(region)),
	}

	var unusedResources []util.AWSResourceObject

	// Run all the checks
	unusedAddresses, err := ec2Client.GetUnusedElasticIPAddresses(context.TODO())
	if err == nil {
		unusedResources = append(unusedResources, unusedAddresses...)
	}
	unusedGateways, err := ec2Client.GetUnusedNATGateways(context.TODO())
	if err == nil {
		unusedResources = append(unusedResources, unusedGateways...)
	}
	unusedVolumes, err := ec2Client.GetUnusedEBSVolumes(context.TODO())
	if err == nil {
		unusedResources = append(unusedResources, unusedVolumes...)
	}
	unusedDynamoDBTables, err := dynamoClient.GetUnusedDynamoDBTables((context.TODO()))
	if err == nil {
		unusedResources = append(unusedResources, unusedDynamoDBTables...)
	}

	for _, r := range unusedResources {
		fmt.Printf("%s - %s\n", r.R.Type(), r.R.ID())
	}
}
