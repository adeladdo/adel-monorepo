package outbound

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"time"
	"web-crawler/app/internal/core/ports/outbound"
)

type StdHTTPClient struct {
	client *http.Client
}

func NewStdHTTPClient(timeout time.Duration) *StdHTTPClient {
	return &StdHTTPClient{
		client: &http.Client{
			Timeout: timeout,
			CheckRedirect: func(req *http.Request, via []*http.Request) error {
				if len(via) >= 10 {
					return errors.New("stopped after 10 redirects")
				}
				return nil
			},
		},
	}
}

func (c *StdHTTPClient) Fetch(ctx context.Context, url string) (outbound.HttpResponse, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return outbound.HttpResponse{}, fmt.Errorf("create request: %w", err)
	}

	resp, err := c.client.Do(req)
	if err != nil {
		return outbound.HttpResponse{}, fmt.Errorf("http request failed: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return outbound.HttpResponse{}, fmt.Errorf("failed to read response body: %w", err)
	}

	return outbound.HttpResponse{
		StatusCode:  resp.StatusCode,
		Body:        body,
		ContentType: resp.Header.Get("Content-Type"),
	}, nil

}
