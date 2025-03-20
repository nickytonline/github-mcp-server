package github

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/google/go-github/v69/github"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

// getIssue creates a tool to get details of a specific issue in a GitHub repository.
func getIssue(client *github.Client) (tool mcp.Tool, handler server.ToolHandlerFunc) {
	return mcp.NewTool("get_issue",
			mcp.WithDescription("Get details of a specific issue in a GitHub repository."),
			mcp.WithString("owner",
				mcp.Required(),
				mcp.Description("The owner of the repository."),
			),
			mcp.WithString("repo",
				mcp.Required(),
				mcp.Description("The name of the repository."),
			),
			mcp.WithNumber("issue_number",
				mcp.Required(),
				mcp.Description("The number of the issue."),
			),
		),
		func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			owner := request.Params.Arguments["owner"].(string)
			repo := request.Params.Arguments["repo"].(string)
			issueNumber := int(request.Params.Arguments["issue_number"].(float64))

			issue, resp, err := client.Issues.Get(ctx, owner, repo, issueNumber)
			if err != nil {
				return nil, fmt.Errorf("failed to get issue: %w", err)
			}
			defer func() { _ = resp.Body.Close() }()

			if resp.StatusCode != http.StatusOK {
				body, err := io.ReadAll(resp.Body)
				if err != nil {
					return nil, fmt.Errorf("failed to read response body: %w", err)
				}
				return mcp.NewToolResultError(fmt.Sprintf("failed to get issue: %s", string(body))), nil
			}

			r, err := json.Marshal(issue)
			if err != nil {
				return nil, fmt.Errorf("failed to marshal issue: %w", err)
			}

			return mcp.NewToolResultText(string(r)), nil
		}
}

// addIssueComment creates a tool to add a comment to an issue.
func addIssueComment(client *github.Client) (tool mcp.Tool, handler server.ToolHandlerFunc) {
	return mcp.NewTool("add_issue_comment",
			mcp.WithDescription("Add a comment to an existing issue"),
			mcp.WithString("owner",
				mcp.Required(),
				mcp.Description("Repository owner"),
			),
			mcp.WithString("repo",
				mcp.Required(),
				mcp.Description("Repository name"),
			),
			mcp.WithNumber("issue_number",
				mcp.Required(),
				mcp.Description("Issue number to comment on"),
			),
			mcp.WithString("body",
				mcp.Required(),
				mcp.Description("Comment text"),
			),
		),
		func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			owner := request.Params.Arguments["owner"].(string)
			repo := request.Params.Arguments["repo"].(string)
			issueNumber := int(request.Params.Arguments["issue_number"].(float64))
			body := request.Params.Arguments["body"].(string)

			comment := &github.IssueComment{
				Body: github.Ptr(body),
			}

			createdComment, resp, err := client.Issues.CreateComment(ctx, owner, repo, issueNumber, comment)
			if err != nil {
				return nil, fmt.Errorf("failed to create comment: %w", err)
			}
			defer func() { _ = resp.Body.Close() }()

			if resp.StatusCode != http.StatusCreated {
				body, err := io.ReadAll(resp.Body)
				if err != nil {
					return nil, fmt.Errorf("failed to read response body: %w", err)
				}
				return mcp.NewToolResultError(fmt.Sprintf("failed to create comment: %s", string(body))), nil
			}

			r, err := json.Marshal(createdComment)
			if err != nil {
				return nil, fmt.Errorf("failed to marshal response: %w", err)
			}

			return mcp.NewToolResultText(string(r)), nil
		}
}

