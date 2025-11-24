package outbound

import "context"

type HttpClient interface {
	Fetch(ctx context.Context, url string) (HttpResponse, error)
}

type HttpResponse struct {
	StatusCode  int
	Body        []byte
	ContentType string
}
