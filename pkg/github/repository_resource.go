package github

import (
	"context"
	"encoding/base64"
	"mime"
	"path/filepath"
	"strings"

	"github.com/github/github-mcp-server/pkg/translations"
	"github.com/google/go-github/v69/github"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

// getRepositoryContent defines the resource template and handler for the Repository Content API.
func getRepositoryContent(client *github.Client, t translations.TranslationHelperFunc) (mainTemplate mcp.ResourceTemplate, reftemplate mcp.ResourceTemplate, shaTemplate mcp.ResourceTemplate, tagTemplate mcp.ResourceTemplate, prTemplate mcp.ResourceTemplate, handler server.ResourceTemplateHandlerFunc) {

	return mcp.NewResourceTemplate(
			"repo://{owner}/{repo}/contents{/path*}", // Resource template
			t("RESOURCE_REPOSITORY_CONTENT_DESCRIPTION", "Repository Content"),
		), mcp.NewResourceTemplate(
			"repo://{owner}/{repo}/refs/heads/{branch}/contents{/path*}", // Resource template
			t("RESOURCE_REPOSITORY_CONTENT_BRANCH_DESCRIPTION", "Repository Content for specific branch"),
		), mcp.NewResourceTemplate(
			"repo://{owner}/{repo}/sha/{sha}/contents{/path*}", // Resource template
			t("RESOURCE_REPOSITORY_CONTENT_COMMIT_DESCRIPTION", "Repository Content for specific commit"),
		), mcp.NewResourceTemplate(
			"repo://{owner}/{repo}/refs/tags/{tag}/contents{/path*}", // Resource template
			t("RESOURCE_REPOSITORY_CONTENT_TAG_DESCRIPTION", "Repository Content for specific tag"),
		), mcp.NewResourceTemplate(
			"repo://{owner}/{repo}/refs/pull/{pr_number}/head/contents{/path*}", // Resource template
			t("RESOURCE_REPOSITORY_CONTENT_PR_DESCRIPTION", "Repository Content for specific pull request"),
		), func(ctx context.Context, request mcp.ReadResourceRequest) ([]mcp.ResourceContents, error) {
			// Extract parameters from request.Params.URI

			owner := request.Params.Arguments["owner"].([]string)[0]
			repo := request.Params.Arguments["repo"].([]string)[0]
			// path should be a joined list of the path parts
			path := strings.Join(request.Params.Arguments["path"].([]string), "/")

			opts := &github.RepositoryContentGetOptions{}

			sha, ok := request.Params.Arguments["sha"].([]string)
			if ok {
				opts.Ref = sha[0]
			}

			branch, ok := request.Params.Arguments["branch"].([]string)
			if ok {
				opts.Ref = "refs/heads/" + branch[0]
			}

			tag, ok := request.Params.Arguments["tag"].([]string)
			if ok {
				opts.Ref = "refs/tags/" + tag[0]
			}
			prNumber, ok := request.Params.Arguments["pr_number"].([]string)
			if ok {
				opts.Ref = "refs/pull/" + prNumber[0] + "/head"
			}

			// Use the GitHub client to fetch repository content
			fileContent, directoryContent, _, err := client.Repositories.GetContents(ctx, owner, repo, path, opts)
			if err != nil {
				return nil, err
			}

			if directoryContent != nil {
				// Process the directory content and return it as resource contents
				var resources []mcp.ResourceContents
				for _, entry := range directoryContent {
					mimeType := "text/directory"
					if entry.GetType() == "file" {
						mimeType = mime.TypeByExtension(filepath.Ext(entry.GetName()))
					}
					resources = append(resources, mcp.TextResourceContents{
						URI:      entry.GetHTMLURL(),
						MIMEType: mimeType,
						Text:     entry.GetName(),
					})

				}
				return resources, nil

			} else if fileContent != nil {
				// Process the file content and return it as a binary resource

				if fileContent.Content != nil {
					decodedContent, err := fileContent.GetContent()
					if err != nil {
						return nil, err
					}

					mimeType := mime.TypeByExtension(filepath.Ext(fileContent.GetName()))

					// Check if the file is text-based
					if strings.HasPrefix(mimeType, "text") {
						// Return as TextResourceContents
						return []mcp.ResourceContents{
							mcp.TextResourceContents{
								URI:      request.Params.URI,
								MIMEType: mimeType,
								Text:     decodedContent,
							},
						}, nil
					}

					// Otherwise, return as BlobResourceContents
					return []mcp.ResourceContents{
						mcp.BlobResourceContents{
							URI:      request.Params.URI,
							MIMEType: mimeType,
							Blob:     base64.StdEncoding.EncodeToString([]byte(decodedContent)), // Encode content as Base64
						},
					}, nil
				}
			}

			return nil, nil
		}
}
