package ec2

import (
	"context"
	"errors"
	"testing"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/aws/aws-sdk-go/service/pricing"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/suite"
)

type EBSTestSuite struct {
	suite.Suite
	m      *mockedEC2
	p      *mockedPricing
	region string
	client Client
}

func (suite *EBSTestSuite) SetupTest() {
	suite.m = new(mockedEC2)
	suite.p = new(mockedPricing)
	suite.region = "us-east-1"
	suite.client = Client{EC2: suite.m, Pricing: suite.p}
}

func (suite *EBSTestSuite) TestAnalyzeEBSVolumeWaste() {
	assert := assert.New(suite.T())

	const vol1Name = "vol1"

	rate := "0.1000000000"
	expectedUnit := "GB-Mo"
	var unusedVolumeSize int64 = 500 // GB
	var expectedRate float64 = float64(.1) * float64(unusedVolumeSize)

	suite.p.On("GetProductsWithContext", mock.Anything, mock.Anything, mock.Anything).
		Return(&pricing.GetProductsOutput{
			PriceList: []aws.JSONValue{
				{
					"product": map[string]interface{}{
						"attributes": map[string]interface{}{
							"volumeApiName": "gp2",
						},
					},
					"terms": map[string]interface{}{
						"OnDemand": map[string]interface{}{
							"1": map[string]interface{}{
								"priceDimensions": map[string]interface{}{
									"1": map[string]interface{}{
										"unit": expectedUnit,
										"pricePerUnit": map[string]interface{}{
											"USD": rate,
										},
									},
								},
							},
						},
					},
				},
			},
		}, nil).Once()

	suite.m.On("DescribeVolumesPagesWithContext", mock.Anything, mock.Anything, mock.Anything).
		Return(&ec2.DescribeVolumesOutput{
			Volumes: []*ec2.Volume{
				{ // unused
					VolumeId:   aws.String(vol1Name),
					State:      aws.String("available"),
					Size:       aws.Int64(unusedVolumeSize),
					VolumeType: aws.String("gp2"),
				},
				{ // used
					VolumeId: aws.String("vol2"),
					State:    aws.String("in-use"),
				},
			},
		}, nil).Once()

	wastedVolumes, err := suite.client.AnalyzeEBSVolumeWaste(context.TODO(), suite.region)

	assert.Nil(err)
	assert.NotNil(wastedVolumes)
	assert.Equal(1, len(wastedVolumes))
	assert.Equal(vol1Name, wastedVolumes[0].Resource.R.ID())
	assert.Equal(expectedUnit, wastedVolumes[0].Price.Unit)
	assert.Equal(expectedRate, wastedVolumes[0].Price.Rate)

	// Test error cases
	suite.p.On("GetProductsWithContext", mock.Anything, mock.Anything, mock.Anything).
		Return(nil, errors.New("error")).Once()

	wastedVolumes, err = suite.client.AnalyzeEBSVolumeWaste(context.TODO(), suite.region)
	assert.Nil(wastedVolumes)
	assert.NotNil(err)
}

func (suite *EBSTestSuite) TestGetUnusedEBSVolumes() {
	assert := assert.New(suite.T())

	const vol1Name = "vol1"

	suite.m.On("DescribeVolumesPagesWithContext", mock.Anything, mock.Anything, mock.Anything).
		Return(&ec2.DescribeVolumesOutput{
			Volumes: []*ec2.Volume{
				{ // unused
					VolumeId: aws.String(vol1Name),
					State:    aws.String("available"),
				},
				{ // used
					VolumeId: aws.String("vol2"),
					State:    aws.String("in-use"),
				},
			},
		}, nil).Once()

	unusedVolumes, err := suite.client.GetUnusedEBSVolumes(context.Background())

	assert.Equal(1, len(unusedVolumes))
	assert.Equal(vol1Name, unusedVolumes[0].R.ID())
	assert.Equal(ebsVolumeType, unusedVolumes[0].R.Type())
	assert.Nil(err)

	// Test error cases
	suite.m.On("DescribeVolumesPagesWithContext", mock.Anything, mock.Anything, mock.Anything).
		Return(nil, errors.New("error")).Once()

	unusedVolumes, err = suite.client.GetUnusedEBSVolumes(context.Background())
	assert.Nil(unusedVolumes)
	assert.NotNil(err)
}

func (suite *EBSTestSuite) TestGetEBSVolumePricing() {
	assert := assert.New(suite.T())

	expectedRate := "0.0100000000"
	expectedUnit := "GB-Mo"

	suite.p.On("GetProductsWithContext", mock.Anything, mock.Anything, mock.Anything).
		Return(&pricing.GetProductsOutput{
			PriceList: []aws.JSONValue{
				{
					"product": map[string]interface{}{
						"attributes": map[string]interface{}{
							"volumeApiName": "gp2",
						},
					},
					"terms": map[string]interface{}{
						"OnDemand": map[string]interface{}{
							"1": map[string]interface{}{
								"priceDimensions": map[string]interface{}{
									"1": map[string]interface{}{
										"unit": expectedUnit,
										"pricePerUnit": map[string]interface{}{
											"USD": expectedRate,
										},
									},
								},
							},
						},
					},
				},
			},
		}, nil).Once()

	pricingRet, err := suite.client.GetEBSVolumePricing(context.Background(), suite.region)

	gp2Pricing := pricingRet[EBSVolumeType("gp2")]
	assert.NotNil(gp2Pricing)
	assert.Equal(expectedRate, gp2Pricing.OnDemand.Rate)
	assert.Equal(expectedUnit, gp2Pricing.OnDemand.Unit)
	assert.Nil(err)

	// Test error cases
	suite.p.On("GetProductsWithContext", mock.Anything, mock.Anything, mock.Anything).
		Return(nil, errors.New("error")).Once()

	pricingRet, err = suite.client.GetEBSVolumePricing(context.Background(), suite.region)
	assert.Nil(pricingRet)
	assert.NotNil(err)

	suite.p.On("GetProductsWithContext", mock.Anything, mock.Anything, mock.Anything).
		Return(&pricing.GetProductsOutput{
			PriceList: []aws.JSONValue{
				{},
			},
		}, nil).Once()
	pricingRet, err = suite.client.GetEBSVolumePricing(context.Background(), suite.region)
	assert.Nil(pricingRet)
	assert.NotNil(err)
}

// In order for 'go test' to run this suite, we need to create
// a normal test function and pass our suite to suite.Run
func TestEBSTestSuite(t *testing.T) {
	suite.Run(t, new(EBSTestSuite))
}
