package inbound

import (
	"context"
	"web-crawler/app/internal/core/domain"
)

type CrawlerAPI interface {
	StartCrawl(ctx context.Context, site domain.Site)
}
