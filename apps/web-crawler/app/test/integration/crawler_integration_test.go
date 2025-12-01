package integration

import (
	"context"
	"log/slog"
	"net/url"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	outbound "web-crawler/app/internal/adapters/outbound"
	"web-crawler/app/internal/core/domain"
	"web-crawler/app/internal/core/services"
)

func TestCrawler_Integration(t *testing.T) {

	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))

	frontier := outbound.NewMemoryFrontier()
	queue := outbound.NewChannelTaskQueue(100)
	client := outbound.NewStdHTTPClient(10 * time.Second)
	parser := outbound.NewGoqueryHTMLParser()

	crawler := services.NewCrawlerService(
		frontier,
		queue,
		client,
		parser,
		logger,
		2,
	)

	u, err := url.Parse("https://crawlme.monzo.com/")
	require.NoError(t, err)

	site := domain.Site{
		ID:          uuid.NewString(),
		StartURL:    u,
		AllowedHost: u.Host,
		MaxDepth:    1,
	}

	err = crawler.StartCrawl(ctx, site)
	require.NoError(t, err, "crawl should complete without error")

	stats, err := frontier.Stats(ctx, site.ID)
	require.NoError(t, err)

	assert.Greater(t, stats.VisitedCount, 0, "should visit at least the start URL")
	assert.Equal(t, 0, stats.QueueSize, "queue should be empty after crawl completes")

	t.Logf("integration stats: visited=%d failed=%d", stats.VisitedCount, stats.FailedCount)
}

func TestCrawler_Integration_ConcurrentWorkers(t *testing.T) {

	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))

	frontier := outbound.NewMemoryFrontier()
	queue := outbound.NewChannelTaskQueue(100)
	client := outbound.NewStdHTTPClient(10 * time.Second)
	parser := outbound.NewGoqueryHTMLParser()

	crawler := services.NewCrawlerService(
		frontier,
		queue,
		client,
		parser,
		logger,
		50,
	)

	u, err := url.Parse("https://crawlme.monzo.com/")
	require.NoError(t, err)

	site := domain.Site{
		ID:          uuid.NewString(),
		StartURL:    u,
		AllowedHost: u.Host,
		MaxDepth:    2,
	}

	err = crawler.StartCrawl(ctx, site)
	require.NoError(t, err, "crawl with many workers should not deadlock or error")

	stats, err := frontier.Stats(ctx, site.ID)
	require.NoError(t, err)

	assert.Greater(t, stats.VisitedCount, 0, "should visit at least the start URL")
	assert.Equal(t, 0, stats.QueueSize, "queue should be empty after crawl completes")

	t.Logf("integration stats: visited=%d failed=%d", stats.VisitedCount, stats.FailedCount)
}
