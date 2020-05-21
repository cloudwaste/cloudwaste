package dynamodb

import (
	"context"
	"strconv"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/cloudwatch"
	"github.com/aws/aws-sdk-go/service/cloudwatch/cloudwatchiface"
	"github.com/aws/aws-sdk-go/service/dynamodb"
	"github.com/aws/aws-sdk-go/service/dynamodb/dynamodbiface"
	"github.com/aws/aws-sdk-go/service/pricing"

	pricingWaste "github.com/cloudwaste/cloudwaste/pkg/aws/pricing"
	util "github.com/cloudwaste/cloudwaste/pkg/aws/util"
)

type DynamoPricingFacet string

const (
	ReadCapacityUnitHours      DynamoPricingFacet = "ReadCapacityUnit-Hrs"
	WriteCapacityUnitHours     DynamoPricingFacet = "WriteCapacityUnit-Hrs"
	ReplWriteCapacityUnitHours DynamoPricingFacet = "ReplWriteCapacityUnit-Hrs"
)

const (
	tableType = "DynamoDB Table"
)

type Client struct {
	DynamoDB   dynamodbiface.DynamoDBAPI
	Cloudwatch cloudwatchiface.CloudWatchAPI
	Pricing    pricingWaste.PricingInterface
}

type DynamoDBTable struct {
	r *dynamodb.TableDescription
}

func (a DynamoDBTable) Type() string {
	return tableType
}

func (a DynamoDBTable) ID() string {
	return aws.StringValue(a.r.TableName)
}

func (client *Client) AnalyzeDynamodBTableWaste(ctx context.Context, region string) ([]util.AWSWastedResource, error) {
	pricing, err := client.GetDynamoDBTablePricing(ctx, region)
	if err != nil {
		return nil, err
	}

	unusedTables, err := client.GetUnusedDynamoDBTables(ctx)
	if err != nil {
		return nil, err
	}

	var wastedResources []util.AWSWastedResource

	for _, unusedResource := range unusedTables {
		unusedTable, ok := unusedResource.R.(*DynamoDBTable)
		if !ok {
			return nil, util.PricingError
		}

		// TODO: account for other things like storage, streams, etc...
		if unusedTable.r.BillingModeSummary == nil ||
			(unusedTable.r.BillingModeSummary.BillingMode != nil &&
				*unusedTable.r.BillingModeSummary.BillingMode == "PROVISIONED") {
			readUnits := *unusedTable.r.ProvisionedThroughput.ReadCapacityUnits
			writeUnits := *unusedTable.r.ProvisionedThroughput.WriteCapacityUnits

			rate := pricing[ReadCapacityUnitHours].Rate*float64(readUnits) +
				pricing[WriteCapacityUnitHours].Rate*float64(writeUnits)

			wastedResources = append(wastedResources, util.AWSWastedResource{
				Resource: unusedResource,
				Price: util.Price{
					Unit: "Hr",
					Rate: rate,
				},
			})
		} else {
			wastedResources = append(wastedResources, util.AWSWastedResource{
				Resource: unusedResource,
				Price: util.Price{
					Unit: "Hr",
					Rate: 0.0,
				},
			})
		}
	}

	return wastedResources, nil
}

func (client *Client) GetUnusedDynamoDBTables(ctx context.Context) ([]util.AWSResourceObject, error) {
	var unusedTables []util.AWSResourceObject

	err := client.DynamoDB.ListTablesPagesWithContext(ctx, &dynamodb.ListTablesInput{},
		func(page *dynamodb.ListTablesOutput, lastPage bool) bool {
			for _, tableName := range page.TableNames {
				tableOutput, err := client.DynamoDB.DescribeTableWithContext(ctx, &dynamodb.DescribeTableInput{
					TableName: tableName,
				})
				// TODO: do somethign with errors in individual items
				if err != nil {
					continue
				}

				table := tableOutput.Table

				if *table.ItemCount == 0 || *table.TableSizeBytes == 0 {
					unusedTables = append(unusedTables, util.AWSResourceObject{R: &DynamoDBTable{table}})
				}

				startTime := time.Now().Add(-24 * 14 * time.Hour)
				endTime := time.Now()

				tableMetrics, err := client.Cloudwatch.GetMetricDataWithContext(ctx, &cloudwatch.GetMetricDataInput{
					StartTime: &startTime,
					EndTime:   &endTime,
					MetricDataQueries: []*cloudwatch.MetricDataQuery{
						{
							Id: aws.String("readcapacity"),
							MetricStat: &cloudwatch.MetricStat{
								Period: aws.Int64(5 * 60),
								Stat:   aws.String("Average"),
								Metric: &cloudwatch.Metric{
									MetricName: aws.String("ConsumedReadCapacityUnits"),
									Namespace:  aws.String("AWS/DynamoDB"),
									Dimensions: []*cloudwatch.Dimension{
										{
											Name:  aws.String("TableName"),
											Value: table.TableName,
										},
									},
								},
							},
						},
					},
				})
				// TODO: do something with errors in individual items
				if err != nil {
					continue
				}

				used := false

				for _, result := range tableMetrics.MetricDataResults {
					for _, value := range result.Values {
						// NOTE: Just looking at the table in the console will make it look used
						if *value != 0.0 {
							used = true
						}
					}
				}

				if !used {
					unusedTables = append(unusedTables, util.AWSResourceObject{R: &DynamoDBTable{table}})
				}
			}

			return true
		})

	if err != nil {
		return nil, err
	}

	return unusedTables, nil
}

func (client *Client) GetDynamoDBTablePricing(ctx context.Context, region string) (map[DynamoPricingFacet]*util.Price, error) {
	pricing, err := client.Pricing.GetProducts(ctx, &pricingWaste.GetProductsInput{
		Region:      region,
		ServiceCode: pricingWaste.DynamoDB,
		Filters:     []*pricing.Filter{},
	})
	if err != nil {
		return nil, err
	}

	dynamoDBPricing := make(map[DynamoPricingFacet]*util.Price)

	for _, pricingItem := range pricing {
		usageType := pricingItem.Product.Attributes.UsageType

		if strings.Contains(usageType, string(ReadCapacityUnitHours)) {
			for _, term := range pricingItem.Terms.OnDemand {
				for _, priceDimension := range term.PriceDimensions {
					if priceDimension.EndRange == "Inf" {
						rate, err := strconv.ParseFloat(priceDimension.PricePerUnit.USD, 64)
						if err != nil {
							return nil, err
						}

						dynamoDBPricing[ReadCapacityUnitHours] = &util.Price{
							Rate: rate,
							Unit: "Hr",
						}
					}
				}
			}
		} else if strings.Contains(usageType, string(WriteCapacityUnitHours)) &&
			!strings.Contains(usageType, string(ReplWriteCapacityUnitHours)) {
			for _, term := range pricingItem.Terms.OnDemand {
				for _, priceDimension := range term.PriceDimensions {
					if priceDimension.EndRange == "Inf" {
						rate, err := strconv.ParseFloat(priceDimension.PricePerUnit.USD, 64)
						if err != nil {
							return nil, err
						}

						dynamoDBPricing[WriteCapacityUnitHours] = &util.Price{
							Rate: rate,
							Unit: "Hr",
						}
					}
				}
			}
		}

	}

	return dynamoDBPricing, nil
}
