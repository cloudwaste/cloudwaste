package main

import "github.com/timmyers/cloudwaste/cmd"

func main() {
	cmd.Execute()
}

// func main() {
// 	sess := session.Must(session.NewSession())

// 	// unusedAddresses, err := ec2.GetUnusedElasticIPAddresses(sess)
// 	// if err != nil {
// 	// 	return
// 	// }
// 	// log.Printf("%d unused addresses", len(unusedAddresses))
// 	// _ = ec2.GetUnusedElasticIPAddressPrice(sess)

// 	client := &ec2Waste.EC2{
// 		Client: ec2.New(sess),
// 	}

// 	unusedGateways, err := client.GetUnusedNATGateways(context.TODO())
// 	if err != nil {
// 		return
// 	}
// 	log.Printf("%d unused gateways", len(unusedGateways))
// }
