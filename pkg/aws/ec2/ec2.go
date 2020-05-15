package ec2

import (
	"context"
	"errors"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/aws/aws-sdk-go/service/ec2/ec2iface"
	"github.com/aws/aws-sdk-go/service/pricing"
	"github.com/aws/aws-sdk-go/service/pricing/pricingiface"
	"go.uber.org/zap"

	util "github.com/cloudwaste/cloudwaste/pkg/aws/util"
)

var (
	serviceCode             = "AmazonEC2"
	multiplePriceCodesError = errors.New("Couldn't find single price code")
	couldntParseError       = errors.New("Couldn't parse price code")
)

const (
	UsageTypeNatGatewayHours = "NatGateway-Hours"
)

type Client struct {
	Logger  *zap.SugaredLogger
	EC2     ec2iface.EC2API
	Pricing pricingiface.PricingAPI
}

type ElasticIPAddress struct {
	r *ec2.Address
}

type NatGateway struct {
	r *ec2.NatGateway
}

type ElasticIPAddressPricing util.Price

type NATGatewayPricing struct {
	PerHour *util.Price
}

func (a ElasticIPAddress) Type() string {
	return "Elastic IP Address"
}

func (a ElasticIPAddress) ID() string {
	return aws.StringValue(a.r.AllocationId)
}

func (r NatGateway) Type() string {
	return "NAT Gateway"
}

func (r NatGateway) ID() string {
	return aws.StringValue(r.r.NatGatewayId)
}

func (client *Client) AnalyzeElasticIPAddressWaste(ctx context.Context, region string) ([]util.AWSWastedResource, error) {
	pricing, err := client.GetElasticIPAddressPricing(ctx, region)
	if err != nil {
		return nil, err
	}

	unusedIPAddress, err := client.GetUnusedElasticIPAddresses(ctx)
	if err != nil {
		return nil, err
	}

	var wastedResources []util.AWSWastedResource

	for _, unusedAddress := range unusedIPAddress {
		if pricing.Unit != "Hrs" {
			return nil, errors.New("Unhandled pricing unit")
		}

		wastedResources = append(wastedResources, util.AWSWastedResource{
			Resource: unusedAddress,
			Price: util.Price{
				Unit: "Hr",
				Rate: pricing.Rate,
			},
		})
	}

	return wastedResources, nil
}

func (client *Client) AnalyzeNATGatewayWaste(ctx context.Context, region string) ([]util.AWSWastedResource, error) {
	pricing, err := client.GetNATGatewayPricing(ctx, region)
	if err == util.NoResourceFoundError {
		return []util.AWSWastedResource{}, nil
	}

	if err != nil {
		client.Logger.Errorf("couldn't get NAT Gateway Pricing: %v\n", err)
		return nil, err
	}

	unusedNatGateways, err := client.GetUnusedNATGateways(ctx)
	if err != nil {
		client.Logger.Errorf("couldn't get Unused NAT Gateways: %v\n", err)
		return nil, err
	}

	var wastedResources []util.AWSWastedResource

	for _, unusedResource := range unusedNatGateways {
		wastedResources = append(wastedResources, util.AWSWastedResource{
			Resource: unusedResource,
			Price: util.Price{
				Unit: "Hr",
				Rate: pricing.PerHour.Rate,
			},
		})
	}

	return wastedResources, nil
}

func (client *Client) GetUnusedElasticIPAddresses(ctx context.Context) ([]util.AWSResourceObject, error) {
	resp, err := client.EC2.DescribeAddressesWithContext(ctx, &ec2.DescribeAddressesInput{})

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

func (client *Client) GetUnusedNATGateways(ctx context.Context) ([]util.AWSResourceObject, error) {
	var unusedGateways []util.AWSResourceObject

	err := client.EC2.DescribeNatGatewaysPagesWithContext(ctx, &ec2.DescribeNatGatewaysInput{},
		func(page *ec2.DescribeNatGatewaysOutput, lastPage bool) bool {
			for _, gateway := range page.NatGateways {
				if *gateway.State == "deleted" {
					continue
				}

				resp, _ := client.EC2.DescribeRouteTablesWithContext(ctx, &ec2.DescribeRouteTablesInput{
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

func (client *Client) GetElasticIPAddressPricing(ctx context.Context, region string) (*util.Price, error) {
	regionName := util.RegionLongNames[region]

	resp, err := client.Pricing.GetProductsWithContext(ctx, &pricing.GetProductsInput{
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

	priceItem, err := util.ParsePriceItem(resp.PriceList[0])
	if err != nil {
		return nil, err
	}

	for _, dimension := range priceItem.OnDemand.Dimensions {
		if dimension.BeginRange == "1" {
			return &util.Price{
				Rate: dimension.Rate,
				Unit: dimension.Unit,
			}, nil
		}
	}

	return nil, couldntParseError
}

func (client *Client) GetNATGatewayPricing(ctx context.Context, region string) (*NATGatewayPricing, error) {
	regionName := util.RegionLongNames[region]

	resp, err := client.Pricing.GetProductsWithContext(ctx, &pricing.GetProductsInput{
		ServiceCode: aws.String(serviceCode),
		Filters: []*pricing.Filter{
			{
				Type:  aws.String("TERM_MATCH"),
				Field: aws.String("productFamily"),
				Value: aws.String("NAT Gateway"),
			},
			{
				Type:  aws.String("TERM_MATCH"),
				Field: aws.String("location"),
				Value: aws.String(regionName),
			},
		},
	})

	if err != nil {
		client.Logger.Errorf("couldn't GetProductsWithContext: %v", err)
		return nil, err
	}

	for _, p := range resp.PriceList {
		priceItem, err := util.ParsePriceItem(p)
		if err != nil {
			client.Logger.Errorf("couldn't ParsePriceItem: %v", err)
			return nil, err
		}

		if priceItem.UsageType == UsageTypeNatGatewayHours {
			if len(priceItem.OnDemand.Dimensions) != 1 {
				client.Logger.Error("priceItem.OnDemand.Dimensions was not 1")
				return nil, util.PricingError
			}
			dimension := priceItem.OnDemand.Dimensions[0]

			if dimension.Unit != "Hrs" {
				client.Logger.Error("dimension.Unit was not Hrs")
				return nil, util.PricingError
			}

			return &NATGatewayPricing{
				PerHour: &util.Price{
					Unit: "Hrs",
					Rate: dimension.Rate,
				},
			}, nil
		}
	}

	return nil, util.NoResourceFoundError
}
