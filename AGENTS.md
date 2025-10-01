# Repository Guidelines

## Project Structure & Module Organization
The primary entrypoint lives in `cmd/github-mcp-server/main.go`; `generate_docs.go` updates documentation and `cmd/mcpcurl` offers a lightweight client. Shared logic lives under `pkg/` (tool handlers, toolsets, translations) while internal-only helpers stay in `internal/` for server wiring and schema snapshots. `e2e/` holds black-box tests, `docs/` tracks installation and policy guides, and `script/` automates linting, docs, and release chores.

## Build, Test, and Development Commands
`go build ./cmd/github-mcp-server` produces the local server binary. `go test -v ./...` runs unit and snapshot suites; use `script/test` when you need the race detector. `script/lint` wraps the required `golangci-lint run` configuration. Regenerate published docs with `script/generate-docs`, and probe tools by piping JSON-RPC into `go run ./cmd/github-mcp-server main.go stdio` as shown in `script/get-me`. Launch the HTTP transport with `github-mcp-server http --listen :8080` and authenticate requests with an `Authorization: Bearer <token>` header if you are fronting the server with OAuth-aware infrastructure like Pomerium.

## Coding Style & Naming Conventions
Format Go code with `gofmt` (tabs for indentation) and keep imports tidy via `goimports` or the editor equivalent. Follow the active `golangci-lint` ruleset (bodyclose, gosec, revive, etc.) and prefer explicit error handling. Tool identifiers exposed through MCP stay snake_case (e.g., `list_discussions`), while exported Go symbols remain PascalCase.

## Testing Guidelines
Place unit tests alongside source files with the `_test.go` suffix, using `testify/require` for fatal assertions and the project mocks (`go-github-mock`, `githubv4mock`) for GitHub APIs. When a change alters tool schemas, refresh snapshots via `UPDATE_TOOLSNAPS=true go test ./...`. End-to-end coverage lives in `e2e/`; run it with a scoped PAT using `GITHUB_MCP_SERVER_E2E_TOKEN=<token> go test --tags e2e ./e2e`, or flip `GITHUB_MCP_SERVER_E2E_DEBUG=true` to debug without Docker.

## Commit & Pull Request Guidelines
Commits in this repo typically use an imperative summary with optional context and the PR number, e.g., `Add tool for project fields (#1145)`. Before opening a PR, run `script/lint`, `go test ./...`, and any impacted e2e flows. Describe the motivation, list validation steps, and link related issues; include screenshots or logs when the change affects UX or remote output.

## Security & Configuration Tips
Never commit personal access tokens; prefer environment variables such as `GITHUB_PERSONAL_ACCESS_TOKEN` or `.env` files excluded via `.gitignore`. Limit PAT scopes to the minimum (`repo`, `read:packages`, `read:org`) and rotate them regularly. For local testing, confirm Docker is running before executing e2e suites, and review `server.json` plus docs in `docs/remote-server.md` before changing configuration defaults.
