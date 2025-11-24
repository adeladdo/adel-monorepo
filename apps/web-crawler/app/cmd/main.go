package main

import (
	"context"
	"fmt"
	"log"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"
	"web-crawler/app/internal/adapters/outbound"
	"web-crawler/app/internal/config"
	"web-crawler/app/internal/core/domain"
	"web-crawler/app/internal/core/services"

	"github.com/google/uuid"
)

func main() {

	ctx := context.Background()

	ctx, stop := signal.NotifyContext(ctx, os.Interrupt, syscall.SIGTERM)
	defer stop()

	cfg, err := config.Load(ctx)
	if err != nil {
		log.Fatalf("Error loading config: %v", err)
	}

	if cfg.StartURL.Scheme != "http" && cfg.StartURL.Scheme != "https" {
		log.Fatalf("StartURL scheme must be http or https")
	}

	run(ctx, cfg)
}

func run(ctx context.Context, cfg *config.Config) {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))

	frontierStore := outbound.NewMemoryFrontier()
	taskQueue := outbound.NewChannelTaskQueue(cfg.TaskQueueBuffer)
	httpClient := outbound.NewStdHTTPClient(cfg.HTTPTimeout)
	htmlParser := outbound.NewGoqueryHTMLParser()

	crawler := services.NewCrawlerService(frontierStore, taskQueue, httpClient,
		htmlParser, logger, cfg.Workers)

	site := domain.Site{
		ID:          uuid.NewString(),
		StartURL:    cfg.StartURL,
		AllowedHost: cfg.StartURL.Host,
		MaxDepth:    cfg.MaxDepth,
	}

	fmt.Println("RUNNING WEB-CRAWLER-V1")
	fmt.Println("==============================================")
	fmt.Printf("URL:      %s\n", cfg.StartURL.String())
	fmt.Printf("Host:     %s\n", cfg.StartURL.Host)
	fmt.Printf("Depth:    %d\n", cfg.MaxDepth)
	fmt.Printf("Workers:  %d\n", cfg.Workers)
	fmt.Println("==============================================")

	startTime := time.Now()
	err := crawler.StartCrawl(ctx, site)
	if err != nil {
		logger.Error("crawl failed", slog.String("error", err.Error()))
		os.Exit(1)
	}

	duration := time.Since(startTime)

	stats, err := frontierStore.Stats(ctx, site.ID)
	if err == nil {
		fmt.Println("==============================================")
		fmt.Println(" WEB-CRAWLER STATS")
		fmt.Println("==============================================")
		fmt.Printf("Queued:      %d\n", stats.QueueSize)
		fmt.Printf("Failed:      %d\n", stats.FailedCount)
		fmt.Printf("Completed:   %d\n", stats.VisitedCount)
		fmt.Printf("Total time taken: %s\n", duration)
		fmt.Println("==============================================")

	}
}
