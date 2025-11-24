package testutils

import (
	"net/url"

	"github.com/stretchr/testify/mock"
)

type MockHTMLParser struct {
	mock.Mock
}

func (m *MockHTMLParser) ExtractLinks(base *url.URL, html string) ([]*url.URL, error) {
	args := m.Called(base, html)
	return args.Get(0).([]*url.URL), args.Error(1)
}
