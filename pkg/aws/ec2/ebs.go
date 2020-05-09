package ec2

import (
	"context"
	"errors"
	"strconv"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/aws/aws-sdk-go/service/pricing"
	util "github.com/timmyers/cloudwaste/pkg/aws/util"
)

const (
	ebsVolumeType = "EBS Volume"
)

type EBSVolume struct {
	r *ec2.Volume
}

type EBSVolumeType string
type EBSVolumePricing map[EBSVolumeType]*util.AWSPriceItem

func (r EBSVolume) Type() string {
	return ebsVolumeType
}

func (r EBSVolume) ID() string {
	return aws.StringValue(r.r.VolumeId)
}

func (r EBSVolume) VolumeType() EBSVolumeType {
	volumeType := aws.StringValue(r.r.VolumeType)
	return EBSVolumeType(volumeType)
}

func (r EBSVolume) VolumeSizeinGb() int64 {
	return *r.r.Size
}

func (client *Client) AnalyzeEBSVolumeWaste(ctx context.Context, region string) ([]util.AWSWastedResource, error) {
	pricing, err := client.GetEBSVolumePricing(ctx, region)
	if err != nil {
		return nil, err
	}

	unusedVolumes, err := client.GetUnusedEBSVolumes(ctx)
	if err != nil {
		return nil, err
	}

	var wastedResources []util.AWSWastedResource

	for _, unusedResource := range unusedVolumes {
		unusedVolume, ok := unusedResource.R.(*EBSVolume)
		if !ok {
			return nil, errors.New("Failed to analyze EBS volume waste")
		}

		volumeTypePricing, ok := pricing[unusedVolume.VolumeType()]
		if !ok {
			return nil, errors.New("Failed to analyze EBS volume waste")
		}

		if volumeTypePricing.OnDemand.Unit != "GB-Mo" {
			return nil, errors.New("Unhandled pricing unit")
		}

		rate, err := strconv.ParseFloat(volumeTypePricing.OnDemand.Rate, 64)
		if err != nil {
			return nil, err
		}

		wastedResources = append(wastedResources, util.AWSWastedResource{
			Resource: unusedResource,
			Price: util.Price{
				Unit: volumeTypePricing.OnDemand.Unit,
				Rate: rate * float64(unusedVolume.VolumeSizeinGb()),
			},
		})
	}

	return wastedResources, nil
}

func (client *Client) GetUnusedEBSVolumes(ctx context.Context) ([]util.AWSResourceObject, error) {
	var unusedVolumes []util.AWSResourceObject

	err := client.EC2.DescribeVolumesPagesWithContext(ctx, &ec2.DescribeVolumesInput{},
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

func (client *Client) GetEBSVolumePricing(ctx context.Context, region string) (EBSVolumePricing, error) {
	regionName := util.RegionLongNames[region]

	resp, err := client.Pricing.GetProductsWithContext(ctx, &pricing.GetProductsInput{
		ServiceCode: aws.String(serviceCode),
		Filters: []*pricing.Filter{
			{
				Type:  aws.String("TERM_MATCH"),
				Field: aws.String("location"),
				Value: aws.String(regionName),
			},
			{
				Type:  aws.String("TERM_MATCH"),
				Field: aws.String("productFamily"),
				Value: aws.String("Storage"),
			},
		},
	})
	if err != nil {
		return nil, err
	}

	pricing := EBSVolumePricing{}

	for _, priceItemJson := range resp.PriceList {
		priceItem, err := util.ParsePriceItem(priceItemJson)
		if err != nil {
			return nil, err
		}

		if volumeType, ok := priceItem.Attributes["volumeApiName"].(string); ok {
			pricing[EBSVolumeType(volumeType)] = priceItem
		}
	}

	return pricing, nil
}
