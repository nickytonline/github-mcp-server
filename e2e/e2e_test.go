//go:build e2e

package e2e_test

import (
	"context"
	"encoding/json"
	"os"
	"os/exec"
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
	e2eToken     string

	buildOnce  sync.Once
	buildError error
)

// getE2EToken ensures the environment variable is checked only once and returns the token
func getE2EToken(t *testing.T) string {
	getTokenOnce.Do(func() {
		e2eToken = os.Getenv("GITHUB_MCP_SERVER_E2E_TOKEN")
		if e2eToken == "" {
			t.Fatalf("GITHUB_MCP_SERVER_E2E_TOKEN environment variable is not set")
		}
	})
	return e2eToken
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

func TestE2E(t *testing.T) {
	token := getE2EToken(t)
	ensureDockerImageBuilt(t)

	t.Setenv("GITHUB_PERSONAL_ACCESS_TOKEN", token) // The MCP Client merges the existing environment.
	args := []string{
		"docker",
		"run",
		"-i",
		"--rm",
		"-e",
		"GITHUB_PERSONAL_ACCESS_TOKEN",
		"github/e2e-github-mcp-server",
	}
	t.Log("Starting Stdio MCP client...")
	client, err := mcpClient.NewStdioMCPClient(args[0], []string{}, args[1:]...)
	require.NoError(t, err, "expected to create client successfully")

	t.Run("Initialize", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		request := mcp.InitializeRequest{}
		request.Params.ProtocolVersion = "2025-03-26"
		request.Params.ClientInfo = mcp.Implementation{
			Name:    "e2e-test-client",
			Version: "0.0.1",
		}

		result, err := client.Initialize(ctx, request)
		require.NoError(t, err, "expected to initialize successfully")

		require.Equal(t, "github-mcp-server", result.ServerInfo.Name)
	})

	t.Run("CallTool get_me", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		// When we call the "get_me" tool
		request := mcp.CallToolRequest{}
		request.Params.Name = "get_me"

		response, err := client.CallTool(ctx, request)
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
		client := github.NewClient(nil).WithAuthToken(token)
		user, _, err := client.Users.Get(context.Background(), "")
		require.NoError(t, err, "expected to get user successfully")
		require.Equal(t, trimmedContent.Login, *user.Login, "expected login to match")
	})

	require.NoError(t, client.Close(), "expected to close client successfully")
}
