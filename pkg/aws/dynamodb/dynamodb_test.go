package dynamodb

import (
	"context"
	"errors"
	"testing"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/request"
	"github.com/aws/aws-sdk-go/service/cloudwatch"
	"github.com/aws/aws-sdk-go/service/cloudwatch/cloudwatchiface"
	"github.com/aws/aws-sdk-go/service/dynamodb"
	"github.com/aws/aws-sdk-go/service/dynamodb/dynamodbiface"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

type mockData struct {
	Table   *dynamodb.DescribeTableOutput
	Metrics *cloudwatch.GetMetricDataOutput
}

var tables = map[string]mockData{
	"table1": { // used
		Table: &dynamodb.DescribeTableOutput{
			Table: &dynamodb.TableDescription{
				TableName:      aws.String("table1"),
				ItemCount:      aws.Int64(2),
				TableSizeBytes: aws.Int64(30),
			},
		},
		Metrics: &cloudwatch.GetMetricDataOutput{
			MetricDataResults: []*cloudwatch.MetricDataResult{{
				Id:     aws.String("readcapacity"),
				Values: aws.Float64Slice([]float64{0.0, 1.0}),
			}},
		},
	},
	"table2": { // unused
		Table: &dynamodb.DescribeTableOutput{
			Table: &dynamodb.TableDescription{
				TableName:      aws.String("table2"),
				ItemCount:      aws.Int64(4),
				TableSizeBytes: aws.Int64(0),
			},
		},
		Metrics: &cloudwatch.GetMetricDataOutput{
			MetricDataResults: []*cloudwatch.MetricDataResult{{
				Id:     aws.String("readcapacity"),
				Values: aws.Float64Slice([]float64{0.0, 1.0}),
			}},
		},
	},
	"table3": { // unused
		Table: &dynamodb.DescribeTableOutput{
			Table: &dynamodb.TableDescription{
				TableName:      aws.String("table3"),
				ItemCount:      aws.Int64(0),
				TableSizeBytes: aws.Int64(4),
			},
		},
		Metrics: &cloudwatch.GetMetricDataOutput{
			MetricDataResults: []*cloudwatch.MetricDataResult{{
				Id:     aws.String("readcapacity"),
				Values: aws.Float64Slice([]float64{0.0, 1.0}),
			}},
		},
	},
	"table4": { // unused
		Table: &dynamodb.DescribeTableOutput{
			Table: &dynamodb.TableDescription{
				TableName:      aws.String("table4"),
				ItemCount:      aws.Int64(4),
				TableSizeBytes: aws.Int64(4),
			},
		},
		Metrics: &cloudwatch.GetMetricDataOutput{
			MetricDataResults: []*cloudwatch.MetricDataResult{{
				Id:     aws.String("readcapacity"),
				Values: aws.Float64Slice([]float64{0.0, 0.0}),
			}},
		},
	},
}

type mockedDynamoDB struct {
	mock.Mock
	dynamodbiface.DynamoDBAPI
}

type mockedCloudwatch struct {
	mock.Mock
	cloudwatchiface.CloudWatchAPI
}

func (m *mockedDynamoDB) ListTablesPagesWithContext(ctx context.Context, input *dynamodb.ListTablesInput, fn func(*dynamodb.ListTablesOutput, bool) bool, opts ...request.Option) error {
	args := m.Called(ctx, input, fn)

	if args.Error(1) == nil {
		fn(args.Get(0).(*dynamodb.ListTablesOutput), true)
	}
	return args.Error(1)
}

func (m *mockedDynamoDB) DescribeTableWithContext(ctx context.Context, input *dynamodb.DescribeTableInput, options ...request.Option) (*dynamodb.DescribeTableOutput, error) {
	args := m.Called(ctx, input, options)

	return tables[*input.TableName].Table, args.Error(0)
}

func (m *mockedCloudwatch) GetMetricDataWithContext(ctx context.Context, input *cloudwatch.GetMetricDataInput, options ...request.Option) (*cloudwatch.GetMetricDataOutput, error) {
	args := m.Called(ctx, input, options)

	return tables[*input.MetricDataQueries[0].MetricStat.Metric.Dimensions[0].Value].Metrics, args.Error(0)
}

func TestGetUnusedDynamoDBTables(t *testing.T) {
	assert := assert.New(t)

	tableNames := make([]*string, 0, len(tables))
	for k := range tables {
		tableNames = append(tableNames, aws.String(k))
	}

	md := new(mockedDynamoDB)
	mc := new(mockedCloudwatch)
	md.On("ListTablesPagesWithContext", mock.Anything, mock.Anything, mock.Anything).
		Return(&dynamodb.ListTablesOutput{
			TableNames: tableNames,
		}, nil)
	md.On("DescribeTableWithContext", mock.Anything, mock.Anything, mock.Anything).
		Return(nil)
	mc.On("GetMetricDataWithContext", mock.Anything, mock.Anything, mock.Anything).
		Return(nil)

	client := Client{DynamoDB: md, Cloudwatch: mc}
	unusedTables, err := client.GetUnusedDynamoDBTables(context.Background())

	assert.Equal(3, len(unusedTables))

	unusedTableNames := make([]string, 0, len(unusedTables))
	for _, unusedTable := range unusedTables {
		unusedTableNames = append(unusedTableNames, unusedTable.R.ID())
	}

	assert.NotContains(unusedTableNames, "table1")
	assert.Contains(unusedTableNames, "table2")
	assert.Contains(unusedTableNames, "table3")
	assert.Contains(unusedTableNames, "table4")
	assert.Nil(err)

	assert.Equal(tableType, unusedTables[0].R.Type())

	// Test error cases
	md = new(mockedDynamoDB)
	mc = new(mockedCloudwatch)
	md.On("ListTablesPagesWithContext", mock.Anything, mock.Anything, mock.Anything).
		Return(nil, errors.New("failed"))

	client = Client{DynamoDB: md, Cloudwatch: mc}
	unusedTables, err = client.GetUnusedDynamoDBTables(context.Background())
	assert.Nil(unusedTables)
	assert.NotNil(err)
}
