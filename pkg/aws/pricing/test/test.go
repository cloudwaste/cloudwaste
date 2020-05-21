package test

import (
	"context"

	"github.com/cloudwaste/cloudwaste/pkg/aws/pricing"
	"github.com/stretchr/testify/mock"
)

type MockedPricingInterface struct {
	mock.Mock
	pricing.PricingInterface
}

func (m *MockedPricingInterface) GetProducts(ctx context.Context, options *pricing.GetProductsInput) ([]*pricing.AWSPriceItem, error) {
	args := m.Called(ctx, options)

	if args.Error(1) == nil {
		return args.Get(0).([]*pricing.AWSPriceItem), nil
	}
	return nil, args.Error(1)
}
