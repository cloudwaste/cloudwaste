package pricing

import (
	"context"
	"errors"
	"testing"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/request"
	"github.com/aws/aws-sdk-go/service/pricing"
	"github.com/aws/aws-sdk-go/service/pricing/pricingiface"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/suite"
)

type mockedPricing struct {
	mock.Mock
	pricingiface.PricingAPI
	aws.Config
}

type PricingTestSuite struct {
	suite.Suite
	mockedPricing *mockedPricing
	region        string
	client        Client
}

func (m *mockedPricing) GetProductsPagesWithContext(ctx context.Context, input *pricing.GetProductsInput, handlePage func(*pricing.GetProductsOutput, bool) bool, options ...request.Option) error {
	args := m.Called(ctx, input, handlePage, options)

	argI := 0

	for {
		err, isErr := args.Get(argI).(error)
		if isErr {
			return err
		}
		nilVal := args.Get(argI)
		if nilVal == nil {
			return nil
		}

		_, lastPage := args.Get(argI + 1).(error)
		products, _ := args.Get(argI).(*pricing.GetProductsOutput)

		nextPage := handlePage(products, lastPage)

		if !nextPage {
			return nil
		}

		argI = argI + 1
	}
}

func (suite *PricingTestSuite) SetupTest() {
	suite.mockedPricing = new(mockedPricing)
	suite.region = "us-east-1"
	suite.client = Client{Pricing: suite.mockedPricing}
}

func (suite *PricingTestSuite) TestGetProducts() {
	assert := assert.New(suite.T())

	suite.mockedPricing.On("GetProductsPagesWithContext", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).
		Return(&pricing.GetProductsOutput{
			PriceList: []aws.JSONValue{
				{
					"product": aws.JSONValue{
						"attributes": aws.JSONValue{
							"location":  suite.region,
							"usagetype": "usagetype1",
						},
					},
					"terms": aws.JSONValue{
						"OnDemand": aws.JSONValue{
							"1": aws.JSONValue{
								"priceDimensions": aws.JSONValue{
									"1.1": aws.JSONValue{
										"description": "description1",
										"beginRange":  "beginRange1",
										"endRange":    "endRange1",
										"pricePerUnit": aws.JSONValue{
											"USD": "1",
										},
									},
								},
								"SKU": "sku1",
							},
						},
					},
				},
			},
		}, nil).Once()

	priceItems, err := suite.client.GetProducts(context.Background(), &GetProductsInput{
		Region:      suite.region,
		ServiceCode: EC2,
	})

	if assert.Nil(err) {
		assert.NotNil(priceItems)
	}

	suite.mockedPricing.On("GetProductsPagesWithContext", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).
		Return(&pricing.GetProductsOutput{
			PriceList: []aws.JSONValue{},
		}, errors.New("error")).Once()

	priceItems, err = suite.client.GetProducts(context.Background(), &GetProductsInput{
		Region:      suite.region,
		ServiceCode: EC2,
	})

	assert.NotNil(err)
	assert.Nil(priceItems)
}

func TestPricingTestSuite(t *testing.T) {
	suite.Run(t, new(PricingTestSuite))
}
