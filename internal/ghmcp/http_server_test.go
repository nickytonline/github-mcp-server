package ghmcp

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestTokenMiddlewareAddsTokenToContext(t *testing.T) {
	const token = "abc123"

	handler := tokenMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		extracted, err := TokenFromContext(r.Context())
		require.NoError(t, err)
		require.Equal(t, token, extracted)
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/mcp", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	require.Equal(t, http.StatusOK, rr.Code)
}

func TestTokenMiddlewareRejectsInvalidAuthorization(t *testing.T) {
	cases := map[string]string{
		"missing header": "",
		"wrong scheme":   "Basic abc123",
		"missing token":  "Bearer    ",
		"extra spaces":   "Bearer",
	}

	for name, header := range cases {
		t.Run(name, func(t *testing.T) {
			handler := tokenMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				t.Fatalf("handler should not be called")
			}))

			req := httptest.NewRequest(http.MethodPost, "/mcp", nil)
			if header != "" {
				req.Header.Set("Authorization", header)
			}
			rr := httptest.NewRecorder()

			handler.ServeHTTP(rr, req)
			require.Equal(t, http.StatusUnauthorized, rr.Code)
		})
	}
}

func TestTokenContextHelpers(t *testing.T) {
	ctx := ContextWithToken(nil, "  token-value  ")
	token, err := TokenFromContext(ctx)
	require.NoError(t, err)
	require.Equal(t, "token-value", token)

	_, err = TokenFromContext(nil)
	require.Error(t, err)
}
