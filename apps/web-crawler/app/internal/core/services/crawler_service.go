package services

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"
	"web-crawler/app/internal/core/domain"
	"web-crawler/app/internal/core/ports/outbound"
)

type CrawlerService struct {
	frontierStore outbound.FrontierStore
	taskQueue     outbound.TaskQueue
	httpClient    outbound.HttpClient
	htmlParser    outbound.HTMLParser
	logger        *slog.Logger

	workerCount int
}

func NewCrawlerService(frontierStore outbound.FrontierStore,
	taskQueue outbound.TaskQueue,
	httpClient outbound.HttpClient,
	htmlParser outbound.HTMLParser,
	logger *slog.Logger,
	workerCount int) *CrawlerService {
	return &CrawlerService{
		frontierStore: frontierStore,
		taskQueue:     taskQueue,
		httpClient:    httpClient,
		htmlParser:    htmlParser,
		logger:        logger,
		workerCount:   workerCount,
	}
}

func (s *CrawlerService) StartCrawl(ctx context.Context, site domain.Site) error {
	// seed the frontier with the starting url
	err := s.seedFrontier(ctx, site)
	if err != nil {
		return fmt.Errorf("error adding starting url to frontier: %w", err)
	}

	// start the worker pool
	err = s.runWorkerPool(ctx, site)
	if err != nil {
		return fmt.Errorf("starting working pool: %w", err)
	}

	s.logger.Info("crawl completed", slog.String("site_id", site.ID))
	return nil
}

func (s *CrawlerService) seedFrontier(ctx context.Context, site domain.Site) error {

	// create entry in frontier
	entry, err := s.enqueueURL(ctx, site.ID, site.StartURL.String(), 0, 100)
	if err != nil {
		return err
	}

	s.logger.Debug("frontier seeded",
		slog.String("site_id", site.ID),
		slog.String("url", entry.URL),
		slog.String("entry_id", entry.ID))

	return nil
}

func (s *CrawlerService) enqueueURL(ctx context.Context, siteID string, url string, depth int, priority int) (*domain.FrontierEntry, error) {
	entry, isNew, err := s.frontierStore.EnsureEntry(ctx, siteID, url, depth, priority)
	if err != nil {
		return nil, fmt.Errorf("failed to add to frontier: %w", err)
	}

	shouldPublish := isNew || entry.Status == domain.URLStatusFailed

	if shouldPublish {
		task := domain.CrawlTask{
			EntryID: entry.ID,
			URL:     entry.URL,
			SiteID:  siteID,
			Depth:   entry.Depth,
		}

		// publish task to queue
		err := s.taskQueue.PublishCrawlTask(ctx, task)
		if err != nil {
			return nil, fmt.Errorf("publish seed task: %w", err)
		}
	}
	return entry, nil
}

func (s *CrawlerService) runWorkerPool(ctx context.Context, site domain.Site) error {
	taskChan, err := s.taskQueue.ConsumeCrawlTask(ctx)
	if err != nil {
		return fmt.Errorf("consume crawl tasks: %w", err)
	}

	var wg sync.WaitGroup
	for i := 0; i < s.workerCount; i++ {
		wg.Add(1)
		go func(workerId int) {
			defer wg.Done()
			s.workerLoop(ctx, site, workerId, taskChan)
		}(i)
	}
	wg.Wait()
	return nil
}

func (s *CrawlerService) workerLoop(ctx context.Context, site domain.Site, id int, taskChan <-chan domain.CrawlTask) {
	ticker := time.NewTicker(500 * time.Millisecond)
	defer ticker.Stop()

	s.logger.Debug("worker started", slog.Int("worker_id", id))
	for {
		select {
		case task, ok := <-taskChan:
			if !ok {
				s.logger.Debug("worker finished: task channel closed", slog.Int("worker_id", id))
				return
			}
			s.processTask(ctx, site, task)

		case <-ticker.C:
			stats, err := s.frontierStore.Stats(ctx, site.ID)
			if err == nil && stats.QueueSize == 0 {
				s.logger.Debug("worker exiting: no pending work", slog.Int("worker_id", id))
				return
			}

		case <-ctx.Done():
			s.logger.Debug("worker cancelled: context closed", slog.Int("worker_id", id))
			return
		}
	}
}

