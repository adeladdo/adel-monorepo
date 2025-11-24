package services

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"net/url"
	"strings"
	"testing"
	"web-crawler/app/internal/core/domain"
	"web-crawler/app/internal/core/ports/outbound"
	"web-crawler/app/internal/testutils"

	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestSubmitDiscoveredURL(t *testing.T) {
	tests := []struct {
		name              string
		allowedHost       string
		maxDepth          int
		depth             int
		priority          int
		url               string
		ensureEntryCalled bool
		publishCalled     bool
	}{
		{name: "skips external hosts",
			allowedHost:       "crawlme.monzo.com",
			maxDepth:          5,
			depth:             1,
			priority:          99,
			url:               "https://facebook.com/page",
			ensureEntryCalled: false,
			publishCalled:     false,
		},
		{name: "crawls same subdomain",
			allowedHost:       "crawlme.monzo.com",
			maxDepth:          5,
			depth:             1,
			priority:          99,
			url:               "https://crawlme.monzo.com/page",
			ensureEntryCalled: true,
			publishCalled:     true,
		},
		{name: "skips invalid scheme",
			allowedHost:       "crawlme.monzo.com",
			maxDepth:          5,
			depth:             1,
			priority:          99,
			url:               "mailto:foo@bar.com",
			ensureEntryCalled: false,
			publishCalled:     false,
		},
		{name: "skips links after max depth",
			allowedHost:       "crawlme.monzo.com",
			maxDepth:          5,
			depth:             6,
			priority:          99,
			url:               "mailto:foo@bar.com",
			ensureEntryCalled: false,
			publishCalled:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			ctx := context.Background()

			site := domain.Site{
				ID:          "site-1",
				AllowedHost: tt.allowedHost,
				MaxDepth:    tt.maxDepth,
			}

			mockFrontier := &testutils.MockFrontierStore{}
			mockTaskQueue := &testutils.MockTaskQueue{}
			logger := slog.New(slog.NewTextHandler(io.Discard, nil))

			entry := &domain.FrontierEntry{
				ID:     "frontier-id",
				URL:    tt.url,
				Depth:  tt.depth,
				Status: domain.URLStatusPending,
			}

			task := domain.CrawlTask{
				EntryID: entry.ID,
				URL:     entry.URL,
				Depth:   entry.Depth,
				SiteID:  site.ID,
			}
			if tt.ensureEntryCalled {
				mockFrontier.On("EnsureEntry", ctx, site.ID, tt.url, tt.depth, tt.priority).Return(entry, true, nil)
			}
			if tt.publishCalled {
				mockTaskQueue.On("PublishCrawlTask", ctx,
					mock.MatchedBy(func(t domain.CrawlTask) bool {
						return t.URL == tt.url && t.Depth == tt.depth && t.SiteID == site.ID
					})).Return(nil)
			}

			svc := NewCrawlerService(mockFrontier, mockTaskQueue, nil, nil, logger, 1)

			link, err := url.Parse(tt.url)
			require.NoError(t, err)

			err = svc.submitDiscoveredURL(context.Background(), site, link, tt.depth)
			require.NoError(t, err)

			if tt.ensureEntryCalled {
				mockFrontier.AssertCalled(t, "EnsureEntry", mock.Anything, site.ID, tt.url, tt.depth, tt.priority)
			} else {
				mockFrontier.AssertNotCalled(t, "EnsureEntry", mock.Anything, site.ID, tt.url, tt.depth, tt.priority)
			}

			if tt.publishCalled {
				mockTaskQueue.AssertCalled(t, "PublishCrawlTask", mock.Anything, task)
			} else {
				mockTaskQueue.AssertNotCalled(t, "PublishCrawlTask", mock.Anything, task)
			}

			mockFrontier.AssertExpectations(t)
			mockTaskQueue.AssertExpectations(t)
		})
	}
}

