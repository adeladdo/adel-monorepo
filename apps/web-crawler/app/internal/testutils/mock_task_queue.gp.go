package testutils

import (
	"context"
	"web-crawler/app/internal/core/domain"

	"github.com/stretchr/testify/mock"
)

type MockTaskQueue struct {
	mock.Mock
}

func (m *MockTaskQueue) ConsumeCrawlTask(ctx context.Context) (<-chan domain.CrawlTask, error) {
	args := m.Called(ctx)
	return args.Get(0).(chan domain.CrawlTask), args.Error(1)
}

func (m *MockTaskQueue) PublishCrawlTask(ctx context.Context, task domain.CrawlTask) error {
	args := m.Called(ctx, task)
	return args.Error(0)
}
