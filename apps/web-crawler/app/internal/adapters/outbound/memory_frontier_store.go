package outbound

import (
	"context"
	"fmt"
	"sync"
	"time"
	"web-crawler/app/internal/core/domain"

	"github.com/google/uuid"
)

type MemoryFrontier struct {
	mu                  sync.RWMutex
	entriesByID         map[string]*domain.FrontierEntry
	entriesBySiteAndURL map[string]map[string]*domain.FrontierEntry
}

func NewMemoryFrontier() *MemoryFrontier {
	return &MemoryFrontier{
		entriesBySiteAndURL: make(map[string]map[string]*domain.FrontierEntry),
		entriesByID:         make(map[string]*domain.FrontierEntry),
	}
}
func (m *MemoryFrontier) EnsureEntry(ctx context.Context, siteID string, url string, depth int, priority int) (*domain.FrontierEntry, bool, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	//Check if already exists
	siteMap, ok := m.entriesBySiteAndURL[siteID]
	if !ok {
		siteMap = make(map[string]*domain.FrontierEntry)
		m.entriesBySiteAndURL[siteID] = siteMap
	}

	entry, ok := siteMap[url]
	if ok {
		return entry, false, nil
	}

	//If it doesn't create it
	entry = &domain.FrontierEntry{
		ID:           uuid.New().String(),
		SiteID:       siteID,
		URL:          url,
		Status:       domain.URLStatusPending,
		Priority:     priority,
		Depth:        depth,
		DiscoveredAt: time.Now(),
	}

	siteMap[url] = entry
	m.entriesByID[entry.ID] = entry
	return entry, true, nil
}

func (m *MemoryFrontier) MarkCrawling(ctx context.Context, id string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	entry, ok := m.entriesByID[id]
	if !ok {
		return fmt.Errorf("entry not found for id %s", id)
	}
	entry.Status = domain.URLStatusCrawling
	return nil
}

func (m *MemoryFrontier) MarkCompleted(ctx context.Context, id string, visitedAt time.Time) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	entry, ok := m.entriesByID[id]
	if !ok {
		return fmt.Errorf("entry not found for id %s", id)
	}
	entry.Status = domain.URLStatusCompleted
	entry.VisitedAt = visitedAt
	entry.Error = ""
	return nil
}

func (m *MemoryFrontier) MarkFailed(ctx context.Context, id string, err error) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	entry, ok := m.entriesByID[id]
	if !ok {
		return fmt.Errorf("entry not found for id %s", id)
	}
	entry.Status = domain.URLStatusFailed
	entry.Error = err.Error()
	return nil

}

func (m *MemoryFrontier) Stats(ctx context.Context, siteID string) (*domain.FrontierStats, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	siteMap, ok := m.entriesBySiteAndURL[siteID]
	if !ok {
		return &domain.FrontierStats{
			SitedID:      siteID,
			QueueSize:    0,
			VisitedCount: 0,
			FailedCount:  0,
		}, nil
	}

	var queueSize, visited, failed int
	for _, entry := range siteMap {
		switch entry.Status {
		case domain.URLStatusPending, domain.URLStatusCrawling:
			queueSize++
		case domain.URLStatusCompleted:
			visited++
		case domain.URLStatusFailed:
			failed++
		}

	}
	return &domain.FrontierStats{
		SitedID:      siteID,
		QueueSize:    queueSize,
		VisitedCount: visited,
		FailedCount:  failed,
	}, nil

}

func (m *MemoryFrontier) GetByID(ctx context.Context, id string) (*domain.FrontierEntry, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	entry, ok := m.entriesByID[id]
	if !ok {
		return nil, fmt.Errorf("entry not found for id %s", id)
	}
	return entry, nil
}

func (m *MemoryFrontier) GetByURL(ctx context.Context, url string, siteID string) (*domain.FrontierEntry, error) {

	m.mu.RLock()
	defer m.mu.RUnlock()

	siteMap, ok := m.entriesBySiteAndURL[siteID]
	if !ok {
		return nil, fmt.Errorf("entry not found for id %s", siteID)
	}
	entry, ok := siteMap[url]
	if !ok {
		return nil, fmt.Errorf("entry not found for url %s", url)
	}
	return entry, nil

}
