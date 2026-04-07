# GitHub Integration

Use GitHub integrations to import repositories, deploy from branches, and enable automatic redeploys on push.

## What This Enables

- Connect a personal GitHub account
- Connect a GitHub organization account from the dashboard
- Browse repositories in deployment setup
- Create repository webhooks for automatic deploys

## Prerequisites

- A GitHub account with access to the repositories you want to deploy
- Access to the Obiente dashboard
- For organization connections: an Obiente organization you can manage
- For self-hosted setups: a publicly reachable dashboard URL

## GitHub OAuth App Setup

Create a GitHub OAuth app in GitHub Developer Settings:

1. Go to `https://github.com/settings/developers`
2. Open `OAuth Apps`
3. Create a new app

Use these values:

- `Application name`: your Obiente deployment name
- `Homepage URL`: your public dashboard URL
- `Authorization callback URL`: `https://YOUR-DASHBOARD/api/github/callback`

Examples:

```text
Production: https://obiente.example.com/api/github/callback
Development: http://localhost:3000/api/github/callback
```

The callback URL must match exactly, including protocol, host, port, and path.

## Required Environment Variables

The dashboard reads the GitHub OAuth credentials from runtime config or environment variables.

Set at least:

```bash
NUXT_PUBLIC_GITHUB_CLIENT_ID=your_github_oauth_client_id
GITHUB_CLIENT_SECRET=your_github_oauth_client_secret
```

Also supported:

```bash
GITHUB_CLIENT_ID=your_github_oauth_client_id
NUXT_GITHUB_CLIENT_SECRET=your_github_oauth_client_secret
```

Recommended for self-hosted deployments:

```bash
GITHUB_TOKEN_ENCRYPTION_KEY=base64_or_high_entropy_secret
```

Notes:

- `NUXT_PUBLIC_GITHUB_CLIENT_ID` is safe to expose to the browser
- `GITHUB_CLIENT_SECRET` and `NUXT_GITHUB_CLIENT_SECRET` must stay server-side only
- If you do not provide `GITHUB_TOKEN_ENCRYPTION_KEY`, Obiente falls back to other service secrets, but a dedicated key is the safest setup

## OAuth Scopes

Obiente requests:

- `repo`
- `read:user`
- `admin:repo_hook`

Why:

- `repo`: access public and private repositories
- `read:user`: identify the connected GitHub identity
- `admin:repo_hook`: create and manage webhooks for auto-deploy

## How The Connection Flow Works

The dashboard starts OAuth from the server, not directly from the browser.

- User account connection: `/api/github/connect?type=user`
- Organization account connection: `/api/github/connect?type=organization&orgId=ORG_ID`

The server:

1. Builds the callback URL from the incoming request, including forwarded proxy headers
2. Stores a short-lived OAuth state cookie
3. Redirects to GitHub
4. Exchanges the code on the callback
5. Persists the integration through `auth-service`

This is important for load-balanced and reverse-proxied deployments because the callback URL must reflect the public origin GitHub sees.

## Connecting A Personal Account

1. Open `Settings -> Integrations`
2. Start `Connect GitHub`
3. Approve the OAuth app in GitHub
4. Return to the settings page

After success, the account should appear under connected accounts and be available in deployment repository pickers.

## Connecting An Organization Account

1. Open `Settings -> Integrations`
2. Choose the Obiente organization you want to connect for
3. Start the GitHub organization connection flow
4. Complete OAuth in GitHub

Use organization connections when repository access should belong to the organization rather than a single user.

## Auto-Deploy Webhooks

Once a repository is connected:

1. Configure the repository and branch on a deployment
2. Enable automatic deploys
3. Obiente creates or updates the GitHub webhook

Webhook endpoint:

```text
https://YOUR-DOMAIN/api/webhooks/github
```

## Troubleshooting

### `GitHub integration is not properly configured`

Usually means the dashboard cannot read the GitHub client ID or secret at runtime.

Check:

- `NUXT_PUBLIC_GITHUB_CLIENT_ID` or `GITHUB_CLIENT_ID`
- `GITHUB_CLIENT_SECRET` or `NUXT_GITHUB_CLIENT_SECRET`
- the dashboard was restarted or redeployed after changing env vars

### `Redirect URI mismatch`

The callback URL registered in GitHub does not exactly match the URL Obiente generated.

Check:

- the GitHub OAuth app callback URL
- reverse proxy headers such as `x-forwarded-host` and `x-forwarded-proto`
- that the dashboard public URL is the same URL users actually visit

### OAuth succeeds, but no connected account appears

Check the backend path, not just the browser:

- confirm the callback redirects back with `provider=github`
- inspect `auth-service` logs for failed integration saves
- make sure `auth-service` has a stable token encryption secret available
- make sure both `dashboard` and `auth-service` were redeployed after changing GitHub or encryption env vars

### `failed to encrypt GitHub token`

`auth-service` could not initialize the token cipher.

Best fix:

```bash
GITHUB_TOKEN_ENCRYPTION_KEY=your_dedicated_secret
```

Then redeploy `auth-service` and any service that reads GitHub tokens, especially `deployments-service`.

### Private repositories are missing

Check:

- the connected GitHub account actually has access
- the app was authorized with `repo`
- the connection was recreated after changing scopes

### Auto-deploy webhook creation fails

Check:

- the connection includes `admin:repo_hook`
- the repository allows webhook management
- GitHub can reach `https://YOUR-DOMAIN/api/webhooks/github`

## Security Notes

- OAuth state is validated server-side
- Tokens are stored server-side only
- Stored GitHub tokens are encrypted before persistence when service secrets are configured correctly
- Production should always use HTTPS for dashboard and webhook endpoints

## Recommended Self-Hosted Checklist

Before enabling GitHub deploys in production, verify:

- dashboard public URL is correct
- GitHub callback URL matches exactly
- `NUXT_PUBLIC_GITHUB_CLIENT_ID` is set
- `GITHUB_CLIENT_SECRET` is set
- `GITHUB_TOKEN_ENCRYPTION_KEY` is set
- `auth-service` and `deployments-service` were redeployed after secret changes
