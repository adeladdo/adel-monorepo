package domain

import (
	"net/url"
	"time"
)

type Site struct {
	ID          string   `json:"id"`
	StartURL    *url.URL `json:"start_url"`
	AllowedHost string   `json:"allowed_host"`
	MaxDepth    int      `json:"max_depth"`
}

type FrontierEntry struct {
	ID           string    `json:"id"`
	SiteID       string    `json:"site_id"`
	URL          string    `json:"url"`
	ParentURL    string    `json:"parent_url"`
	Status       URLStatus `json:"status"`
	Priority     int       `json:"priority"`
	Depth        int       `json:"depth"`
	DiscoveredAt time.Time `json:"discovered_at"`
	VisitedAt    time.Time `json:"visited_at"`
	Error        string    `json:"error"`
}

type FrontierStats struct {
	SitedID      string `json:"sited_id"`
	QueueSize    int    `json:"queue_size"`
	VisitedCount int    `json:"visited_count"`
	FailedCount  int    `json:"failed_count"`
}

type URLStatus string

const (
	URLStatusPending   URLStatus = "pending"
	URLStatusCrawling  URLStatus = "crawling"
	URLStatusCompleted URLStatus = "completed"
	URLStatusFailed    URLStatus = "failed"
)

type CrawlTask struct {
	EntryID string `json:"id"`
	URL     string `json:"url"`
	Depth   int    `json:"depth"`
	SiteID  string `json:"site_id"`
}

type PageResult struct {
	URL         string        `json:"url"`
	Depth       int           `json:"depth"`
	Success     bool          `json:"success"`
	Links       []string      `json:"links"`
	StartedAt   time.Time     `json:"started_at"`
	CompletedAt time.Time     `json:"completed_at"`
	Duration    time.Duration `json:"duration"`
}
