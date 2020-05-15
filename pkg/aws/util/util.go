package util

import (
	"strconv"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/endpoints"
	"github.com/pkg/errors"
)

const (
	// FlagRegion is a viper flag for the region to run in
	FlagRegion = "region"
)

var (
	PricingError         = errors.New("Pricing error")
	NoResourceFoundError = errors.New("There are no instances of the requested resource")
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

type AWSPriceItemDimension struct {
	BeginRange  string
	EndRange    string
	Unit        string
	Rate        float64
	Description string
}

type AWSPriceItem struct {
	Attributes map[string]interface{}
	UsageType  string
	OnDemand   struct {
		Dimensions []AWSPriceItemDimension
	}
}

var RegionLongNames = map[string]string{
	endpoints.UsEast1RegionID: "US East (N. Virginia)",
	endpoints.UsEast2RegionID: "US East (Ohio)",
	endpoints.UsWest2RegionID: "US West (Oregon)",
}

func ParsePriceItem(priceItem aws.JSONValue) (priceItemRet *AWSPriceItem, err error) {
	priceItemRet = &AWSPriceItem{}

	if product, ok := priceItem["product"].(map[string]interface{}); ok {
		if attributes, ok := product["attributes"].(map[string]interface{}); ok {
			usageType, ok := attributes["usagetype"].(string)
			if !ok {
				return nil, errors.New("Could not parse Attributes")
			}
			priceItemRet.Attributes = attributes
			priceItemRet.UsageType = usageType
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
								beginRange, ok3 := priceDimensionsValue["beginRange"].(string)
								endRange, ok4 := priceDimensionsValue["endRange"].(string)
								description, ok5 := priceDimensionsValue["description"].(string)

								if ok1 && ok2 && ok3 && ok4 && ok5 {
									if usd, ok := pricePerUnit["USD"].(string); ok {

										rate, err := strconv.ParseFloat(usd, 64)
										if err != nil {
											return nil, err
										}

										priceItemRet.OnDemand.Dimensions = append(priceItemRet.OnDemand.Dimensions, AWSPriceItemDimension{
											BeginRange:  beginRange,
											EndRange:    endRange,
											Unit:        unit,
											Rate:        rate,
											Description: description,
										})
									}
								} else {
									return nil, errors.New("Could not parse OnDemand")
								}
							}
						}

						return
					}
				}
			}
		}
	}

	return nil, errors.New("Could not parse OnDemand")
}