func TestProcessTask(t *testing.T) {
	tests := []struct {
		name                string
		url                 string
		fetchResp           outbound.HttpResponse
		fetchError          error
		parseLinks          []string
		parseError          error
		expectStatus        domain.URLStatus
		expectSubmittedURLs []string
	}{
		{
			name: "fetches, parses and submits internal links",
			url:  "https://crawlme.monzo.com/page1",
			fetchResp: outbound.HttpResponse{
				StatusCode:  200,
				Body:        []byte("<html><a href='/link1'>Link 1</a></html>"),
				ContentType: "text/html",
			},
			parseLinks:          []string{"https://crawlme.monzo.com/link1"},
			expectStatus:        domain.URLStatusCompleted,
			expectSubmittedURLs: []string{"https://crawlme.monzo.com/link1"},
		},
		{
			name:         "fetch fails - marks as failed",
			url:          "https://crawlme.monzo.com/page2",
			fetchError:   errors.New("network timeout"),
			expectStatus: domain.URLStatusFailed,
		},
		{
			name: "HTTP 404 - marks as failed",
			url:  "https://crawlme.monzo.com/notfound",
			fetchResp: outbound.HttpResponse{
				StatusCode:  404,
				Body:        []byte("Not Found"),
				ContentType: "text/html",
			},
			expectStatus: domain.URLStatusFailed,
		},
		{
			name: "non-HTML content - marks completed",
			url:  "https://crawlme.monzo.com/document.pdf",
			fetchResp: outbound.HttpResponse{
				StatusCode:  200,
				Body:        []byte("%PDF-1.4..."),
				ContentType: "application/pdf",
			},
			expectStatus: domain.URLStatusCompleted,
		},
		{
			name: "parse fails - marks completed anyway",
			url:  "https://crawlme.monzo.com/malformed",
			fetchResp: outbound.HttpResponse{
				StatusCode:  200,
				Body:        []byte("<html><div>Malformed</html>"),
				ContentType: "text/html",
			},
			parseError:   errors.New("malformed HTML"),
			expectStatus: domain.URLStatusCompleted,
		},
		{
			name: "no links found - marks completed",
			url:  "https://crawlme.monzo.com/empty",
			fetchResp: outbound.HttpResponse{
				StatusCode:  200,
				Body:        []byte("<html><p>No links here</p></html>"),
				ContentType: "text/html",
			},
			parseLinks:   []string{},
			expectStatus: domain.URLStatusCompleted,
		},
		{
			name: "filters external links",
			url:  "https://crawlme.monzo.com/page",
			fetchResp: outbound.HttpResponse{
				StatusCode:  200,
				Body:        []byte("<html><a href='/internal'>Internal</a><a href='https://facebook.com'>External</a></html>"),
				ContentType: "text/html",
			},
			parseLinks: []string{
				"https://crawlme.monzo.com/internal",
				"https://facebook.com/page",
			},
			expectStatus:        domain.URLStatusCompleted,
			expectSubmittedURLs: []string{"https://crawlme.monzo.com/internal"}, // External filtered
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()

			site := domain.Site{
				ID:          "site-1",
				AllowedHost: "crawlme.monzo.com",
				MaxDepth:    5,
			}

			const entryID = "entry-id-1"
			task := domain.CrawlTask{
				EntryID: entryID,
				URL:     tt.url,
				SiteID:  site.ID,
				Depth:   0,
			}

			mockFrontier := &testutils.MockFrontierStore{}
			mockQueue := &testutils.MockTaskQueue{}
			mockHTTP := &testutils.MockHttpClient{}
			mockParser := &testutils.MockHTMLParser{}
			logger := slog.New(slog.NewTextHandler(io.Discard, nil))

			mockFrontier.On("MarkCrawling", mock.Anything, entryID).Return(nil).Once()

			if tt.fetchError != nil {
				mockHTTP.On("Fetch", mock.Anything, tt.url).Return(outbound.HttpResponse{}, tt.fetchError).Once()
			} else {
				mockHTTP.On("Fetch", mock.Anything, tt.url).Return(tt.fetchResp, nil).Once()
			}

			shouldParse := tt.fetchError == nil &&
				tt.fetchResp.StatusCode == 200 && strings.HasPrefix(tt.fetchResp.ContentType, "text/html")

			if shouldParse {
				baseURL, _ := url.Parse(tt.url)
				if tt.parseError != nil {
					mockParser.
						On("ExtractLinks", baseURL, string(tt.fetchResp.Body)).
						Return([]*url.URL(nil), tt.parseError).
						Once()
				} else {
					parsedLinks := make([]*url.URL, len(tt.parseLinks))
					for i, link := range tt.parseLinks {
						parsedLinks[i], _ = url.Parse(link)
					}
					mockParser.On("ExtractLinks", baseURL, string(tt.fetchResp.Body)).Return(parsedLinks, nil).Once()
				}
			}

			for _, submittedURL := range tt.expectSubmittedURLs {
				entry := &domain.FrontierEntry{
					ID:     "new-entry-" + submittedURL,
					URL:    submittedURL,
					Depth:  task.Depth + 1,
					Status: domain.URLStatusPending,
				}
				mockFrontier.On("EnsureEntry",
					mock.Anything,
					site.ID,
					submittedURL,
					task.Depth+1,
					mock.AnythingOfType("int"),
				).Return(entry, true, nil).Once()

				mockQueue.On("PublishCrawlTask",
					mock.Anything,
					mock.MatchedBy(func(t domain.CrawlTask) bool {
						return t.URL == submittedURL && t.Depth == task.Depth+1
					}),
				).Return(nil).Once()
			}

			switch tt.expectStatus {
			case domain.URLStatusCompleted:
				mockFrontier.On("MarkCompleted", mock.Anything, entryID, mock.AnythingOfType("time.Time")).Return(nil).Once()
			case domain.URLStatusFailed:
				mockFrontier.On("MarkFailed", mock.Anything, entryID, mock.Anything).Return(nil).Once()
			}

			svc := NewCrawlerService(mockFrontier, mockQueue, mockHTTP, mockParser, logger, 1)
			svc.processTask(ctx, site, task)

			mockFrontier.AssertExpectations(t)
			mockQueue.AssertExpectations(t)
			mockHTTP.AssertExpectations(t)
			mockParser.AssertExpectations(t)
		})
	}
}

