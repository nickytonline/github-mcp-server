package github

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/google/go-github/v72/github"
	"github.com/mark3labs/mcp-go/mcp"
)

func searchHandler(
	ctx context.Context,
	getClient GetClientFn,
	request mcp.CallToolRequest,
	searchType string,
	errorPrefix string,
) (*mcp.CallToolResult, error) {
	query, err := RequiredParam[string](request, "query")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	query = fmt.Sprintf("is:%s %s", searchType, query)

	sort, err := OptionalParam[string](request, "sort")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	order, err := OptionalParam[string](request, "order")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	pagination, err := OptionalPaginationParams(request)
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	opts := &github.SearchOptions{
		Sort:  sort,
		Order: order,
		ListOptions: github.ListOptions{
			Page:    pagination.page,
			PerPage: pagination.perPage,
		},
	}

	client, err := getClient(ctx)
	if err != nil {
		return nil, fmt.Errorf("%s: failed to get GitHub client: %w", errorPrefix, err)
	}
	result, resp, err := client.Search.Issues(ctx, query, opts)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", errorPrefix, err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		body, err := io.ReadAll(resp.Body)
		if err != nil {
			return nil, fmt.Errorf("%s: failed to read response body: %w", errorPrefix, err)
		}
		return mcp.NewToolResultError(fmt.Sprintf("%s: %s", errorPrefix, string(body))), nil
	}

	r, err := json.Marshal(result)
	if err != nil {
		return nil, fmt.Errorf("%s: failed to marshal response: %w", errorPrefix, err)
	}

	return mcp.NewToolResultText(string(r)), nil
}
