package dynamodb

import (
	"context"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/cloudwatch"
	"github.com/aws/aws-sdk-go/service/cloudwatch/cloudwatchiface"
	"github.com/aws/aws-sdk-go/service/dynamodb"
	"github.com/aws/aws-sdk-go/service/dynamodb/dynamodbiface"

	util "github.com/timmyers/cloudwaste/pkg/aws/util"
)

const (
	tableType = "DynamoDB Table"
)

type Client struct {
	DynamoDB   dynamodbiface.DynamoDBAPI
	Cloudwatch cloudwatchiface.CloudWatchAPI
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
				// TODO: do somethign with errors in individual items
				if err != nil {
					continue
				}

				used := false

				for _, result := range tableMetrics.MetricDataResults {
					for _, value := range result.Values {
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
