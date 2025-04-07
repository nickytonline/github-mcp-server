package github

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"

	"github.com/github/github-mcp-server/pkg/translations"
	"github.com/google/go-github/v69/github"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

// NewServer creates a new GitHub MCP server with the specified GH client and logger.
func NewServer(client *github.Client, version string, readOnly bool, t translations.TranslationHelperFunc) *server.MCPServer {
	// Create a new MCP server
	s := server.NewMCPServer(
		"github-mcp-server",
		version,
		server.WithResourceCapabilities(true, true),
		server.WithLogging())

	// Add GitHub Resources
	s.AddResourceTemplate(GetRepositoryResourceContent(client, t))
	s.AddResourceTemplate(GetRepositoryResourceBranchContent(client, t))
	s.AddResourceTemplate(GetRepositoryResourceCommitContent(client, t))
	s.AddResourceTemplate(GetRepositoryResourceTagContent(client, t))
	s.AddResourceTemplate(GetRepositoryResourcePrContent(client, t))

	// Add GitHub tools - Issues
	s.AddTool(GetIssue(client, t))
	s.AddTool(SearchIssues(client, t))
	s.AddTool(ListIssues(client, t))
	s.AddTool(GetIssueComments(client, t))
	if !readOnly {
		s.AddTool(CreateIssue(client, t))
		s.AddTool(AddIssueComment(client, t))
		s.AddTool(UpdateIssue(client, t))
	}

	// Add GitHub tools - Pull Requests
	s.AddTool(GetPullRequest(client, t))
	s.AddTool(ListPullRequests(client, t))
	s.AddTool(GetPullRequestFiles(client, t))
	s.AddTool(GetPullRequestStatus(client, t))
	s.AddTool(GetPullRequestComments(client, t))
	s.AddTool(GetPullRequestReviews(client, t))
	if !readOnly {
		s.AddTool(MergePullRequest(client, t))
		s.AddTool(UpdatePullRequestBranch(client, t))
		s.AddTool(CreatePullRequestReview(client, t))
		s.AddTool(CreatePullRequest(client, t))
	}

	// Add GitHub tools - Repositories
	s.AddTool(searchRepositories(client, t))
	s.AddTool(GetFileContents(client, t))
	s.AddTool(ListCommits(client, t))
	if !readOnly {
		s.AddTool(CreateOrUpdateFile(client, t))
		s.AddTool(CreateRepository(client, t))
		s.AddTool(ForkRepository(client, t))
		s.AddTool(CreateBranch(client, t))
		s.AddTool(PushFiles(client, t))
	}

	// Add GitHub tools - Search
	s.AddTool(searchCode(client, t))
	s.AddTool(searchUsers(client, t))

	// Add GitHub tools - Users
	s.AddTool(getMe(client, t))

	// Add GitHub tools - Code Scanning
	s.AddTool(GetCodeScanningAlert(client, t))
	s.AddTool(ListCodeScanningAlerts(client, t))
	return s
}

// getMe creates a tool to get details of the authenticated user.
func getMe(client *github.Client, t translations.TranslationHelperFunc) (tool mcp.Tool, handler server.ToolHandlerFunc) {
	return mcp.NewTool("get_me",
			mcp.WithDescription(t("TOOL_GET_ME_DESCRIPTION", "Get details of the authenticated GitHub user. Use this when a request include \"me\", \"my\"...")),
			mcp.WithString("reason",
				mcp.Description("Optional: reason the session was created"),
			),
		),
		func(ctx context.Context, _ mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			user, resp, err := client.Users.Get(ctx, "")
			if err != nil {
				return nil, fmt.Errorf("failed to get user: %w", err)
			}
			defer func() { _ = resp.Body.Close() }()

			if resp.StatusCode != http.StatusOK {
				body, err := io.ReadAll(resp.Body)
				if err != nil {
					return nil, fmt.Errorf("failed to read response body: %w", err)
				}
				return mcp.NewToolResultError(fmt.Sprintf("failed to get user: %s", string(body))), nil
			}

			r, err := json.Marshal(user)
			if err != nil {
				return nil, fmt.Errorf("failed to marshal user: %w", err)
			}

			return mcp.NewToolResultText(string(r)), nil
		}
}

// isAcceptedError checks if the error is an accepted error.
func isAcceptedError(err error) bool {
	var acceptedError *github.AcceptedError
	return errors.As(err, &acceptedError)
}

// requiredParam is a helper function that can be used to fetch a requested parameter from the request.
// It does the following checks:
// 1. Checks if the parameter is present in the request.
// 2. Checks if the parameter is of the expected type.
// 3. Checks if the parameter is not empty, i.e: non-zero value
func requiredParam[T comparable](r mcp.CallToolRequest, p string) (T, error) {
	var zero T

	// Check if the parameter is present in the request
	if _, ok := r.Params.Arguments[p]; !ok {
		return zero, fmt.Errorf("missing required parameter: %s", p)
	}

	// Check if the parameter is of the expected type
	if _, ok := r.Params.Arguments[p].(T); !ok {
		return zero, fmt.Errorf("parameter %s is not of type %T", p, zero)
	}

	if r.Params.Arguments[p].(T) == zero {
		return zero, fmt.Errorf("missing required parameter: %s", p)

	}

	return r.Params.Arguments[p].(T), nil
}

