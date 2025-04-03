//go:build conformance

package conformance_test

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"reflect"
	"strings"
	"testing"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/network"
	"github.com/docker/docker/client"
	"github.com/docker/docker/pkg/stdcopy"
	"github.com/stretchr/testify/require"
)

type maintainer string

const (
	anthropic maintainer = "anthropic"
	github    maintainer = "github"
)

type testLogWriter struct {
	t *testing.T
}

func (w testLogWriter) Write(p []byte) (n int, err error) {
	w.t.Log(string(p))
	return len(p), nil
}

func start(t *testing.T, m maintainer) server {
	var image string
	if m == github {
		image = "github/github-mcp-server"
	} else {
		image = "mcp/github"
	}

	ctx := context.Background()
	dockerClient, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	require.NoError(t, err)

	containerCfg := &container.Config{
		OpenStdin:    true,
		AttachStdin:  true,
		AttachStdout: true,
		AttachStderr: true,
		Env: []string{
			fmt.Sprintf("GITHUB_PERSONAL_ACCESS_TOKEN=%s", os.Getenv("GITHUB_PERSONAL_ACCESS_TOKEN")),
		},
		Image: image,
	}

	resp, err := dockerClient.ContainerCreate(
		ctx,
		containerCfg,
		&container.HostConfig{},
		&network.NetworkingConfig{},
		nil,
		"")
	require.NoError(t, err)

	t.Cleanup(func() {
		require.NoError(t, dockerClient.ContainerRemove(ctx, resp.ID, container.RemoveOptions{Force: true}))
	})

	hijackedResponse, err := dockerClient.ContainerAttach(ctx, resp.ID, container.AttachOptions{
		Stream: true,
		Stdin:  true,
		Stdout: true,
		Stderr: true,
	})
	require.NoError(t, err)
	t.Cleanup(func() { hijackedResponse.Close() })

	require.NoError(t, dockerClient.ContainerStart(ctx, resp.ID, container.StartOptions{}))

	serverStart := make(chan serverStartResult)
	go func() {
		prOut, pwOut := io.Pipe()
		prErr, pwErr := io.Pipe()

		go func() {
			// Ignore error, we should be done?
			// TODO: maybe check for use of closed network connection specifically
			_, _ = stdcopy.StdCopy(pwOut, pwErr, hijackedResponse.Reader)
			pwOut.Close()
			pwErr.Close()
		}()

		bufferedStderr := bufio.NewReader(prErr)
		line, err := bufferedStderr.ReadString('\n')
		if err != nil {
			serverStart <- serverStartResult{err: err}
		}

		if strings.TrimSpace(line) != "GitHub MCP Server running on stdio" {
			serverStart <- serverStartResult{
				err: fmt.Errorf("unexpected server output: %s", line),
			}
			return
		}

		serverStart <- serverStartResult{
			server: server{
				m:      m,
				log:    testLogWriter{t},
				stdin:  hijackedResponse.Conn,
				stdout: bufio.NewReader(prOut),
			},
		}
	}()

	t.Logf("waiting for %s server to start...", m)
	serveResult := <-serverStart
	require.NoError(t, serveResult.err, "expected the server to start successfully")

	return serveResult.server
}

func TestCapabilities(t *testing.T) {
	anthropicServer := start(t, anthropic)
	githubServer := start(t, github)

	req := initializeRequest{
		JSONRPC: "2.0",
		ID:      1,
		Method:  "initialize",
		Params: initializeParams{
			ProtocolVersion: "2025-03-26",
			Capabilities:    clientCapabilities{},
			ClientInfo: clientInfo{
				Name:    "ConformanceTest",
				Version: "0.0.1",
			},
		},
	}

	require.NoError(t, anthropicServer.send(req))

	var anthropicInitializeResponse initializeResponse
	require.NoError(t, anthropicServer.receive(&anthropicInitializeResponse))

	require.NoError(t, githubServer.send(req))

	var ghInitializeResponse initializeResponse
	require.NoError(t, githubServer.receive(&ghInitializeResponse))

	// Any capabilities in the anthropic response should be present in the github response
	// (though the github response may have additional capabilities)
	if diff := diffNonNilFields(anthropicInitializeResponse.Result.Capabilities, ghInitializeResponse.Result.Capabilities, ""); diff != "" {
		t.Errorf("capabilities mismatch:\n%s", diff)
	}
}

