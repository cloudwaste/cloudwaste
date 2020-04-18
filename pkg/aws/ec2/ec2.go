package ec2

import (
	"context"
	"errors"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/aws/aws-sdk-go/service/ec2/ec2iface"
	"github.com/aws/aws-sdk-go/service/pricing"

	wasteAWS "github.com/timmyers/cloudwaste/pkg/aws"
)

var (
	serviceCode             = "AmazonEC2"
	multiplePriceCodesError = errors.New("Couldn't find single price code")
	couldntParseError       = errors.New("Couldn't parse price code")
)

type EC2 struct {
	Client ec2iface.EC2API
}

func (client *EC2) GetUnusedElasticIPAddresses(ctx context.Context) ([]*ec2.Address, error) {
	resp, err := client.Client.DescribeAddressesWithContext(ctx, &ec2.DescribeAddressesInput{})

	if err != nil {
		return nil, err
	}

	var unusedAddresses []*ec2.Address
	for _, address := range resp.Addresses {
		if address.AssociationId == nil {
			unusedAddresses = append(unusedAddresses, address)
		}
	}

	return unusedAddresses, nil
}

func (client *EC2) GetUnusedNATGateways(ctx context.Context) ([]*ec2.NatGateway, error) {
	var unusedGateways []*ec2.NatGateway

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
					unusedGateways = append(unusedGateways, gateway)
				}
			}

			return true
		})

	if err != nil {
		return nil, err
	}

	return unusedGateways, nil
}

func GetUnusedElasticIPAddressPrice(session *session.Session) (*wasteAWS.Price, error) {
	regionName := wasteAWS.RegionLongNames[*session.Config.Region]

	client := pricing.New(session)
	resp, err := client.GetProductsWithContext(context.TODO(), &pricing.GetProductsInput{
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

				return &wasteAWS.Price{
					Unit: unit,
					Rate: usd,
				}, nil
			}
		}
	}

	return nil, nil
}
