package ec2

import (
	"context"
	"errors"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/aws/aws-sdk-go/service/ec2/ec2iface"
	"github.com/aws/aws-sdk-go/service/pricing"
	"github.com/aws/aws-sdk-go/service/pricing/pricingiface"

	util "github.com/timmyers/cloudwaste/pkg/aws/util"
)

var (
	serviceCode             = "AmazonEC2"
	multiplePriceCodesError = errors.New("Couldn't find single price code")
	couldntParseError       = errors.New("Couldn't parse price code")
)

type EC2 struct {
	Client ec2iface.EC2API
}

type Pricing struct {
	Client pricingiface.PricingAPI
}

type ElasticIPAddress struct {
	r *ec2.Address
}

type NatGateway struct {
	r *ec2.NatGateway
}

type EBSVolume struct {
	r *ec2.Volume
}

func (a ElasticIPAddress) Type() string {
	return "Elastic IP Address"
}

func (a ElasticIPAddress) ID() string {
	return aws.StringValue(a.r.AllocationId)
}

func (r NatGateway) Type() string {
	return "Elastic IP Address"
}

func (r NatGateway) ID() string {
	return aws.StringValue(r.r.NatGatewayId)
}

func (r EBSVolume) Type() string {
	return "EBS Volume"
}

func (r EBSVolume) ID() string {
	return aws.StringValue(r.r.VolumeId)
}

func (client *EC2) GetUnusedElasticIPAddresses(ctx context.Context) ([]util.AWSResourceObject, error) {
	resp, err := client.Client.DescribeAddressesWithContext(ctx, &ec2.DescribeAddressesInput{})

	if err != nil {
		return nil, err
	}

	var unusedAddresses []util.AWSResourceObject
	for _, address := range resp.Addresses {
		if address.AssociationId == nil {
			unusedAddresses = append(unusedAddresses, util.AWSResourceObject{R: &ElasticIPAddress{address}})
		}
	}

	return unusedAddresses, nil
}

func (client *EC2) GetUnusedNATGateways(ctx context.Context) ([]util.AWSResourceObject, error) {
	var unusedGateways []util.AWSResourceObject

	err := client.Client.DescribeNatGatewaysPagesWithContext(ctx, &ec2.DescribeNatGatewaysInput{},
		func(page *ec2.DescribeNatGatewaysOutput, lastPage bool) bool {
			for _, gateway := range page.NatGateways {
				resp, _ := client.Client.DescribeRouteTablesWithContext(ctx, &ec2.DescribeRouteTablesInput{
					Filters: []*ec2.Filter{
						{
							Name:   aws.String("route.nat-gateway-id"),
							Values: []*string{gateway.NatGatewayId},
						},
					},
				})

				if len(resp.RouteTables) == 0 {
					unusedGateways = append(unusedGateways, util.AWSResourceObject{R: &NatGateway{gateway}})
				}
			}

			return true
		})

	if err != nil {
		return nil, err
	}

	return unusedGateways, nil
}

func (client *EC2) GetUnusedEBSVolumes(ctx context.Context) ([]util.AWSResourceObject, error) {
	var unusedVolumes []util.AWSResourceObject

	err := client.Client.DescribeVolumesPagesWithContext(ctx, &ec2.DescribeVolumesInput{},
		func(page *ec2.DescribeVolumesOutput, lastPage bool) bool {
			for _, volume := range page.Volumes {
				if *volume.State == "available" {
					unusedVolumes = append(unusedVolumes, util.AWSResourceObject{R: &EBSVolume{volume}})
				}
			}
			return true
		})

	if err != nil {
		return nil, err
	}

	return unusedVolumes, nil
}

func (client *Pricing) GetUnusedElasticIPAddressPrice(ctx context.Context, region string) (*util.Price, error) {
	regionName := util.RegionLongNames[region]

	resp, err := client.Client.GetProductsWithContext(ctx, &pricing.GetProductsInput{
		ServiceCode: aws.String(serviceCode),
		Filters: []*pricing.Filter{
			{
				Type:  aws.String("TERM_MATCH"),
				Field: aws.String("productFamily"),
				Value: aws.String("IP Address"),
			},
			{
				Type:  aws.String("TERM_MATCH"),
				Field: aws.String("group"),
				Value: aws.String("ElasticIP:Address"),
			},
			{
				Type:  aws.String("TERM_MATCH"),
				Field: aws.String("location"),
				Value: aws.String(regionName),
			},
		},
	})

	if err != nil {
		return nil, err
	}
	if len(resp.PriceList) != 1 {
		return nil, multiplePriceCodesError
	}

	priceItem := resp.PriceList[0]
	terms, ok := priceItem["terms"].(map[string]interface{})
	if !ok {
		return nil, couldntParseError
	}
	onDemand, ok := terms["OnDemand"].(map[string]interface{})
	if !ok {
		return nil, couldntParseError
	}

	for _, v := range onDemand {
		onDemandValue, ok := v.(map[string]interface{})
		if !ok {
			return nil, couldntParseError
		}

		priceDimensions, ok := onDemandValue["priceDimensions"].(map[string]interface{})
		if !ok {
			return nil, couldntParseError
		}

		for _, v2 := range priceDimensions {
			priceDimensionsValue, ok := v2.(map[string]interface{})
			if !ok {
				return nil, couldntParseError
			}

			unit, ok := priceDimensionsValue["unit"].(string)
			if !ok {
				return nil, couldntParseError
			}

			if priceDimensionsValue["beginRange"] == "1" {
				pricePerUnit, ok := priceDimensionsValue["pricePerUnit"].(map[string]interface{})
				if !ok {
					return nil, couldntParseError
				}

				usd, ok := pricePerUnit["USD"].(string)
				if !ok {
					return nil, couldntParseError
				}

				return &util.Price{
					Unit: unit,
					Rate: usd,
				}, nil
			}
		}
	}

	return nil, nil
}
