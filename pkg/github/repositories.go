package github

import (
	"context"
	"encoding/json"
	"fmt"
	"io"

	"github.com/aws/smithy-go/ptr"
	"github.com/google/go-github/v69/github"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

// listCommits creates a tool to get commits of a branch in a repository.
func listCommits(client *github.Client) (tool mcp.Tool, handler server.ToolHandlerFunc) {
	return mcp.NewTool("list_commits",
			mcp.WithDescription("Gets commits of a branch in a repository"),
			mcp.WithString("owner",
				mcp.Required(),
				mcp.Description("Repository owner"),
			),
			mcp.WithString("repo",
				mcp.Required(),
				mcp.Description("Repository name"),
			),
			mcp.WithString("sha",
				mcp.Description("Branch name"),
			),
			mcp.WithNumber("page",
				mcp.Description("Page number"),
			),
			mcp.WithNumber("per_page",
				mcp.Description("Number of records per page"),
			),
		),
		func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			owner := request.Params.Arguments["owner"].(string)
			repo := request.Params.Arguments["repo"].(string)
			sha := ""
			if s, ok := request.Params.Arguments["sha"].(string); ok {
				sha = s
			}
			page := 1
			if p, ok := request.Params.Arguments["page"].(float64); ok {
				page = int(p)
			}
			perPage := 30
			if pp, ok := request.Params.Arguments["per_page"].(float64); ok {
				perPage = int(pp)
			}

			opts := &github.CommitsListOptions{
				SHA: sha,
				ListOptions: github.ListOptions{
					Page:    page,
					PerPage: perPage,
				},
			}

			commits, resp, err := client.Repositories.ListCommits(ctx, owner, repo, opts)
			if err != nil {
				return nil, fmt.Errorf("failed to list commits: %w", err)
			}
			defer func() { _ = resp.Body.Close() }()

			if resp.StatusCode != 200 {
				body, err := io.ReadAll(resp.Body)
				if err != nil {
					return nil, fmt.Errorf("failed to read response body: %w", err)
				}
				return mcp.NewToolResultError(fmt.Sprintf("failed to list commits: %s", string(body))), nil
			}

			r, err := json.Marshal(commits)
			if err != nil {
				return nil, fmt.Errorf("failed to marshal response: %w", err)
			}

			return mcp.NewToolResultText(string(r)), nil
		}
}

// createOrUpdateFile creates a tool to create or update a file in a GitHub repository.
func createOrUpdateFile(client *github.Client) (tool mcp.Tool, handler server.ToolHandlerFunc) {
	return mcp.NewTool("create_or_update_file",
			mcp.WithDescription("Create or update a single file in a repository"),
			mcp.WithString("owner",
				mcp.Required(),
				mcp.Description("Repository owner (username or organization)"),
			),
			mcp.WithString("repo",
				mcp.Required(),
				mcp.Description("Repository name"),
			),
			mcp.WithString("path",
				mcp.Required(),
				mcp.Description("Path where to create/update the file"),
			),
			mcp.WithString("content",
				mcp.Required(),
				mcp.Description("Content of the file"),
			),
			mcp.WithString("message",
				mcp.Required(),
				mcp.Description("Commit message"),
			),
			mcp.WithString("branch",
				mcp.Required(),
				mcp.Description("Branch to create/update the file in"),
			),
			mcp.WithString("sha",
				mcp.Description("SHA of file being replaced (for updates)"),
			),
		),
		func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			owner := request.Params.Arguments["owner"].(string)
			repo := request.Params.Arguments["repo"].(string)
			path := request.Params.Arguments["path"].(string)
			content := request.Params.Arguments["content"].(string)
			message := request.Params.Arguments["message"].(string)
			branch := request.Params.Arguments["branch"].(string)

			// Convert content to base64
			contentBytes := []byte(content)

			// Create the file options
			opts := &github.RepositoryContentFileOptions{
				Message: ptr.String(message),
				Content: contentBytes,
				Branch:  ptr.String(branch),
			}

			// If SHA is provided, set it (for updates)
			if sha, ok := request.Params.Arguments["sha"].(string); ok && sha != "" {
				opts.SHA = ptr.String(sha)
			}

			// Create or update the file
			fileContent, resp, err := client.Repositories.CreateFile(ctx, owner, repo, path, opts)
			if err != nil {
				return nil, fmt.Errorf("failed to create/update file: %w", err)
			}
			defer func() { _ = resp.Body.Close() }()

			if resp.StatusCode != 200 && resp.StatusCode != 201 {
				body, err := io.ReadAll(resp.Body)
				if err != nil {
					return nil, fmt.Errorf("failed to read response body: %w", err)
				}
				return mcp.NewToolResultError(fmt.Sprintf("failed to create/update file: %s", string(body))), nil
			}

			r, err := json.Marshal(fileContent)
			if err != nil {
				return nil, fmt.Errorf("failed to marshal response: %w", err)
			}

			return mcp.NewToolResultText(string(r)), nil
		}
}

