package outbound

import (
	"net/url"
	"strings"

	"github.com/PuerkitoBio/goquery"
)

type GoqueryHTMLParser struct {
}

func NewGoqueryHTMLParser() *GoqueryHTMLParser {
	return &GoqueryHTMLParser{}
}

func (p *GoqueryHTMLParser) ExtractLinks(base *url.URL, html string) ([]*url.URL, error) {

	doc, err := goquery.NewDocumentFromReader(strings.NewReader(html))
	if err != nil {
		return nil, err
	}
	var links []*url.URL

	doc.Find("a[href]").Each(func(i int, s *goquery.Selection) {
		href, ok := s.Attr("href")
		if !ok || href == "" {
			return
		}
		u, err := base.Parse(href)
		if err != nil {
			return
		}

		u.Fragment = ""
		links = append(links, u)
	})
	return links, nil
}
