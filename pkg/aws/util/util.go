package util

import (
	"errors"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/endpoints"
)

type AWSResource interface {
	Type() string
	ID() string
}

type AWSResourceObject struct {
	R AWSResource
}

type Price struct {
	Unit string
	Rate float64
}

type AWSWastedResource struct {
	Resource AWSResourceObject
	Price    Price
}

type AWSPriceItem struct {
	Attributes map[string]interface{}
	OnDemand   struct {
		Unit string
		Rate string
	}
}

var RegionLongNames = map[string]string{
	endpoints.UsEast1RegionID: "US East (N. Virginia)",
	endpoints.UsEast2RegionID: "US East (Ohio)",
}

func ParsePriceItem(priceItem aws.JSONValue) (priceItemRet *AWSPriceItem, err error) {
	priceItemRet = &AWSPriceItem{}

	if product, ok := priceItem["product"].(map[string]interface{}); ok {
		if attributes, ok := product["attributes"].(map[string]interface{}); ok {
			priceItemRet.Attributes = attributes
		} else {
			return nil, errors.New("Could not parse attributes")
		}
	}

	if terms, ok := priceItem["terms"].(map[string]interface{}); ok {
		if onDemand, ok := terms["OnDemand"].(map[string]interface{}); ok {
			for _, v := range onDemand {
				if onDemandValue, ok := v.(map[string]interface{}); ok {
					if priceDimensions, ok := onDemandValue["priceDimensions"].(map[string]interface{}); ok {
						for _, v2 := range priceDimensions {
							if priceDimensionsValue, ok := v2.(map[string]interface{}); ok {
								pricePerUnit, ok1 := priceDimensionsValue["pricePerUnit"].(map[string]interface{})
								unit, ok2 := priceDimensionsValue["unit"].(string)

								if ok1 && ok2 {
									if usd, ok := pricePerUnit["USD"].(string); ok {
										priceItemRet.OnDemand.Rate = usd
										priceItemRet.OnDemand.Unit = unit
										return
									}
								}
							}
						}
					}
				}
			}
		}
	}

	return nil, errors.New("Could not parse OnDemand")
}
