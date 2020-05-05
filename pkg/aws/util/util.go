package util

import "github.com/aws/aws-sdk-go/aws/endpoints"

type AWSResource interface {
	Type() string
	ID() string
}

type AWSResourceObject struct {
	R AWSResource
}

type Price struct {
	Unit string
	Rate string
}

var RegionLongNames = map[string]string{
	endpoints.UsEast1RegionID: "US East (N. Virginia)",
	endpoints.UsEast2RegionID: "US East (Ohio)",
}