func (s *CrawlerService) processTask(ctx context.Context, site domain.Site, task domain.CrawlTask) {
	// Mark URL as crawling
	err := s.frontierStore.MarkCrawling(ctx, task.EntryID)
	if err != nil {
		s.logger.Error("failed to mark url as crawling", slog.String("site_id", site.ID),
			slog.String("entry_id", task.EntryID),
			slog.String("url", task.URL),
			slog.String("err", err.Error()),
		)
		return
	}

	// Fetch the page
	response, err := s.httpClient.Fetch(ctx, task.URL)
	if err != nil {
		s.handleTaskFailure(ctx, task, fmt.Errorf("failed to fetch url: %s ", err.Error()))
		return
	}

	// Check if success response
	if response.StatusCode != http.StatusOK {
		s.handleTaskFailure(ctx, task, fmt.Errorf("HTTP %d", response.StatusCode))
		return
	}

	// Only process HTML
	if !strings.HasPrefix(response.ContentType, "text/html") {
		err = s.frontierStore.MarkCompleted(ctx, task.EntryID, time.Now())
		if err != nil {
			s.logger.Error("failed to mark url as completed",
				slog.String("site_id", site.ID),
				slog.String("url", task.URL),
				slog.String("err", err.Error()),
			)
		}
		return
	}

	// Parse the links we found
	baseURL, err := url.Parse(task.URL)
	if err != nil {
		fmt.Printf("\n \u2705 Visited: %s\n", task.URL)
		s.handleTaskFailure(ctx, task, fmt.Errorf("failed to parse url: %s", err.Error()))
		return
	}

	links, err := s.htmlParser.ExtractLinks(baseURL, string(response.Body))
	if err != nil {
		s.logger.Warn("parse failed", slog.String("url", task.URL),
			slog.String("err", err.Error()))

		err := s.frontierStore.MarkCompleted(ctx, task.EntryID, time.Now())
		if err != nil {
			s.logger.Error("failed to mark url as completed",
				slog.String("site_id", site.ID),
				slog.String("url", task.URL),
				slog.String("err", err.Error()))
		}
		return
	}

	printDiscoveredLinks(task.URL, links)

	for _, link := range links {
		err = s.submitDiscoveredURL(ctx, site, link, task.Depth+1)
		if err != nil {
			s.logger.Error("failed to submit discovered url",
				slog.String("link", link.String()),
				slog.String("error", err.Error()),
			)
		}
	}

	// Mark the current entry as completed
	err = s.frontierStore.MarkCompleted(ctx, task.EntryID, time.Now())
	if err != nil {
		s.logger.Error("failed to mark url as completed",
			slog.String("site_id", site.ID),
			slog.String("url", task.URL),
			slog.String("err", err.Error()))
	}
}

func (s *CrawlerService) handleTaskFailure(ctx context.Context, task domain.CrawlTask, err error) {

	s.logger.Error("task failed",
		slog.String("entry_id", task.EntryID),
		slog.String("url", task.URL),
		slog.String("err", err.Error()),
	)

	fmt.Printf("Error: %s -  %s\n", task.URL, err.Error())

	err = s.frontierStore.MarkFailed(ctx, task.EntryID, err)
	if err != nil {
		s.logger.Error("failed to mark url as failed",
			slog.String("site_id", task.EntryID),
			slog.String("url", task.URL),
			slog.String("err", err.Error()))
	}
}

func (s *CrawlerService) submitDiscoveredURL(ctx context.Context, site domain.Site, link *url.URL, depth int) error {

	// Enforce max depth
	if site.MaxDepth > 0 && depth > site.MaxDepth {
		s.logger.Debug("skipping discovered url, max depth reached", slog.String("url", link.String()))
		return nil
	}

	// Enforce one subdomain only
	if link.Host != site.AllowedHost {
		s.logger.Debug("skipping discovered url, host not allowed",
			slog.String("url", link.String()),
			slog.String("allowed host", site.AllowedHost),
			slog.String("discovered url host", link.Host))
		return nil
	}

	// Filter out non http schemes eg mailto:foo@bar.com
	if link.Scheme != "http" && link.Scheme != "https" {
		s.logger.Debug("skipping discovered url, scheme not allowed", slog.String("url", link.String()))
		return nil
	}

	// Add priority , shallow pages have higher priority
	priority := 100 - depth

	// Add to frontier for crawling
	entry, err := s.enqueueURL(ctx, site.ID, link.String(), depth, priority)
	if err != nil {
		return err
	}

	s.logger.Debug("discovered url added to frontier",
		slog.String("site_id", site.ID),
		slog.String("url", entry.URL),
		slog.String("entry_id", entry.ID))
	return nil
}

var printMu sync.Mutex

func printDiscoveredLinks(url string, links []*url.URL) {
	printMu.Lock()
	defer printMu.Unlock()

	fmt.Printf("✅ Visited: %s\n", url)
	fmt.Printf("Links found (%d):\n", len(links))
	for i, link := range links {
		fmt.Printf("%d -  %s\n", i+1, link.String())
	}
	fmt.Println()
}
