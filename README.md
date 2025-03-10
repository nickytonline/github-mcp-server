# GitHub MCP Server

GitHub MCP Server implemented in Go.

## Setup

Create a GitHub Personal Access Token with the appropriate permissions
and set it as the GITHUB_PERSONAL_ACCESS_TOKEN environment variable.


## Tools

### Users
- **get_me** - Get details of the authenticated user
   - No parameters required

### Issues
- **get_issue** - Gets the contents of an issue within a repository
   - `owner`: Repository owner (string, required)
   - `repo`: Repository name (string, required)
   - `issue_number`: Issue number (number, required)

- **add_issue_comment** - Add a comment to an issue
   - `owner`: Repository owner (string, required)
   - `repo`: Repository name (string, required)
   - `issue_number`: Issue number (number, required)
   - `body`: Comment text (string, required)

- **search_issues** - Search for issues and pull requests
   - `query`: Search query (string, required)
   - `sort`: Sort field (string, optional)
   - `order`: Sort order (string, optional)
   - `page`: Page number (number, optional)
   - `per_page`: Results per page (number, optional)

### Pull Requests
- **get_pull_request** - Get details of a specific pull request
   - `owner`: Repository owner (string, required)
   - `repo`: Repository name (string, required)
   - `pull_number`: Pull request number (number, required)

- **list_pull_requests** - List and filter repository pull requests
   - `owner`: Repository owner (string, required)
   - `repo`: Repository name (string, required)
   - `state`: PR state (string, optional)
   - `sort`: Sort field (string, optional)
   - `direction`: Sort direction (string, optional)
   - `per_page`: Results per page (number, optional)
   - `page`: Page number (number, optional)

- **merge_pull_request** - Merge a pull request
   - `owner`: Repository owner (string, required)
   - `repo`: Repository name (string, required)
   - `pull_number`: Pull request number (number, required)
   - `commit_title`: Title for the merge commit (string, optional)
   - `commit_message`: Message for the merge commit (string, optional)
   - `merge_method`: Merge method (string, optional)

- **get_pull_request_files** - Get the list of files changed in a pull request
   - `owner`: Repository owner (string, required)
   - `repo`: Repository name (string, required)
   - `pull_number`: Pull request number (number, required)

- **get_pull_request_status** - Get the combined status of all status checks for a pull request
   - `owner`: Repository owner (string, required)
   - `repo`: Repository name (string, required)
   - `pull_number`: Pull request number (number, required)

- **update_pull_request_branch** - Update a pull request branch with the latest changes from the base branch
   - `owner`: Repository owner (string, required)
   - `repo`: Repository name (string, required)
   - `pull_number`: Pull request number (number, required)
   - `expected_head_sha`: The expected SHA of the pull request's HEAD ref (string, optional)

- **get_pull_request_comments** - Get the review comments on a pull request
   - `owner`: Repository owner (string, required)
   - `repo`: Repository name (string, required)
   - `pull_number`: Pull request number (number, required)

- **get_pull_request_reviews** - Get the reviews on a pull request
   - `owner`: Repository owner (string, required)
   - `repo`: Repository name (string, required)
   - `pull_number`: Pull request number (number, required)

### Repositories
- **create_or_update_file** - Create or update a single file in a repository
   - `owner`: Repository owner (string, required)
   - `repo`: Repository name (string, required)
   - `path`: File path (string, required)
   - `message`: Commit message (string, required)
   - `content`: File content (string, required)
   - `branch`: Branch name (string, optional)
   - `sha`: File SHA if updating (string, optional)

- **search_repositories** - Search for GitHub repositories
   - `query`: Search query (string, required)
   - `sort`: Sort field (string, optional)
   - `order`: Sort order (string, optional)
   - `page`: Page number (number, optional)
   - `per_page`: Results per page (number, optional)

- **create_repository** - Create a new GitHub repository
   - `name`: Repository name (string, required)
   - `description`: Repository description (string, optional)
   - `private`: Whether the repository is private (boolean, optional)
   - `auto_init`: Auto-initialize with README (boolean, optional)
   - `gitignore_template`: Gitignore template name (string, optional)

- **get_file_contents** - Get contents of a file or directory
   - `owner`: Repository owner (string, required)
   - `repo`: Repository name (string, required)
   - `path`: File path (string, required)
   - `ref`: Git reference (string, optional)

- **fork_repository** - Fork a repository
   - `owner`: Repository owner (string, required)
   - `repo`: Repository name (string, required)
   - `organization`: Target organization name (string, optional)

- **create_branch** - Create a new branch
   - `owner`: Repository owner (string, required)
   - `repo`: Repository name (string, required)
   - `branch`: New branch name (string, required)
   - `sha`: SHA to create branch from (string, required)

- **list_commits** - Gets commits of a branch in a repository
   - `owner`: Repository owner (string, required)
   - `repo`: Repository name (string, required)
   - `sha`: Branch name, tag, or commit SHA (string, optional)
   - `path`: Only commits containing this file path (string, optional)
   - `page`: Page number (number, optional)
   - `per_page`: Results per page (number, optional)

### Search
- **search_code** - Search for code across GitHub repositories
   - `query`: Search query (string, required)
   - `sort`: Sort field (string, optional)
   - `order`: Sort order (string, optional)
   - `page`: Page number (number, optional)
   - `per_page`: Results per page (number, optional)

- **search_users** - Search for GitHub users
   - `query`: Search query (string, required)
   - `sort`: Sort field (string, optional)
   - `order`: Sort order (string, optional)
   - `page`: Page number (number, optional)
   - `per_page`: Results per page (number, optional)

## Standard input/output server

```sh
go run cmd/server/main.go stdio
```

E.g:

Set the PAT token in the environment variable and run:

```sh
script/get-me
```

And you should see the output of the GitHub MCP server responding with the user information.

```sh
GitHub MCP Server running on stdio
{
  "jsonrpc": "2.0",
  "id": 3,
  "result": {
    "content": [
      {
        "type": "text",
        "text": "{\"login\":\"juruen\",\"id\" ... }
      }
    ]
  }
}

```
## TODO

Lots of things!

Missing tools:

- push_files (files array)
- create_issue (assignees and labels arrays)
- list_issues (labels array)
- update_issue (labels and assignees arrays)
- create_pull_request_review (comments array)


Testing 

- Unit tests
- Integration tests
- Blackbox testing: ideally comparing output to Anthromorphic's server to make sure that this is fully compatible drop-in replacement.

And some other stuff:
- ...


