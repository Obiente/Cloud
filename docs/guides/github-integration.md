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
GITHUB_WEBHOOK_SECRET=your_github_webhook_signing_secret
GITHUB_WEBHOOK_URL=https://api.your-domain.example/webhooks/github
GITHUB_APP_SLUG=your-obiente-github-app-slug
NUXT_PUBLIC_GITHUB_APP_SLUG=your-obiente-github-app-slug
GITHUB_APP_ID=123456
GITHUB_APP_PRIVATE_KEY_BASE64=base64_encoded_private_key_pem
```

Generate the webhook secret yourself, then use the same value in both Obiente and
the GitHub webhook configuration. GitHub does not generate this secret for you.

```bash
openssl rand -hex 32
```

Notes:

- `NUXT_PUBLIC_GITHUB_CLIENT_ID` is safe to expose to the browser
- `GITHUB_CLIENT_SECRET` and `NUXT_GITHUB_CLIENT_SECRET` must stay server-side only
- If you do not provide `GITHUB_TOKEN_ENCRYPTION_KEY`, Obiente falls back to other service secrets, but a dedicated key is the safest setup
- `GITHUB_WEBHOOK_SECRET` is a random value you create; it must match the secret configured on GitHub webhooks for automatic deployments
- `GITHUB_WEBHOOK_URL` is optional when `API_URL` is a public URL; otherwise set it explicitly

## OAuth Scopes

Obiente requests:

- `repo`
- `read:user`
- `admin:repo_hook`

Why:

- `repo`: access public and private repositories
- `read:user`: identify the connected GitHub identity
- `admin:repo_hook`: create and manage webhooks for auto-deploy

GitHub also requires the connected user to have **Admin** access on each
repository where Obiente should create or update webhooks. Read, triage, write,
or maintain access can be enough to list or clone a repository, but it is not
enough to manage repository webhooks.

## GitHub App Organization Installs

For Obiente organization-level connections, prefer the GitHub App install flow.
The GitHub organization owner installs the Obiente GitHub App on all
repositories or selected repositories, and Obiente stores the installation ID on
the Obiente organization.

Configure the GitHub App with:

- setup URL: `https://YOUR-DASHBOARD-DOMAIN/api/github/app/callback`
- enable **Redirect on update**
- webhook URL: `https://YOUR-API-DOMAIN/webhooks/github`
- webhook secret: the same value as `GITHUB_WEBHOOK_SECRET`
- repository permissions:
  - Metadata: read
  - Contents: read
- subscribe to the `push` webhook event
- keep **Request user authorization (OAuth) during installation** disabled

Then set:

```bash
GITHUB_APP_SLUG=your-github-app-slug
NUXT_PUBLIC_GITHUB_APP_SLUG=your-github-app-slug
GITHUB_APP_ID=123456
GITHUB_APP_PRIVATE_KEY_BASE64="$(base64 -w0 path/to/private-key.pem)"
```

On macOS, use:

```bash
GITHUB_APP_PRIVATE_KEY_BASE64="$(base64 < path/to/private-key.pem | tr -d '\n')"
```

The setup URL uses the dashboard domain, not the API domain. If your dashboard
is served from `https://obiente.cloud`, use:

```text
https://obiente.cloud/api/github/app/callback
```

If your dashboard is served from `https://dashboard.example.com`, use:

```text
https://dashboard.example.com/api/github/app/callback
```

Enable **Redirect on update** so GitHub sends users back to Obiente after they
add or remove repositories from the installation. Obiente uses the live
installation token permissions when listing repos, so repository selection
changes become visible after the update redirect.

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

When a deployment is saved with a connected GitHub repository, `deployments-service` automatically creates or refreshes a repository `push` webhook. GitHub calls `/webhooks/github`, and Obiente matches the pushed repository and branch to deployments that have `github_integration_id` set.

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
2. Leave **Auto Deploy** enabled, or turn it back on in deployment settings
3. Obiente creates or updates the GitHub webhook

Webhook endpoint:

```text
https://YOUR-API-DOMAIN/webhooks/github
```

Webhook secret:

```bash
openssl rand -hex 32
```

Save the generated value as `GITHUB_WEBHOOK_SECRET` and paste the same value into
the GitHub webhook **Secret** field.

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
- the connected GitHub user has **Admin** access on the selected repository
- the GitHub OAuth app is approved/allowed for the repository's organization
- the repository allows webhook management
- `GITHUB_WEBHOOK_SECRET` is set on `deployments-service`
- `GITHUB_WEBHOOK_URL` is set, or `API_URL` points at the public API gateway
- GitHub can reach `https://YOUR-API-DOMAIN/webhooks/github`

If GitHub returns `Resource not accessible by integration` after a fresh
connection, the token is usually valid but cannot administer webhooks for that
repository. Reconnect after the organization owner approves the OAuth app and
the connected GitHub user has repository Admin access.

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
