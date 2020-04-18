package main

import (
	"context"
	"log"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/ec2"
	ec2Waste "github.com/timmyers/cloudwaste/pkg/aws/ec2"
)

func main() {
	sess := session.Must(session.NewSession())

	// unusedAddresses, err := ec2.GetUnusedElasticIPAddresses(sess)
	// if err != nil {
	// 	return
	// }
	// log.Printf("%d unused addresses", len(unusedAddresses))
	// _ = ec2.GetUnusedElasticIPAddressPrice(sess)

	client := &ec2Waste.EC2{
		Client: ec2.New(sess),
	}

	unusedGateways, err := client.GetUnusedNATGateways(context.TODO())
	if err != nil {
		return
	}
	log.Printf("%d unused gateways", len(unusedGateways))
}
