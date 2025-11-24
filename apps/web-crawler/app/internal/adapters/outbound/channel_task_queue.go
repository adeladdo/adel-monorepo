package outbound

import (
	"context"
	"web-crawler/app/internal/core/domain"
)

type ChannelTaskQueue struct {
	taskChan chan domain.CrawlTask
}

func NewChannelTaskQueue(bufferSize int) *ChannelTaskQueue {
	return &ChannelTaskQueue{
		taskChan: make(chan domain.CrawlTask, bufferSize),
	}
}

func (c *ChannelTaskQueue) PublishCrawlTask(ctx context.Context, task domain.CrawlTask) error {
	select {
	case c.taskChan <- task:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

func (c *ChannelTaskQueue) ConsumeCrawlTask(ctx context.Context) (<-chan domain.CrawlTask, error) {
	return c.taskChan, nil
}
