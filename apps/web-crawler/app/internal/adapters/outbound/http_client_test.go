package outbound

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestStdHTTPClient_Fetch_Sucess(t *testing.T) {
	ctx := context.Background()
	client := NewStdHTTPClient(10 * time.Second)

	expectedBody := []byte("<html>ok</html>")
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write(expectedBody)
		w.Header().Set("Content-Type", "text/html")

	}))
	defer mockServer.Close()

	resp, err := client.Fetch(ctx, mockServer.URL)
	require.NoError(t, err)
	require.NotNil(t, resp)
	require.Equal(t, resp.StatusCode, 200)
	require.Equal(t, expectedBody, resp.Body)

}

func TestStdHTTPClient_Fetch_InvalidURL(t *testing.T) {
	ctx := context.Background()
	url := "://bad-url"
	client := NewStdHTTPClient(10 * time.Second)
	_, err := client.Fetch(ctx, url)
	require.Error(t, err)
}