// searchIssues creates a tool to search for issues and pull requests.
func searchIssues(client *github.Client) (tool mcp.Tool, handler server.ToolHandlerFunc) {
	return mcp.NewTool("search_issues",
			mcp.WithDescription("Search for issues and pull requests across GitHub repositories"),
			mcp.WithString("q",
				mcp.Required(),
				mcp.Description("Search query using GitHub issues search syntax"),
			),
			mcp.WithString("sort",
				mcp.Description("Sort field (comments, reactions, created, etc.)"),
			),
			mcp.WithString("order",
				mcp.Description("Sort order ('asc' or 'desc')"),
			),
			mcp.WithNumber("per_page",
				mcp.Description("Results per page (max 100)"),
			),
			mcp.WithNumber("page",
				mcp.Description("Page number"),
			),
		),
		func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			query := request.Params.Arguments["q"].(string)
			sort := ""
			if s, ok := request.Params.Arguments["sort"].(string); ok {
				sort = s
			}
			order := ""
			if o, ok := request.Params.Arguments["order"].(string); ok {
				order = o
			}
			perPage := 30
			if pp, ok := request.Params.Arguments["per_page"].(float64); ok {
				perPage = int(pp)
			}
			page := 1
			if p, ok := request.Params.Arguments["page"].(float64); ok {
				page = int(p)
			}

			opts := &github.SearchOptions{
				Sort:  sort,
				Order: order,
				ListOptions: github.ListOptions{
					PerPage: perPage,
					Page:    page,
				},
			}

			result, resp, err := client.Search.Issues(ctx, query, opts)
			if err != nil {
				return nil, fmt.Errorf("failed to search issues: %w", err)
			}
			defer func() { _ = resp.Body.Close() }()

			if resp.StatusCode != http.StatusOK {
				body, err := io.ReadAll(resp.Body)
				if err != nil {
					return nil, fmt.Errorf("failed to read response body: %w", err)
				}
				return mcp.NewToolResultError(fmt.Sprintf("failed to search issues: %s", string(body))), nil
			}

			r, err := json.Marshal(result)
			if err != nil {
				return nil, fmt.Errorf("failed to marshal response: %w", err)
			}

			return mcp.NewToolResultText(string(r)), nil
		}
}

// createIssue creates a tool to create a new issue in a GitHub repository.
func createIssue(client *github.Client) (tool mcp.Tool, handler server.ToolHandlerFunc) {
	return mcp.NewTool("create_issue",
			mcp.WithDescription("Create a new issue in a GitHub repository"),
			mcp.WithString("owner",
				mcp.Required(),
				mcp.Description("Repository owner"),
			),
			mcp.WithString("repo",
				mcp.Required(),
				mcp.Description("Repository name"),
			),
			mcp.WithString("title",
				mcp.Required(),
				mcp.Description("Issue title"),
			),
			mcp.WithString("body",
				mcp.Description("Issue body content"),
			),
			mcp.WithString("assignees",
				mcp.Description("Comma-separate list of usernames to assign to this issue"),
			),
			mcp.WithString("labels",
				mcp.Description("Comma-separate list of labels to apply to this issue"),
			),
		),
		func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			owner := request.Params.Arguments["owner"].(string)
			repo := request.Params.Arguments["repo"].(string)
			title := request.Params.Arguments["title"].(string)

			// Optional parameters
			var body string
			if b, ok := request.Params.Arguments["body"].(string); ok {
				body = b
			}

			// Parse assignees if present
			assignees := []string{} // default to empty slice, can't be nil
			if a, ok := request.Params.Arguments["assignees"].(string); ok && a != "" {
				assignees = parseCommaSeparatedList(a)
			}

			// Parse labels if present
			labels := []string{} // default to empty slice, can't be nil
			if l, ok := request.Params.Arguments["labels"].(string); ok && l != "" {
				labels = parseCommaSeparatedList(l)
			}

			// Create the issue request
			issueRequest := &github.IssueRequest{
				Title:     github.Ptr(title),
				Body:      github.Ptr(body),
				Assignees: &assignees,
				Labels:    &labels,
			}

			issue, resp, err := client.Issues.Create(ctx, owner, repo, issueRequest)
			if err != nil {
				return nil, fmt.Errorf("failed to create issue: %w", err)
			}
			defer func() { _ = resp.Body.Close() }()

			if resp.StatusCode != http.StatusCreated {
				body, err := io.ReadAll(resp.Body)
				if err != nil {
					return nil, fmt.Errorf("failed to read response body: %w", err)
				}
				return mcp.NewToolResultError(fmt.Sprintf("failed to create issue: %s", string(body))), nil
			}

			r, err := json.Marshal(issue)
			if err != nil {
				return nil, fmt.Errorf("failed to marshal response: %w", err)
			}

			return mcp.NewToolResultText(string(r)), nil
		}
}