// createRepository creates a tool to create a new GitHub repository.
func createRepository(client *github.Client) (tool mcp.Tool, handler server.ToolHandlerFunc) {
	return mcp.NewTool("create_repository",
			mcp.WithDescription("Create a new GitHub repository"),
			mcp.WithString("name",
				mcp.Required(),
				mcp.Description("Repository name"),
			),
			mcp.WithString("description",
				mcp.Description("Repository description"),
			),
			mcp.WithBoolean("private",
				mcp.Description("Whether repo should be private"),
			),
			mcp.WithBoolean("auto_init",
				mcp.Description("Initialize with README"),
			),
		),
		func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			name := request.Params.Arguments["name"].(string)
			description := ""
			if desc, ok := request.Params.Arguments["description"].(string); ok {
				description = desc
			}
			private := false
			if priv, ok := request.Params.Arguments["private"].(bool); ok {
				private = priv
			}
			autoInit := false
			if init, ok := request.Params.Arguments["auto_init"].(bool); ok {
				autoInit = init
			}

			repo := &github.Repository{
				Name:        github.String(name),
				Description: github.String(description),
				Private:     github.Bool(private),
				AutoInit:    github.Bool(autoInit),
			}

			createdRepo, resp, err := client.Repositories.Create(ctx, "", repo)
			if err != nil {
				return nil, fmt.Errorf("failed to create repository: %w", err)
			}
			defer func() { _ = resp.Body.Close() }()

			if resp.StatusCode != 201 {
				body, err := io.ReadAll(resp.Body)
				if err != nil {
					return nil, fmt.Errorf("failed to read response body: %w", err)
				}
				return mcp.NewToolResultError(fmt.Sprintf("failed to create repository: %s", string(body))), nil
			}

			r, err := json.Marshal(createdRepo)
			if err != nil {
				return nil, fmt.Errorf("failed to marshal response: %w", err)
			}

			return mcp.NewToolResultText(string(r)), nil
		}
}

// getFileContents creates a tool to get the contents of a file or directory from a GitHub repository.
func getFileContents(client *github.Client) (tool mcp.Tool, handler server.ToolHandlerFunc) {
	return mcp.NewTool("get_file_contents",
			mcp.WithDescription("Get contents of a file or directory"),
			mcp.WithString("owner",
				mcp.Required(),
				mcp.Description("Repository owner (username or organization)"),
			),
			mcp.WithString("repo",
				mcp.Required(),
				mcp.Description("Repository name"),
			),
			mcp.WithString("path",
				mcp.Required(),
				mcp.Description("Path to file/directory"),
			),
			mcp.WithString("branch",
				mcp.Description("Branch to get contents from"),
			),
		),
		func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			owner := request.Params.Arguments["owner"].(string)
			repo := request.Params.Arguments["repo"].(string)
			path := request.Params.Arguments["path"].(string)
			branch := ""
			if b, ok := request.Params.Arguments["branch"].(string); ok {
				branch = b
			}

			opts := &github.RepositoryContentGetOptions{Ref: branch}
			fileContent, dirContent, resp, err := client.Repositories.GetContents(ctx, owner, repo, path, opts)
			if err != nil {
				return nil, fmt.Errorf("failed to get file contents: %w", err)
			}
			defer func() { _ = resp.Body.Close() }()

			if resp.StatusCode != 200 {
				body, err := io.ReadAll(resp.Body)
				if err != nil {
					return nil, fmt.Errorf("failed to read response body: %w", err)
				}
				return mcp.NewToolResultError(fmt.Sprintf("failed to get file contents: %s", string(body))), nil
			}

			var result interface{}
			if fileContent != nil {
				result = fileContent
			} else {
				result = dirContent
			}

			r, err := json.Marshal(result)
			if err != nil {
				return nil, fmt.Errorf("failed to marshal response: %w", err)
			}

			return mcp.NewToolResultText(string(r)), nil
		}
}

