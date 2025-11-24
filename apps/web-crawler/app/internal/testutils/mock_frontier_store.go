package testutils

import (
	"context"
	"time"
	"web-crawler/app/internal/core/domain"

	"github.com/stretchr/testify/mock"
)

type MockFrontierStore struct {
	mock.Mock
}

func (m *MockFrontierStore) EnsureEntry(ctx context.Context, siteID string, url string, depth int, prioroty int) (*domain.FrontierEntry, bool, error) {
	args := m.Called(ctx, siteID, url, depth, prioroty)
	return args.Get(0).(*domain.FrontierEntry), args.Bool(1), args.Error(2)
}

func (m *MockFrontierStore) GetByID(ctx context.Context, id string) (*domain.FrontierEntry, error) {
	args := m.Called(ctx, id)
	return args.Get(0).(*domain.FrontierEntry), args.Error(1)
}
func (m *MockFrontierStore) GetByURL(ctx context.Context, url string, siteID string) (*domain.FrontierEntry, error) {
	args := m.Called(ctx, url, siteID)
	return args.Get(0).(*domain.FrontierEntry), args.Error(1)
}

func (m *MockFrontierStore) MarkCrawling(ctx context.Context, id string) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}
func (m *MockFrontierStore) MarkCompleted(ctx context.Context, id string, visitedAt time.Time) error {
	args := m.Called(ctx, id, visitedAt)
	return args.Error(0)
}
func (m *MockFrontierStore) MarkFailed(ctx context.Context, id string, err error) error {
	args := m.Called(ctx, id, time.Now())
	return args.Error(0)
}

func (m *MockFrontierStore) Stats(ctx context.Context, siteID string) (*domain.FrontierStats, error) {
	args := m.Called(ctx, siteID)
	return args.Get(0).(*domain.FrontierStats), args.Error(1)
}
