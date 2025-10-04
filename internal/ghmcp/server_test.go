package ghmcp

import (
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
	"sync"
	"testing"

	githubv4 "github.com/shurcooL/githubv4"
	"github.com/stretchr/testify/require"
)

func TestGitHubClientFactory_UsesContextTokenAndUserAgent(t *testing.T) {
	const (
		version   = "test-version"
		token     = "token-123"
		defaultUA = "github-mcp-http/" + version
		customUA  = "custom-agent/1.0"
	)

	type captured struct {
		method string
		path   string
		header http.Header
	}

	var (
		mu       sync.Mutex
		requests []captured
	)

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		requests = append(requests, captured{
			method: r.Method,
			path:   r.URL.Path,
			header: r.Header.Clone(),
		})
		mu.Unlock()

		switch r.URL.Path {
		case "/user":
			w.WriteHeader(http.StatusOK)
		case "/graphql":
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"data":{"viewer":{"login":"octocat"}}}`))
		default:
			w.WriteHeader(http.StatusOK)
		}
	})

	srv := httptest.NewServer(handler)
	defer srv.Close()

	mustParse := func(raw string) *url.URL {
		parsed, err := url.Parse(raw)
		require.NoError(t, err)
		return parsed
	}

	apiHost := apiHost{
		baseRESTURL: mustParse(srv.URL + "/"),
		graphqlURL:  mustParse(srv.URL + "/graphql"),
		uploadURL:   mustParse(srv.URL + "/upload"),
		rawURL:      mustParse(srv.URL + "/raw/"),
	}

	factory := newGitHubClientFactory(version, apiHost, TokenFromContext)

	ctx := ContextWithToken(context.Background(), token)

	restClient, err := factory.getRESTClient(ctx)
	require.NoError(t, err)

	req, err := restClient.NewRequest(http.MethodGet, "user", nil)
	require.NoError(t, err)
	resp, err := restClient.Do(ctx, req, nil)
	require.NoError(t, err)
	require.NoError(t, resp.Body.Close())

	rawClient, err := factory.getRawClient(ctx)
	require.NoError(t, err)
	rawResp, err := rawClient.GetRawContent(ctx, "octocat", "hello-world", "README.md", nil)
	require.NoError(t, err)
	require.NoError(t, rawResp.Body.Close())

	factory.setUserAgent(customUA)

	gqlClient, err := factory.getGraphQLClient(ctx)
	require.NoError(t, err)
	var query struct {
		Viewer struct {
			Login githubv4.String
		}
	}
	err = gqlClient.Query(ctx, &query, nil)
	require.NoError(t, err)
	require.Equal(t, githubv4.String("octocat"), query.Viewer.Login)

	mu.Lock()
	defer mu.Unlock()
	require.Len(t, requests, 3)

	restReq := requests[0]
	require.Equal(t, "/user", restReq.path)
	require.Equal(t, "Bearer "+token, restReq.header.Get("Authorization"))
	require.Equal(t, defaultUA, restReq.header.Get("User-Agent"))

	rawReq := requests[1]
	require.Contains(t, rawReq.path, "/raw/")
	require.Equal(t, "Bearer "+token, rawReq.header.Get("Authorization"))
	require.Equal(t, defaultUA, rawReq.header.Get("User-Agent"))

	gqlReq := requests[2]
	require.Equal(t, "/graphql", gqlReq.path)
	require.Equal(t, "Bearer "+token, gqlReq.header.Get("Authorization"))
	require.Equal(t, customUA, gqlReq.header.Get("User-Agent"))
}
