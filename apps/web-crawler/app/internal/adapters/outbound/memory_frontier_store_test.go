package outbound

import (
	"context"
	"fmt"
	"testing"
	"time"
	"web-crawler/app/internal/core/domain"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMemoryFrontier_EnsureEntryIdempotent(t *testing.T) {
	ctx := context.Background()
	siteID := "site-1"
	url := "https://crawlme.monzo.com/start"

	// New entry
	frontier := NewMemoryFrontier()
	entry, isNew, err := frontier.EnsureEntry(ctx, siteID, url, 0, 100)

	require.NoError(t, err)
	require.True(t, isNew)
	require.Equal(t, "site-1", entry.SiteID)
	require.Equal(t, "https://crawlme.monzo.com/start", entry.URL)
	require.Equal(t, entry.Status, domain.URLStatusPending)

	// Return existing if duplicate
	entry2, isNew2, err := frontier.EnsureEntry(ctx, siteID, url, 0, 100)
	require.NoError(t, err)
	require.False(t, isNew2)
	require.Equal(t, entry.ID, entry2.ID)
}

func TestMemoryFrontier_DifferentEntriesMultipleSites(t *testing.T) {
	ctx := context.Background()

	// New entry
	frontier := NewMemoryFrontier()
	entry, _, _ := frontier.EnsureEntry(ctx, "site-1", "https://crawlme.monzo.com/start", 0, 100)
	entry2, _, _ := frontier.EnsureEntry(ctx, "site-1", "https://crawlme.monzo.com/page3", 0, 100)
	entry3, _, _ := frontier.EnsureEntry(ctx, "site-2", "https://crawlme.monzo.com/start", 0, 100)

	require.NotEqual(t, entry.ID, entry3.ID)
	require.NotEqual(t, entry2.ID, entry3.ID)

	stats1, err := frontier.Stats(ctx, "site-1")
	require.NoError(t, err)
	assert.Equal(t, 2, stats1.QueueSize)

	stats2, err := frontier.Stats(ctx, "site-2")
	require.NoError(t, err)
	assert.Equal(t, 1, stats2.QueueSize)
}

func TestMemoryFrontier_UpdateStatus(t *testing.T) {
	ctx := context.Background()
	siteID := "site-1"
	url := "https://crawlme.monzo.com/start"

	// New entry
	frontier := NewMemoryFrontier()
	entry, isNew, err := frontier.EnsureEntry(ctx, siteID, url, 0, 100)

	require.NoError(t, err)
	require.True(t, isNew)
	require.Equal(t, entry.Status, domain.URLStatusPending)

	err = frontier.MarkCrawling(ctx, entry.ID)
	require.NoError(t, err)

	entry, err = frontier.GetByID(ctx, entry.ID)
	require.NoError(t, err)
	require.Equal(t, entry.Status, domain.URLStatusCrawling)

	err = frontier.MarkCompleted(ctx, entry.ID, time.Now())
	require.NoError(t, err)

	entry, err = frontier.GetByID(ctx, entry.ID)
	require.NoError(t, err)
	require.Equal(t, entry.Status, domain.URLStatusCompleted)

}

func TestMemoryFrontier_StatsComputedCorrectly(t *testing.T) {
	ctx := context.Background()
	siteID := "site-1"
	url := "https://crawlme.monzo.com/start"

	// New entry
	frontier := NewMemoryFrontier()
	entry, _, _ := frontier.EnsureEntry(ctx, siteID, url, 0, 100)
	entry1, _, _ := frontier.EnsureEntry(ctx, siteID, "https://crawlme.monzo.com/page1", 0, 100)
	entry2, _, _ := frontier.EnsureEntry(ctx, siteID, "https://crawlme.monzo.com/page3", 0, 100)
	entry3, _, _ := frontier.EnsureEntry(ctx, siteID, "https://crawlme.monzo.com/page2", 0, 100)

	require.Equal(t, entry.Status, domain.URLStatusPending)

	_ = frontier.MarkCrawling(ctx, entry1.ID)
	_ = frontier.MarkCompleted(ctx, entry2.ID, time.Now())
	_ = frontier.MarkFailed(ctx, entry3.ID, fmt.Errorf("failed to fetch url"))

	stats, err := frontier.Stats(ctx, siteID)
	require.NoError(t, err)

	require.Equal(t, 2, stats.QueueSize)
	require.Equal(t, 1, stats.FailedCount)
	require.Equal(t, 1, stats.VisitedCount)
}
