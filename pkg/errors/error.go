package errors

import (
	"fmt"

	"github.com/google/go-github/v72/github"
)

type GitHubAPIError struct {
	Message  string           `json:"message"`
	Response *github.Response `json:"-"`
	Err      error            `json:"-"`
}

func NewGitHubAPIError(message string, resp *github.Response, err error) *GitHubAPIError {
	return &GitHubAPIError{
		Message:  message,
		Response: resp,
		Err:      err,
	}
}

func (e *GitHubAPIError) Error() string {
	return fmt.Errorf("%s: %w", e.Message, e.Err).Error()
}

type GitHubGraphQLError struct {
	Message string `json:"message"`
	Err     error  `json:"-"`
}

func NewGitHubGraphQLError(message string, err error) *GitHubGraphQLError {
	return &GitHubGraphQLError{
		Message: message,
		Err:     err,
	}
}

func (e *GitHubGraphQLError) Error() string {
	return fmt.Errorf("%s: %w", e.Message, e.Err).Error()
}
