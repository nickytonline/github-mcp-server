# Deploying GitHub MCP Server behind Pomerium

This example demonstrates how to run the GitHub MCP Server in HTTP mode behind [Pomerium](https://www.pomerium.com/), forwarding GitHub OAuth tokens to the server on each request.

1. Start the server in HTTP mode:

```bash
github-mcp-server http \
  --listen :8080 \
  --http-path /mcp \
  --health-path /health
```

2. Configure Pomerium to authenticate users with GitHub and pass the resulting access token to the upstream via the `Authorization` header. A simplified route looks like:

```yaml
routes:
  - from: https://mcp.example.com
    to: http://github-mcp-server:8080
    preserve_host_header: true

    enable_google_cloud_serverless_authentication: false
    pass_identity_headers: true

    # Forward OAuth tokens from GitHub to the MCP server
    set_request_headers:
      Authorization: "Bearer {{ .Pomerium.JWT.OAuth.AccessToken }}"

    upstream_oauth2:
      client_id: ${GITHUB_OAUTH_CLIENT_ID}
      client_secret: ${GITHUB_OAUTH_CLIENT_SECRET}
      scopes:
        - read:user
        - user:email
        - repo
        - read:org
      endpoint:
        auth_url: https://github.com/login/oauth/authorize
        token_url: https://github.com/login/oauth/access_token
```

3. Point your MCP host at `https://mcp.example.com/mcp` and omit the static `GITHUB_PERSONAL_ACCESS_TOKEN`. Each request will be authenticated with the user’s GitHub OAuth token issued by Pomerium.

Refer to [Pomerium's MCP documentation](https://www.pomerium.com/docs/capabilities/mcp) for deployment details and advanced routing options.
