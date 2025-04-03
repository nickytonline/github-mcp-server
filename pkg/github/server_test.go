package github

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/github/github-mcp-server/pkg/translations"
	"github.com/google/go-github/v69/github"
	"github.com/migueleliasweb/go-github-mock/src/mock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test_GetMe(t *testing.T) {
	// Verify tool definition
	mockClient := github.NewClient(nil)
	tool, _ := getMe(mockClient, translations.NullTranslationHelper)

	assert.Equal(t, "get_me", tool.Name)
	assert.NotEmpty(t, tool.Description)
	assert.Contains(t, tool.InputSchema.Properties, "reason")
	assert.Empty(t, tool.InputSchema.Required) // No required parameters

	// Setup mock user response
	mockUser := &github.User{
		Login:     github.Ptr("testuser"),
		Name:      github.Ptr("Test User"),
		Email:     github.Ptr("test@example.com"),
		Bio:       github.Ptr("GitHub user for testing"),
		Company:   github.Ptr("Test Company"),
		Location:  github.Ptr("Test Location"),
		HTMLURL:   github.Ptr("https://github.com/testuser"),
		CreatedAt: &github.Timestamp{Time: time.Now().Add(-365 * 24 * time.Hour)},
		Type:      github.Ptr("User"),
		Plan: &github.Plan{
			Name: github.Ptr("pro"),
		},
	}

	tests := []struct {
		name           string
		mockedClient   *http.Client
		requestArgs    map[string]interface{}
		expectError    bool
		expectedUser   *github.User
		expectedErrMsg string
	}{
		{
			name: "successful get user",
			mockedClient: mock.NewMockedHTTPClient(
				mock.WithRequestMatch(
					mock.GetUser,
					mockUser,
				),
			),
			requestArgs:  map[string]interface{}{},
			expectError:  false,
			expectedUser: mockUser,
		},
		{
			name: "successful get user with reason",
			mockedClient: mock.NewMockedHTTPClient(
				mock.WithRequestMatch(
					mock.GetUser,
					mockUser,
				),
			),
			requestArgs: map[string]interface{}{
				"reason": "Testing API",
			},
			expectError:  false,
			expectedUser: mockUser,
		},
		{
			name: "get user fails",
			mockedClient: mock.NewMockedHTTPClient(
				mock.WithRequestMatchHandler(
					mock.GetUser,
					http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
						w.WriteHeader(http.StatusUnauthorized)
						_, _ = w.Write([]byte(`{"message": "Unauthorized"}`))
					}),
				),
			),
			requestArgs:    map[string]interface{}{},
			expectError:    true,
			expectedErrMsg: "failed to get user",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// Setup client with mock
			client := github.NewClient(tc.mockedClient)
			_, handler := getMe(client, translations.NullTranslationHelper)

			// Create call request
			request := createMCPRequest(tc.requestArgs)

			// Call handler
			result, err := handler(context.Background(), request)

			// Verify results
			if tc.expectError {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tc.expectedErrMsg)
				return
			}

			require.NoError(t, err)

			// Parse result and get text content if no error
			textContent := getTextResult(t, result)

			// Unmarshal and verify the result
			var returnedUser github.User
			err = json.Unmarshal([]byte(textContent.Text), &returnedUser)
			require.NoError(t, err)

			// Verify user details
			assert.Equal(t, *tc.expectedUser.Login, *returnedUser.Login)
			assert.Equal(t, *tc.expectedUser.Name, *returnedUser.Name)
			assert.Equal(t, *tc.expectedUser.Email, *returnedUser.Email)
			assert.Equal(t, *tc.expectedUser.Bio, *returnedUser.Bio)
			assert.Equal(t, *tc.expectedUser.HTMLURL, *returnedUser.HTMLURL)
			assert.Equal(t, *tc.expectedUser.Type, *returnedUser.Type)
		})
	}
}

func Test_IsAcceptedError(t *testing.T) {
	tests := []struct {
		name           string
		err            error
		expectAccepted bool
	}{
		{
			name:           "github AcceptedError",
			err:            &github.AcceptedError{},
			expectAccepted: true,
		},
		{
			name:           "regular error",
			err:            fmt.Errorf("some other error"),
			expectAccepted: false,
		},
		{
			name:           "nil error",
			err:            nil,
			expectAccepted: false,
		},
		{
			name:           "wrapped AcceptedError",
			err:            fmt.Errorf("wrapped: %w", &github.AcceptedError{}),
			expectAccepted: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result := isAcceptedError(tc.err)
			assert.Equal(t, tc.expectAccepted, result)
		})
	}
}

func Test_RequiredStringParam(t *testing.T) {
	tests := []struct {
		name        string
		params      map[string]interface{}
		paramName   string
		expected    string
		expectError bool
	}{
		{
			name:        "valid string parameter",
			params:      map[string]interface{}{"name": "test-value"},
			paramName:   "name",
			expected:    "test-value",
			expectError: false,
		},
		{
			name:        "missing parameter",
			params:      map[string]interface{}{},
			paramName:   "name",
			expected:    "",
			expectError: true,
		},
		{
			name:        "empty string parameter",
			params:      map[string]interface{}{"name": ""},
			paramName:   "name",
			expected:    "",
			expectError: true,
		},
		{
			name:        "wrong type parameter",
			params:      map[string]interface{}{"name": 123},
			paramName:   "name",
			expected:    "",
			expectError: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			request := createMCPRequest(tc.params)
			result, err := requiredParam[string](request, tc.paramName)

			if tc.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tc.expected, result)
			}
		})
	}
}

