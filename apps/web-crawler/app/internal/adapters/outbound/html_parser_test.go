package outbound

import (
	"net/url"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestHTMLParserImpl_ExtractLinks(t *testing.T) {
	parser := NewGoqueryHTMLParser()

	base, err := url.Parse("https://crawlme.monzo.com/start")
	require.NoError(t, err)

	tests := []struct {
		name string
		html string
		want []string
	}{
		{
			name: "resolves relative links to absolute",
			html: `
				<html>
					<body>
						<a href="/page1">Page 1</a>
						<a href="page2">Page 2</a>
					</body>
				</html>`,
			want: []string{
				"https://crawlme.monzo.com/page1",
				"https://crawlme.monzo.com/page2",
			},
		},
		{
			name: "keeps absolute links and strips fragments",
			html: `
				<html>
					<body>
						<a href="https://crawlme.monzo.com/page#section">With fragment</a>
						<a href="https://facebook.com/ext#foo">External</a>
					</body>
				</html>`,
			want: []string{
				"https://crawlme.monzo.com/page",
				"https://facebook.com/ext",
			},
		},
		{
			name: "ignores invalid and empty hrefs",
			html: `
				<html>
					<body>
						<a>no href</a>
						<a href="">empty href</a>
						<a href="://bad-url">invalid</a>
						<a href="/ok">OK</a>
					</body>
				</html>`,
			want: []string{
				"https://crawlme.monzo.com/ok",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotURLs, err := parser.ExtractLinks(base, tt.html)
			require.NoError(t, err)

			got := make([]string, 0, len(gotURLs))
			for _, u := range gotURLs {
				got = append(got, u.String())
			}

			require.Equal(t, tt.want, got)
		})
	}
}
