package outbound

import (
	"context"
	"testing"
	"time"
	"web-crawler/app/internal/core/domain"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestChannelTaskQueue_PublishAndConsume(t *testing.T) {
	queue := NewChannelTaskQueue(10)
	ctx := context.Background()

	task := domain.CrawlTask{
		URL:    "https://crawlme.monzo.com/start",
		Depth:  1,
		SiteID: "site-1",
	}

	require.NoError(t, queue.PublishCrawlTask(ctx, task))

	taskChan, err := queue.ConsumeCrawlTask(ctx)
	require.NoError(t, err)

	received := <-taskChan
	assert.Equal(t, task, received)
}

func TestChannelTaskQueue_PublishRespectsContextCancellation(t *testing.T) {
	queue := NewChannelTaskQueue(1)
	ctx := context.Background()

	task := domain.CrawlTask{
		URL:    "https://crawlme.monzo.com/start",
		Depth:  1,
		SiteID: "site-1",
	}

	require.NoError(t, queue.PublishCrawlTask(ctx, task))

	// buffer is full next publish should block
	ctx2, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	err := queue.PublishCrawlTask(ctx2, task)

	assert.Error(t, err)
	assert.ErrorIs(t, err, context.DeadlineExceeded)
}
