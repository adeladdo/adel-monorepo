package outbound

import (
	"context"
	"web-crawler/app/internal/core/domain"
)

type TaskQueue interface {
	PublishCrawlTask(ctx context.Context, task domain.CrawlTask) error
	ConsumeCrawlTask(ctx context.Context) (<-chan domain.CrawlTask, error)
}