func diffNonNilFields(a, b interface{}, path string) string {
	var sb strings.Builder

	va := reflect.ValueOf(a)
	vb := reflect.ValueOf(b)

	if !va.IsValid() {
		return ""
	}

	if va.Kind() == reflect.Ptr {
		if va.IsNil() {
			return ""
		}
		if !vb.IsValid() || vb.IsNil() {
			sb.WriteString(path + "\n")
			return sb.String()
		}
		va = va.Elem()
		vb = vb.Elem()
	}

	if va.Kind() != reflect.Struct || vb.Kind() != reflect.Struct {
		return ""
	}

	t := va.Type()
	for i := range va.NumField() {
		field := t.Field(i)
		if !field.IsExported() {
			continue
		}

		subPath := field.Name
		if path != "" {
			subPath = fmt.Sprintf("%s.%s", path, field.Name)
		}

		fieldA := va.Field(i)
		fieldB := vb.Field(i)

		switch fieldA.Kind() {
		case reflect.Ptr:
			if fieldA.IsNil() {
				continue // not required
			}
			if fieldB.IsNil() {
				sb.WriteString(subPath + "\n")
				continue
			}
			sb.WriteString(diffNonNilFields(fieldA.Interface(), fieldB.Interface(), subPath))

		case reflect.Struct:
			sb.WriteString(diffNonNilFields(fieldA.Interface(), fieldB.Interface(), subPath))

		default:
			zero := reflect.Zero(fieldA.Type())
			if !reflect.DeepEqual(fieldA.Interface(), zero.Interface()) {
				// fieldA is non-zero; now check that fieldB matches
				if !reflect.DeepEqual(fieldA.Interface(), fieldB.Interface()) {
					sb.WriteString(subPath + "\n")
				}
			}
		}
	}

	return sb.String()
}

type serverStartResult struct {
	server server
	err    error
}

type server struct {
	m   maintainer
	log io.Writer

	stdin  io.Writer
	stdout *bufio.Reader
}

func (s server) send(req request) error {
	b, err := req.marshal()
	if err != nil {
		return err
	}

	fmt.Fprintf(s.log, "sending %s: %s\n", s.m, string(b))

	n, err := s.stdin.Write(append(b, '\n'))
	if err != nil {
		return err
	}

	if n != len(b)+1 {
		return fmt.Errorf("wrote %d bytes, expected %d", n, len(b)+1)
	}

	return nil
}

func (s server) receive(res response) error {
	line, err := s.stdout.ReadBytes('\n')
	if err != nil {
		if err == io.EOF {
			return fmt.Errorf("EOF after reading %s", string(line))
		}
		return err
	}

	fmt.Fprintf(s.log, "received from %s: %s\n", s.m, string(line))

	return res.unmarshal(line)
}

type request interface {
	marshal() ([]byte, error)
}

type response interface {
	unmarshal([]byte) error
}

type jsonRPRCRequest[params any] struct {
	JSONRPC string `json:"jsonrpc"`
	ID      int    `json:"id"`
	Method  string `json:"method"`
	Params  params `json:"params"`
}

func (r jsonRPRCRequest[any]) marshal() ([]byte, error) {
	return json.Marshal(r)
}

type jsonRPRCResponse[result any] struct {
	JSONRPC string `json:"jsonrpc"`
	ID      int    `json:"id"`
	Method  string `json:"method"`
	Result  result `json:"result"`
}

func (r *jsonRPRCResponse[any]) unmarshal(b []byte) error {
	return json.Unmarshal(b, r)
}

type initializeRequest = jsonRPRCRequest[initializeParams]

type initializeParams struct {
	ProtocolVersion string             `json:"protocolVersion"`
	Capabilities    clientCapabilities `json:"capabilities"`
	ClientInfo      clientInfo         `json:"clientInfo"`
}

type clientCapabilities struct{} // don't actually care about any of these right now

type clientInfo struct {
	Name    string `json:"name"`
	Version string `json:"version"`
}

type initializeResponse = jsonRPRCResponse[initializeResult]

type initializeResult struct {
	ProtocolVersion string             `json:"protocolVersion"`
	Capabilities    serverCapabilities `json:"capabilities"`
	ServerInfo      serverInfo         `json:"serverInfo"`
}

type serverCapabilities struct {
	Logging *struct{} `json:"logging,omitempty"`
	Prompts *struct {
		ListChanged bool `json:"listChanged,omitempty"`
	} `json:"prompts,omitempty"`
	Resources *struct {
		Subscribe   bool `json:"subscribe,omitempty"`
		ListChanged bool `json:"listChanged,omitempty"`
	} `json:"resources,omitempty"`
	Tools *struct {
		ListChanged bool `json:"listChanged,omitempty"`
	} `json:"tools,omitempty"`
}

type serverInfo struct {
	Name    string `json:"name"`
	Version string `json:"version"`
}
