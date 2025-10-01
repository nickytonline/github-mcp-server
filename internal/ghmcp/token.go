package ghmcp

import (
	"context"
	"fmt"
	"strings"
)

type tokenContextKey struct{}

// ContextWithToken stores the provided token in the context.
func ContextWithToken(ctx context.Context, token string) context.Context {
	trimmed := strings.TrimSpace(token)
	if ctx == nil {
		ctx = context.Background()
	}
	return context.WithValue(ctx, tokenContextKey{}, trimmed)
}

// TokenFromContext retrieves an authentication token from the context.
func TokenFromContext(ctx context.Context) (string, error) {
	if ctx == nil {
		return "", fmt.Errorf("context is nil")
	}
	if token, ok := ctx.Value(tokenContextKey{}).(string); ok {
		trimmed := strings.TrimSpace(token)
		if trimmed != "" {
			return trimmed, nil
		}
	}
	return "", fmt.Errorf("missing authentication token")
}
