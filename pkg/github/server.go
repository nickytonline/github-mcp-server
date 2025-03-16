package github

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"

	"github.com/google/go-github/v69/github"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

// NewServer creates a new GitHub MCP server with the specified GH client and logger.
func NewServer(client *github.Client) *server.MCPServer {
	// Create a new MCP server
	s := server.NewMCPServer(
		"github-mcp-server",
		"0.0.1",
		server.WithResourceCapabilities(true, true),
		server.WithLogging())

	// Add GitHub tools - Issues
	s.AddTool(getIssue(client))
	s.AddTool(addIssueComment(client))
	s.AddTool(searchIssues(client))

	// Add GitHub tools - Pull Requests
	s.AddTool(getPullRequest(client))
	s.AddTool(listPullRequests(client))
	s.AddTool(mergePullRequest(client))
	s.AddTool(getPullRequestFiles(client))
	s.AddTool(getPullRequestStatus(client))
	s.AddTool(updatePullRequestBranch(client))
	s.AddTool(getPullRequestComments(client))
	s.AddTool(getPullRequestReviews(client))

	// Add GitHub tools - Repositories
	s.AddTool(createOrUpdateFile(client))
	s.AddTool(searchRepositories(client))
	s.AddTool(createRepository(client))
	s.AddTool(getFileContents(client))
	s.AddTool(forkRepository(client))
	s.AddTool(createBranch(client))
	s.AddTool(listCommits(client))

	// Add GitHub tools - Search
	s.AddTool(searchCode(client))
	s.AddTool(searchUsers(client))

	// Add GitHub tools - Users
	s.AddTool(getMe(client))

	// Add GitHub tools - Code Scanning
	s.AddTool(getCodeScanningAlert(client))
	s.AddTool(listCodeScanningAlerts(client))

	return s
}

// getMe creates a tool to get details of the authenticated user.
func getMe(client *github.Client) (tool mcp.Tool, handler server.ToolHandlerFunc) {
	return mcp.NewTool("get_me",
			mcp.WithDescription("Get details of the authenticated user."),
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
