package outbound

import "net/url"

type HTMLParser interface {
	ExtractLinks(base *url.URL, html string) ([]*url.URL, error)
}
