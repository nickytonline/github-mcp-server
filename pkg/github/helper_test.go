package github

import (
	"encoding/json"
	"net/http"
	"testing"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// expectQueryParams is a helper function to create a partial mock that expects a
// request with the given query parameters, with the ability to chain a response handler.
func expectQueryParams(t *testing.T, expectedQueryParams map[string]string) *partialMock {
	return &partialMock{
		t:                   t,
		expectedQueryParams: expectedQueryParams,
	}
}

// expectRequestBody is a helper function to create a partial mock that expects a
// request with the given body, with the ability to chain a response handler.
func expectRequestBody(t *testing.T, expectedRequestBody any) *partialMock {
	return &partialMock{
		t:                   t,
		expectedRequestBody: expectedRequestBody,
	}
}

type partialMock struct {
	t                   *testing.T
	expectedQueryParams map[string]string
	expectedRequestBody any
}

func (p *partialMock) andThen(responseHandler http.HandlerFunc) http.HandlerFunc {
	p.t.Helper()
	return func(w http.ResponseWriter, r *http.Request) {
		if p.expectedRequestBody != nil {
			var unmarshaledRequestBody any
			err := json.NewDecoder(r.Body).Decode(&unmarshaledRequestBody)
			require.NoError(p.t, err)

			require.Equal(p.t, p.expectedRequestBody, unmarshaledRequestBody)
		}

		if p.expectedQueryParams != nil {
			require.Equal(p.t, len(p.expectedQueryParams), len(r.URL.Query()))
			for k, v := range p.expectedQueryParams {
				require.Equal(p.t, v, r.URL.Query().Get(k))
			}
		}

		responseHandler(w, r)
	}
}

// mockResponse is a helper function to create a mock HTTP response handler
// that returns a specified status code and marshaled body.
func mockResponse(t *testing.T, code int, body interface{}) http.HandlerFunc {
	t.Helper()
	return func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(code)
		b, err := json.Marshal(body)
		require.NoError(t, err)
		_, _ = w.Write(b)
	}
}

// createMCPRequest is a helper function to create a MCP request with the given arguments.
func createMCPRequest(args map[string]interface{}) mcp.CallToolRequest {
	return mcp.CallToolRequest{
		Params: struct {
			Name      string                 `json:"name"`
			Arguments map[string]interface{} `json:"arguments,omitempty"`
			Meta      *struct {
				ProgressToken mcp.ProgressToken `json:"progressToken,omitempty"`
			} `json:"_meta,omitempty"`
		}{
			Arguments: args,
		},
	}
}

// getTextResult is a helper function that returns a text result from a tool call.
func getTextResult(t *testing.T, result *mcp.CallToolResult) mcp.TextContent {
	t.Helper()
	assert.NotNil(t, result)
	require.Len(t, result.Content, 1)
	require.IsType(t, mcp.TextContent{}, result.Content[0])
	textContent := result.Content[0].(mcp.TextContent)
	assert.Equal(t, "text", textContent.Type)
	return textContent
}