func Test_OptionalStringParam(t *testing.T) {
	tests := []struct {
		name        string
		params      map[string]interface{}
		paramName   string
		expected    string
		expectError bool
	}{
		{
			name:        "valid string parameter",
			params:      map[string]interface{}{"name": "test-value"},
			paramName:   "name",
			expected:    "test-value",
			expectError: false,
		},
		{
			name:        "missing parameter",
			params:      map[string]interface{}{},
			paramName:   "name",
			expected:    "",
			expectError: false,
		},
		{
			name:        "empty string parameter",
			params:      map[string]interface{}{"name": ""},
			paramName:   "name",
			expected:    "",
			expectError: false,
		},
		{
			name:        "wrong type parameter",
			params:      map[string]interface{}{"name": 123},
			paramName:   "name",
			expected:    "",
			expectError: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			request := createMCPRequest(tc.params)
			result, err := optionalParam[string](request, tc.paramName)

			if tc.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tc.expected, result)
			}
		})
	}
}

func Test_RequiredNumberParam(t *testing.T) {
	tests := []struct {
		name        string
		params      map[string]interface{}
		paramName   string
		expected    int
		expectError bool
	}{
		{
			name:        "valid number parameter",
			params:      map[string]interface{}{"count": float64(42)},
			paramName:   "count",
			expected:    42,
			expectError: false,
		},
		{
			name:        "missing parameter",
			params:      map[string]interface{}{},
			paramName:   "count",
			expected:    0,
			expectError: true,
		},
		{
			name:        "wrong type parameter",
			params:      map[string]interface{}{"count": "not-a-number"},
			paramName:   "count",
			expected:    0,
			expectError: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			request := createMCPRequest(tc.params)
			result, err := requiredInt(request, tc.paramName)

			if tc.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tc.expected, result)
			}
		})
	}
}

func Test_OptionalNumberParam(t *testing.T) {
	tests := []struct {
		name        string
		params      map[string]interface{}
		paramName   string
		expected    int
		expectError bool
	}{
		{
			name:        "valid number parameter",
			params:      map[string]interface{}{"count": float64(42)},
			paramName:   "count",
			expected:    42,
			expectError: false,
		},
		{
			name:        "missing parameter",
			params:      map[string]interface{}{},
			paramName:   "count",
			expected:    0,
			expectError: false,
		},
		{
			name:        "zero value",
			params:      map[string]interface{}{"count": float64(0)},
			paramName:   "count",
			expected:    0,
			expectError: false,
		},
		{
			name:        "wrong type parameter",
			params:      map[string]interface{}{"count": "not-a-number"},
			paramName:   "count",
			expected:    0,
			expectError: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			request := createMCPRequest(tc.params)
			result, err := optionalIntParam(request, tc.paramName)

			if tc.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tc.expected, result)
			}
		})
	}
}

func Test_OptionalNumberParamWithDefault(t *testing.T) {
	tests := []struct {
		name        string
		params      map[string]interface{}
		paramName   string
		defaultVal  int
		expected    int
		expectError bool
	}{
		{
			name:        "valid number parameter",
			params:      map[string]interface{}{"count": float64(42)},
			paramName:   "count",
			defaultVal:  10,
			expected:    42,
			expectError: false,
		},
		{
			name:        "missing parameter",
			params:      map[string]interface{}{},
			paramName:   "count",
			defaultVal:  10,
			expected:    10,
			expectError: false,
		},
		{
			name:        "zero value",
			params:      map[string]interface{}{"count": float64(0)},
			paramName:   "count",
			defaultVal:  10,
			expected:    10,
			expectError: false,
		},
		{
			name:        "wrong type parameter",
			params:      map[string]interface{}{"count": "not-a-number"},
			paramName:   "count",
			defaultVal:  10,
			expected:    0,
			expectError: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			request := createMCPRequest(tc.params)
			result, err := optionalIntParamWithDefault(request, tc.paramName, tc.defaultVal)

			if tc.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tc.expected, result)
			}
		})
	}
}

func Test_OptionalBooleanParam(t *testing.T) {
	tests := []struct {
		name        string
		params      map[string]interface{}
		paramName   string
		expected    bool
		expectError bool
	}{
		{
			name:        "true value",
			params:      map[string]interface{}{"flag": true},
			paramName:   "flag",
			expected:    true,
			expectError: false,
		},
		{
			name:        "false value",
			params:      map[string]interface{}{"flag": false},
			paramName:   "flag",
			expected:    false,
			expectError: false,
		},
		{
			name:        "missing parameter",
			params:      map[string]interface{}{},
			paramName:   "flag",
			expected:    false,
			expectError: false,
		},
		{
			name:        "wrong type parameter",
			params:      map[string]interface{}{"flag": "not-a-boolean"},
			paramName:   "flag",
			expected:    false,
			expectError: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			request := createMCPRequest(tc.params)
			result, err := optionalParam[bool](request, tc.paramName)

			if tc.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tc.expected, result)
			}
		})
	}
}