// forkRepository creates a tool to fork a repository.
func forkRepository(client *github.Client) (tool mcp.Tool, handler server.ToolHandlerFunc) {
	return mcp.NewTool("fork_repository",
			mcp.WithDescription("Fork a repository"),
			mcp.WithString("owner",
				mcp.Required(),
				mcp.Description("Repository owner"),
			),
			mcp.WithString("repo",
				mcp.Required(),
				mcp.Description("Repository name"),
			),
			mcp.WithString("organization",
				mcp.Description("Organization to fork to"),
			),
		),
		func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			owner := request.Params.Arguments["owner"].(string)
			repo := request.Params.Arguments["repo"].(string)
			org := ""
			if o, ok := request.Params.Arguments["organization"].(string); ok {
				org = o
			}

			opts := &github.RepositoryCreateForkOptions{}
			if org != "" {
				opts.Organization = org
			}

			forkedRepo, resp, err := client.Repositories.CreateFork(ctx, owner, repo, opts)
			if err != nil {
				return nil, fmt.Errorf("failed to fork repository: %w", err)
			}
			defer func() { _ = resp.Body.Close() }()

			if resp.StatusCode != 202 {
				body, err := io.ReadAll(resp.Body)
				if err != nil {
					return nil, fmt.Errorf("failed to read response body: %w", err)
				}
				return mcp.NewToolResultError(fmt.Sprintf("failed to fork repository: %s", string(body))), nil
			}

			r, err := json.Marshal(forkedRepo)
			if err != nil {
				return nil, fmt.Errorf("failed to marshal response: %w", err)
			}

			return mcp.NewToolResultText(string(r)), nil
		}
}

// createBranch creates a tool to create a new branch.
func createBranch(client *github.Client) (tool mcp.Tool, handler server.ToolHandlerFunc) {
	return mcp.NewTool("create_branch",
			mcp.WithDescription("Create a new branch"),
			mcp.WithString("owner",
				mcp.Required(),
				mcp.Description("Repository owner"),
			),
			mcp.WithString("repo",
				mcp.Required(),
				mcp.Description("Repository name"),
			),
			mcp.WithString("branch",
				mcp.Required(),
				mcp.Description("Name for new branch"),
			),
			mcp.WithString("from_branch",
				mcp.Description("Source branch (defaults to repo default)"),
			),
		),
		func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			owner := request.Params.Arguments["owner"].(string)
			repo := request.Params.Arguments["repo"].(string)
			branch := request.Params.Arguments["branch"].(string)
			fromBranch := ""
			if fb, ok := request.Params.Arguments["from_branch"].(string); ok {
				fromBranch = fb
			}

			// Get the source branch SHA
			var ref *github.Reference
			var err error

			if fromBranch == "" {
				// Get default branch if from_branch not specified
				repository, resp, err := client.Repositories.Get(ctx, owner, repo)
				if err != nil {
					return nil, fmt.Errorf("failed to get repository: %w", err)
				}
				defer func() { _ = resp.Body.Close() }()

				fromBranch = *repository.DefaultBranch
			}

			// Get SHA of source branch
			ref, resp, err := client.Git.GetRef(ctx, owner, repo, "refs/heads/"+fromBranch)
			if err != nil {
				return nil, fmt.Errorf("failed to get reference: %w", err)
			}
			defer func() { _ = resp.Body.Close() }()

			// Create new branch
			newRef := &github.Reference{
				Ref:    github.Ptr("refs/heads/" + branch),
				Object: &github.GitObject{SHA: ref.Object.SHA},
			}

			createdRef, resp, err := client.Git.CreateRef(ctx, owner, repo, newRef)
			if err != nil {
				return nil, fmt.Errorf("failed to create branch: %w", err)
			}
			defer func() { _ = resp.Body.Close() }()

			r, err := json.Marshal(createdRef)
			if err != nil {
				return nil, fmt.Errorf("failed to marshal response: %w", err)
			}

			return mcp.NewToolResultText(string(r)), nil
		}
}