// requiredInt is a helper function that can be used to fetch a requested parameter from the request.
// It does the following checks:
// 1. Checks if the parameter is present in the request.
// 2. Checks if the parameter is of the expected type.
// 3. Checks if the parameter is not empty, i.e: non-zero value
func requiredInt(r mcp.CallToolRequest, p string) (int, error) {
	v, err := requiredParam[float64](r, p)
	if err != nil {
		return 0, err
	}
	return int(v), nil
}

// optionalParam is a helper function that can be used to fetch a requested parameter from the request.
// It does the following checks:
// 1. Checks if the parameter is present in the request, if not, it returns its zero-value
// 2. If it is present, it checks if the parameter is of the expected type and returns it
func optionalParam[T any](r mcp.CallToolRequest, p string) (T, error) {
	var zero T

	// Check if the parameter is present in the request
	if _, ok := r.Params.Arguments[p]; !ok {
		return zero, nil
	}

	// Check if the parameter is of the expected type
	if _, ok := r.Params.Arguments[p].(T); !ok {
		return zero, fmt.Errorf("parameter %s is not of type %T, is %T", p, zero, r.Params.Arguments[p])
	}

	return r.Params.Arguments[p].(T), nil
}

// optionalIntParam is a helper function that can be used to fetch a requested parameter from the request.
// It does the following checks:
// 1. Checks if the parameter is present in the request, if not, it returns its zero-value
// 2. If it is present, it checks if the parameter is of the expected type and returns it
func optionalIntParam(r mcp.CallToolRequest, p string) (int, error) {
	v, err := optionalParam[float64](r, p)
	if err != nil {
		return 0, err
	}
	return int(v), nil
}

// optionalIntParamWithDefault is a helper function that can be used to fetch a requested parameter from the request
// similar to optionalIntParam, but it also takes a default value.
func optionalIntParamWithDefault(r mcp.CallToolRequest, p string, d int) (int, error) {
	v, err := optionalIntParam(r, p)
	if err != nil {
		return 0, err
	}
	if v == 0 {
		return d, nil
	}
	return v, nil
}

// optionalStringArrayParam is a helper function that can be used to fetch a requested parameter from the request.
// It does the following checks:
// 1. Checks if the parameter is present in the request, if not, it returns its zero-value
// 2. If it is present, iterates the elements and checks each is a string
func optionalStringArrayParam(r mcp.CallToolRequest, p string) ([]string, error) {
	// Check if the parameter is present in the request
	if _, ok := r.Params.Arguments[p]; !ok {
		return []string{}, nil
	}

	switch v := r.Params.Arguments[p].(type) {
	case []string:
		return v, nil
	case []any:
		strSlice := make([]string, len(v))
		for i, v := range v {
			s, ok := v.(string)
			if !ok {
				return []string{}, fmt.Errorf("parameter %s is not of type string, is %T", p, v)
			}
			strSlice[i] = s
		}
		return strSlice, nil
	default:
		return []string{}, fmt.Errorf("parameter %s could not be coerced to []string, is %T", p, r.Params.Arguments[p])
	}
}

// withPagination returns a ToolOption that adds "page" and "perPage" parameters to the tool.
// The "page" parameter is optional, min 1. The "perPage" parameter is optional, min 1, max 100.
func withPagination() mcp.ToolOption {
	return func(tool *mcp.Tool) {
		mcp.WithNumber("page",
			mcp.Description("Page number for pagination (min 1)"),
			mcp.Min(1),
		)(tool)

		mcp.WithNumber("perPage",
			mcp.Description("Results per page for pagination (min 1, max 100)"),
			mcp.Min(1),
			mcp.Max(100),
		)(tool)
	}
}

type paginationParams struct {
	page    int
	perPage int
}

// optionalPaginationParams returns the "page" and "perPage" parameters from the request,
// or their default values if not present, "page" default is 1, "perPage" default is 30.
// In future, we may want to make the default values configurable, or even have this
// function returned from `withPagination`, where the defaults are provided alongside
// the min/max values.
func optionalPaginationParams(r mcp.CallToolRequest) (paginationParams, error) {
	page, err := optionalIntParamWithDefault(r, "page", 1)
	if err != nil {
		return paginationParams{}, err
	}
	perPage, err := optionalIntParamWithDefault(r, "perPage", 30)
	if err != nil {
		return paginationParams{}, err
	}
	return paginationParams{
		page:    page,
		perPage: perPage,
	}, nil
}