func TestEnqueueURL(t *testing.T) {
	tests := []struct {
		name          string
		url           string
		depth         int
		priority      int
		isNew         bool
		entryStatus   domain.URLStatus
		shouldPublish bool
	}{
		{
			name:          "new URL - should publish",
			url:           "https://crawlme.monzo.com/new",
			depth:         0,
			priority:      100,
			isNew:         true,
			entryStatus:   domain.URLStatusPending,
			shouldPublish: true,
		},
		{
			name:          "duplicate + failed - should publish (retry)",
			url:           "https://crawlme.monzo.com/failed",
			depth:         1,
			priority:      99,
			isNew:         false,
			entryStatus:   domain.URLStatusFailed,
			shouldPublish: true,
		},
		{
			name:          "duplicate + completed - should NOT publish",
			url:           "https://crawlme.monzo.com/done",
			depth:         1,
			priority:      99,
			isNew:         false,
			entryStatus:   domain.URLStatusCompleted,
			shouldPublish: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			siteID := "site-1"

			mockFrontier := &testutils.MockFrontierStore{}
			mockQueue := &testutils.MockTaskQueue{}
			logger := slog.New(slog.NewTextHandler(io.Discard, nil))

			entry := &domain.FrontierEntry{
				ID:     "entry-id-1",
				SiteID: siteID,
				URL:    tt.url,
				Depth:  tt.depth,
				Status: tt.entryStatus,
			}
			mockFrontier.On("EnsureEntry", mock.Anything, siteID, tt.url, tt.depth, tt.priority).Return(entry, tt.isNew, nil).Once()

			if tt.shouldPublish {
				mockQueue.On("PublishCrawlTask", mock.Anything,
					mock.MatchedBy(func(t domain.CrawlTask) bool {
						return t.EntryID == entry.ID && t.URL == tt.url &&
							t.Depth == tt.depth && t.SiteID == siteID
					})).Return(nil).Once()
			}

			svc := NewCrawlerService(mockFrontier, mockQueue, nil, nil, logger, 1)
			returnedEntry, err := svc.enqueueURL(ctx, siteID, tt.url, tt.depth, tt.priority)

			require.NoError(t, err)
			require.Equal(t, entry, returnedEntry)

			mockFrontier.AssertExpectations(t)
			mockQueue.AssertExpectations(t)

			if !tt.shouldPublish {
				mockQueue.AssertNotCalled(t, "PublishCrawlTask")
			}
		})
	}
}
