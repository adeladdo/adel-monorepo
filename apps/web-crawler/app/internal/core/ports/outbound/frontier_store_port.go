package outbound

import (
	"context"
	"time"
	"web-crawler/app/internal/core/domain"
)

type FrontierStore interface {
	EnsureEntry(ctx context.Context, siteID string, url string, depth int, priority int) (*domain.FrontierEntry, bool, error)
	MarkCrawling(ctx context.Context, id string) error
	MarkCompleted(ctx context.Context, id string, visitedAt time.Time) error
	MarkFailed(ctx context.Context, id string, err error) error

	Stats(ctx context.Context, siteID string) (*domain.FrontierStats, error)
	GetByID(ctx context.Context, id string) (*domain.FrontierEntry, error)
	GetByURL(ctx context.Context, url string, siteID string) (*domain.FrontierEntry, error)
}
