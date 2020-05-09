package ec2

import (
	"context"
	"errors"
	"testing"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/request"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/aws/aws-sdk-go/service/ec2/ec2iface"
	"github.com/aws/aws-sdk-go/service/pricing"
	"github.com/aws/aws-sdk-go/service/pricing/pricingiface"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

type mockedEC2 struct {
	mock.Mock
	ec2iface.EC2API
}

type mockedPricing struct {
	mock.Mock
	pricingiface.PricingAPI
	aws.Config
}

func (m *mockedEC2) DescribeAddressesWithContext(ctx context.Context, input *ec2.DescribeAddressesInput, options ...request.Option) (*ec2.DescribeAddressesOutput, error) {
	args := m.Called(ctx, input, options)

	return args.Get(0).(*ec2.DescribeAddressesOutput), args.Error(1)
}

func (m *mockedEC2) DescribeNatGatewaysPagesWithContext(ctx context.Context, input *ec2.DescribeNatGatewaysInput, fn func(*ec2.DescribeNatGatewaysOutput, bool) bool, opts ...request.Option) error {
	args := m.Called(ctx, input, fn)

	fn(args.Get(0).(*ec2.DescribeNatGatewaysOutput), true)
	return args.Error(1)
}

func (m *mockedEC2) DescribeRouteTablesWithContext(ctx context.Context, input *ec2.DescribeRouteTablesInput, options ...request.Option) (*ec2.DescribeRouteTablesOutput, error) {
	args := m.Called(ctx, input, options)

	return args.Get(0).(*ec2.DescribeRouteTablesOutput), args.Error(1)
}

func (m *mockedEC2) DescribeVolumesPagesWithContext(ctx context.Context, input *ec2.DescribeVolumesInput, fn func(*ec2.DescribeVolumesOutput, bool) bool, opts ...request.Option) error {
	args := m.Called(ctx, input, fn)

	if args.Error(1) == nil {
		fn(args.Get(0).(*ec2.DescribeVolumesOutput), true)
	}
	return args.Error(1)
}

func (m *mockedPricing) GetProductsWithContext(ctx context.Context, input *pricing.GetProductsInput, options ...request.Option) (*pricing.GetProductsOutput, error) {
	args := m.Called(ctx, input, options)

	if args.Error(1) == nil {
		return args.Get(0).(*pricing.GetProductsOutput), nil
	}
	return nil, args.Error(1)
}

func TestGetUnusedElasticIPAddresses(t *testing.T) {
	assert := assert.New(t)

	const alloc2 = "allocation2"

	m := new(mockedEC2)
	m.On("DescribeAddressesWithContext", mock.Anything, mock.Anything, mock.Anything).
		Return(&ec2.DescribeAddressesOutput{
			Addresses: []*ec2.Address{
				{ // used
					AllocationId:  aws.String("allocation1"),
					AssociationId: aws.String("association1"),
				},
				{ // unused
					AllocationId:  aws.String("allocation2"),
					AssociationId: nil,
				},
			},
		}, nil).Once()

	client := Client{EC2: m}
	unusedAddresses, err := client.GetUnusedElasticIPAddresses(context.Background())
	assert.Equal(1, len(unusedAddresses))
	assert.Equal(alloc2, unusedAddresses[0].R.ID())
	assert.Nil(err)

	m.On("DescribeAddressesWithContext", mock.Anything, mock.Anything, mock.Anything).
		Return(&ec2.DescribeAddressesOutput{}, errors.New("AWS Error"))

	unusedAddresses, err = client.GetUnusedElasticIPAddresses(context.Background())
	assert.Nil(unusedAddresses)
	assert.NotNil(err)
}

func TestGetUnusedNATGateways(t *testing.T) {
	assert := assert.New(t)
	m := new(mockedEC2)

	// Unused
	m.On("DescribeNatGatewaysPagesWithContext", mock.Anything, mock.Anything, mock.Anything).
		Return(&ec2.DescribeNatGatewaysOutput{
			NatGateways: []*ec2.NatGateway{
				{
					NatGatewayId: aws.String("gateway1"),
				},
			},
		}, nil).Once()
	m.On("DescribeRouteTablesWithContext", mock.Anything, mock.Anything, mock.Anything).
		Return(&ec2.DescribeRouteTablesOutput{
			RouteTables: []*ec2.RouteTable{},
		}, nil).Once()

	client := Client{EC2: m}
	unusedNatGateways, err := client.GetUnusedNATGateways(context.Background())
	assert.Equal(1, len(unusedNatGateways))
	assert.Nil(err)

	// Used
	m.On("DescribeNatGatewaysPagesWithContext", mock.Anything, mock.Anything, mock.Anything).
		Return(&ec2.DescribeNatGatewaysOutput{
			NatGateways: []*ec2.NatGateway{
				{
					NatGatewayId: aws.String("gateway1"),
				},
			},
		}, nil).Once()
	m.On("DescribeRouteTablesWithContext", mock.Anything, mock.Anything, mock.Anything).
		Return(&ec2.DescribeRouteTablesOutput{
			RouteTables: []*ec2.RouteTable{
				{
					RouteTableId: aws.String("routetable1"),
				},
			},
		}, nil).Once()

	unusedNatGateways, err = client.GetUnusedNATGateways(context.Background())
	assert.Equal(0, len(unusedNatGateways))
	assert.Nil(err)
}

func TestGetUnusedElasticIPAddressPrice(t *testing.T) {
	assert := assert.New(t)
	m := new(mockedPricing)

	m.On("GetProductsWithContext", mock.Anything, mock.Anything, mock.Anything).
		Return(&pricing.GetProductsOutput{
			PriceList: []aws.JSONValue{
				{
					"terms": map[string]interface{}{
						"OnDemand": map[string]interface{}{
							"1": map[string]interface{}{
								"priceDimensions": map[string]interface{}{
									"1": map[string]interface{}{
										"unit":       "Hrs",
										"beginRange": "1",
										"pricePerUnit": map[string]interface{}{
											"USD": "0.0050000000",
										},
									},
								},
							},
						},
					},
				},
			},
		}, nil).Once()

	client := Client{Pricing: m}
	pricing, err := client.GetUnusedElasticIPAddressPrice(context.Background(), "us-east-1")

	assert.Equal(float64(.005), pricing.Rate)
	assert.Equal("Hrs", pricing.Unit)
	assert.Nil(err)
}
