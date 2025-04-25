//go:build e2e

package e2e_test

import (
	"context"
	"encoding/json"
	"os"
	"os/exec"
	"slices"
	"sync"
	"testing"
	"time"

	"github.com/google/go-github/v69/github"
	mcpClient "github.com/mark3labs/mcp-go/client"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/stretchr/testify/require"
)

var (
	// Shared variables and sync.Once instances to ensure one-time execution
	getTokenOnce sync.Once
	token        string

	buildOnce  sync.Once
	buildError error
)

// getE2EToken ensures the environment variable is checked only once and returns the token
func getE2EToken(t *testing.T) string {
	getTokenOnce.Do(func() {
		token = os.Getenv("GITHUB_MCP_SERVER_E2E_TOKEN")
		if token == "" {
			t.Fatalf("GITHUB_MCP_SERVER_E2E_TOKEN environment variable is not set")
		}
	})
	return token
}

// ensureDockerImageBuilt makes sure the Docker image is built only once across all tests
func ensureDockerImageBuilt(t *testing.T) {
	buildOnce.Do(func() {
		t.Log("Building Docker image for e2e tests...")
		cmd := exec.Command("docker", "build", "-t", "github/e2e-github-mcp-server", ".")
		cmd.Dir = ".." // Run this in the context of the root, where the Dockerfile is located.
		output, err := cmd.CombinedOutput()
		buildError = err
		if err != nil {
			t.Logf("Docker build output: %s", string(output))
		}
	})

	// Check if the build was successful
	require.NoError(t, buildError, "expected to build Docker image successfully")
}

// ClientOpts holds configuration options for the MCP client setup
type ClientOpts struct {
	// Environment variables to set before starting the client
	EnvVars map[string]string
}

// ClientOption defines a function type for configuring ClientOpts
type ClientOption func(*ClientOpts)

// WithEnvVars returns an option that adds environment variables to the client options
func WithEnvVars(envVars map[string]string) ClientOption {
	return func(opts *ClientOpts) {
		opts.EnvVars = envVars
	}
}

// setupMCPClient sets up the test environment and returns an initialized MCP client
// It handles token retrieval, Docker image building, and applying the provided options
func setupMCPClient(t *testing.T, options ...ClientOption) *mcpClient.Client {
	// Get token and ensure Docker image is built
	token := getE2EToken(t)
	ensureDockerImageBuilt(t)

	// Create and configure options
	opts := &ClientOpts{
		EnvVars: make(map[string]string),
	}

	// Apply all options to configure the opts struct
	for _, option := range options {
		option(opts)
	}

	// Set the GitHub token and other environment variables
	t.Setenv("GITHUB_PERSONAL_ACCESS_TOKEN", token)
	for key, value := range opts.EnvVars {
		t.Setenv(key, value)
	}

	// Prepare Docker arguments
	args := []string{
		"docker",
		"run",
		"-i",
		"--rm",
		"-e",
		"GITHUB_PERSONAL_ACCESS_TOKEN",
	}

	// Add all environment variables to the Docker arguments
	for key := range opts.EnvVars {
		args = append(args, "-e", key)
	}

	// Add the image name
	args = append(args, "github/e2e-github-mcp-server")

	// Create the client
	t.Log("Starting Stdio MCP client...")
	client, err := mcpClient.NewStdioMCPClient(args[0], []string{}, args[1:]...)
	require.NoError(t, err, "expected to create client successfully")
	t.Cleanup(func() {
		require.NoError(t, client.Close(), "expected to close client successfully")
	})

	// Initialize the client
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	request := mcp.InitializeRequest{}
	request.Params.ProtocolVersion = "2025-03-26"
	request.Params.ClientInfo = mcp.Implementation{
		Name:    "e2e-test-client",
		Version: "0.0.1",
	}

	result, err := client.Initialize(ctx, request)
	require.NoError(t, err, "failed to initialize client")
	require.Equal(t, "github-mcp-server", result.ServerInfo.Name, "unexpected server name")

	return client
}

func TestGetMe(t *testing.T) {
	mcpClient := setupMCPClient(t)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// When we call the "get_me" tool
	request := mcp.CallToolRequest{}
	request.Params.Name = "get_me"

	response, err := mcpClient.CallTool(ctx, request)
	require.NoError(t, err, "expected to call 'get_me' tool successfully")

	require.False(t, response.IsError, "expected result not to be an error")
	require.Len(t, response.Content, 1, "expected content to have one item")

	textContent, ok := response.Content[0].(mcp.TextContent)
	require.True(t, ok, "expected content to be of type TextContent")

	var trimmedContent struct {
		Login string `json:"login"`
	}
	err = json.Unmarshal([]byte(textContent.Text), &trimmedContent)
	require.NoError(t, err, "expected to unmarshal text content successfully")

	// Then the login in the response should match the login obtained via the same
	// token using the GitHub API.
	ghClient := github.NewClient(nil).WithAuthToken(getE2EToken(t))
	user, _, err := ghClient.Users.Get(context.Background(), "")
	require.NoError(t, err, "expected to get user successfully")
	require.Equal(t, trimmedContent.Login, *user.Login, "expected login to match")

}

func TestToolsets(t *testing.T) {
	mcpClient := setupMCPClient(
		t,
		WithEnvVars(map[string]string{
			"GITHUB_TOOLSETS": "repos,issues",
		}),
	)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	request := mcp.ListToolsRequest{}
	response, err := mcpClient.ListTools(ctx, request)
	require.NoError(t, err, "expected to list tools successfully")

	// We could enumerate the tools here, but we'll need to expose that information
	// declaratively in the MCP server, so for the moment let's just check the existence
	// of an issue and repo tool, and the non-existence of a pull_request tool.
	var toolsContains = func(expectedName string) bool {
		return slices.ContainsFunc(response.Tools, func(tool mcp.Tool) bool {
			return tool.Name == expectedName
		})
	}

	require.True(t, toolsContains("get_issue"), "expected to find 'get_issue' tool")
	require.True(t, toolsContains("list_branches"), "expected to find 'list_branches' tool")
	require.False(t, toolsContains("get_pull_request"), "expected not to find 'get_pull_request' tool")
}
