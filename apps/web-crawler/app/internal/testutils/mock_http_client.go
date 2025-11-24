package testutils

import (
	"context"
	"web-crawler/app/internal/core/ports/outbound"

	"github.com/stretchr/testify/mock"
)

type MockHttpClient struct {
	mock.Mock
}

func (m *MockHttpClient) Fetch(ctx context.Context, url string) (outbound.HttpResponse, error) {
	args := m.Called(ctx, url)
	return args.Get(0).(outbound.HttpResponse), args.Error(1)
}
