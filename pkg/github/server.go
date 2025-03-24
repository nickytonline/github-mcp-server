package github

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/github/github-mcp-server/pkg/translations"
	"github.com/google/go-github/v69/github"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

// NewServer creates a new GitHub MCP server with the specified GH client and logger.
func NewServer(client *github.Client, readOnly bool, t translations.TranslationHelperFunc) *server.MCPServer {
	// Create a new MCP server
	s := server.NewMCPServer(
		"github-mcp-server",
		"0.0.1",
		server.WithResourceCapabilities(true, true),
		server.WithLogging())

	// Add GitHub Resources
	defaultTemplate, branchTemplate, tagTemplate, shaTemplate, prTemplate, handler := getRepositoryContent(client, t)

	s.AddResourceTemplate(defaultTemplate, handler)
	s.AddResourceTemplate(branchTemplate, handler)
	s.AddResourceTemplate(tagTemplate, handler)
	s.AddResourceTemplate(shaTemplate, handler)
	s.AddResourceTemplate(prTemplate, handler)

	// Add GitHub tools - Issues
	s.AddTool(getIssue(client, t))
	s.AddTool(searchIssues(client, t))
	s.AddTool(listIssues(client, t))
	if !readOnly {
		s.AddTool(createIssue(client, t))
		s.AddTool(addIssueComment(client, t))
		s.AddTool(createIssue(client, t))
		s.AddTool(updateIssue(client, t))
	}

	// Add GitHub tools - Pull Requests
	s.AddTool(getPullRequest(client, t))
	s.AddTool(listPullRequests(client, t))
	s.AddTool(getPullRequestFiles(client, t))
	s.AddTool(getPullRequestStatus(client, t))
	s.AddTool(getPullRequestComments(client, t))
	s.AddTool(getPullRequestReviews(client, t))
	if !readOnly {
		s.AddTool(mergePullRequest(client, t))
		s.AddTool(updatePullRequestBranch(client, t))
		s.AddTool(createPullRequestReview(client, t))
	}

	// Add GitHub tools - Repositories
	s.AddTool(searchRepositories(client, t))
	s.AddTool(getFileContents(client, t))
	s.AddTool(listCommits(client, t))
	if !readOnly {
		s.AddTool(createOrUpdateFile(client, t))
		s.AddTool(createRepository(client, t))
		s.AddTool(forkRepository(client, t))
		s.AddTool(createBranch(client, t))
	}

	// Add GitHub tools - Search
	s.AddTool(searchCode(client, t))
	s.AddTool(searchUsers(client, t))

	// Add GitHub tools - Users
	s.AddTool(getMe(client, t))

	// Add GitHub tools - Code Scanning
	s.AddTool(getCodeScanningAlert(client, t))
	s.AddTool(listCodeScanningAlerts(client, t))
	return s
}

// getMe creates a tool to get details of the authenticated user.
func getMe(client *github.Client, t translations.TranslationHelperFunc) (tool mcp.Tool, handler server.ToolHandlerFunc) {
	return mcp.NewTool("get_me",
			mcp.WithDescription(t("TOOL_GET_ME_DESCRIPTION", "Get details of the authenticated GitHub user")),
			mcp.WithString("reason",
				mcp.Description("Optional: reason the session was created"),
			),
		),
		func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
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

// parseCommaSeparatedList is a helper function that parses a comma-separated list of strings from the input string.
func parseCommaSeparatedList(input string) []string {
	if input == "" {
		return nil
	}

	parts := strings.Split(input, ",")
	result := make([]string, 0, len(parts))

	for _, part := range parts {
		trimmed := strings.TrimSpace(part)
		if trimmed != "" {
			result = append(result, trimmed)
		}
	}

	return result
}
