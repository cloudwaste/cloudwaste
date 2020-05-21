package pricing

import (
	"context"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/endpoints"
	"github.com/aws/aws-sdk-go/service/pricing"
	"github.com/aws/aws-sdk-go/service/pricing/pricingiface"

	"github.com/mitchellh/mapstructure"
)

type ServiceCode string

const (
	DynamoDB ServiceCode = "AmazonDynamoDB"
	EC2      ServiceCode = "AmazonEC2"
)

type PricingInterface interface {
	GetProducts(ctx context.Context, options *GetProductsInput) ([]*AWSPriceItem, error)
}

type Client struct {
	Pricing pricingiface.PricingAPI
}

type GetProductsInput struct {
	Region      string
	ServiceCode ServiceCode
	Filters     []*pricing.Filter
}

type AWSPriceItemProductAttributes struct {
	Location  string `json:"location"`
	UsageType string `json:"usagetype"`
}
type AWSPriceItemProduct struct {
	Attributes AWSPriceItemProductAttributes `json:"attributes"`
}
type AWSPriceItemPricePerUnit struct {
	USD string `json:"USD"`
}
type AWSPriceItemPriceDimension struct {
	Description  string                   `json:"description"`
	BeginRange   string                   `json:"beginRange"`
	EndRange     string                   `json:"endRange"`
	PricePerUnit AWSPriceItemPricePerUnit `json:"pricePerUnit"`
}
type AWSPriceItemOnDemand struct {
	PriceDimensions map[string]AWSPriceItemPriceDimension `json:"priceDimensions"`
	SKU             string                                `json:"sku"`
}
type AWSPriceItemTerms struct {
	Attributes AWSPriceItemProductAttributes   `json:"attributes"`
	OnDemand   map[string]AWSPriceItemOnDemand `json:"OnDemand,omitempty"`
}

type AWSPriceItem struct {
	Product AWSPriceItemProduct `json:"product"`
	Terms   AWSPriceItemTerms   `json:"terms"`
}

func (client *Client) GetProducts(ctx context.Context, options *GetProductsInput) ([]*AWSPriceItem, error) {
	regionName := endpoints.AwsPartition().Regions()[options.Region].Description()

	var priceItems []*AWSPriceItem
	var cbErr error = nil

	filters := append(options.Filters, &pricing.Filter{
		Type:  aws.String("TERM_MATCH"),
		Field: aws.String("location"),
		Value: aws.String(regionName),
	})

	err := client.Pricing.GetProductsPagesWithContext(ctx, &pricing.GetProductsInput{
		ServiceCode: aws.String(string(options.ServiceCode)),
		Filters:     filters,
	}, func(products *pricing.GetProductsOutput, lastPage bool) bool {
		for _, priceItemJson := range products.PriceList {
			var priceItem AWSPriceItem
			err := mapstructure.Decode(priceItemJson, &priceItem)
			if err != nil {
				cbErr = err
				return false
			}

			priceItems = append(priceItems, &priceItem)
		}

		return true
	})
	if err != nil {
		return nil, err
	}
	if cbErr != nil {
		return nil, cbErr
	}

	return priceItems, nil
}
