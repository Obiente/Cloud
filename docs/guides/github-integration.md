# GitHub Integration

Use the Obiente GitHub App to import repositories, deploy from branches, and trigger automatic redeploys on push.

## What This Enables

- Install GitHub access per Obiente workspace
- Use the same flow for personal and organization workspaces
- Let GitHub owners choose all repositories or selected repositories
- Browse repositories in deployment setup
- Trigger auto-deploys from GitHub App `push` webhooks

## Prerequisites

- A GitHub account that can install GitHub Apps on the target account or organization
- Access to the Obiente dashboard
- An Obiente workspace you can manage
- For self-hosted setups: public dashboard and API URLs

## GitHub App Setup

Create a GitHub App in GitHub Developer Settings:

1. Go to `https://github.com/settings/apps`
2. Create a new GitHub App
3. Configure post-install and webhook settings

Use these values:

- `Homepage URL`: your public dashboard URL
- `Setup URL`: `https://YOUR-DASHBOARD-DOMAIN/api/github/app/callback`
- `Callback URL`: `https://YOUR-DASHBOARD-DOMAIN/api/github/app/callback`
- Enable **Redirect on update**
- Enable **Request user authorization (OAuth) during installation**
- `Webhook URL`: `https://YOUR-API-DOMAIN/webhooks/github`
- `Webhook secret`: the same value as `GITHUB_WEBHOOK_SECRET`

Generate the webhook secret yourself. GitHub does not create it for you:

```bash
openssl rand -hex 32
```

Required repository permissions:

- Metadata: read
- Contents: read

Subscribe to events:

- Push

The setup URL uses the dashboard domain. The webhook URL uses the API domain.
For example, if users visit `https://obiente.cloud` and your API is
`https://api.obiente.cloud`, configure:

```text
Setup URL: https://obiente.cloud/api/github/app/callback
Webhook URL: https://api.obiente.cloud/webhooks/github
```

## Required Environment Variables

Set these in production:

```bash
GITHUB_APP_SLUG=your-github-app-slug
NUXT_PUBLIC_GITHUB_APP_SLUG=your-github-app-slug
GITHUB_APP_ID=123456
GITHUB_APP_CLIENT_ID=your-github-app-client-id
GITHUB_APP_CLIENT_SECRET=your-github-app-client-secret
GITHUB_APP_PRIVATE_KEY_BASE64="$(base64 -w0 path/to/private-key.pem)"
GITHUB_WEBHOOK_SECRET="$(openssl rand -hex 32)"
```

On macOS, encode the private key with:

```bash
GITHUB_APP_PRIVATE_KEY_BASE64="$(base64 < path/to/private-key.pem | tr -d '\n')"
```

Notes:

- `NUXT_PUBLIC_GITHUB_APP_SLUG` is safe to expose to the browser
- `GITHUB_APP_CLIENT_SECRET` is used only to exchange the one-time setup code and must stay server-side
- `GITHUB_APP_PRIVATE_KEY_BASE64` must stay server-side only
- `GITHUB_WEBHOOK_SECRET` must match the secret configured on the GitHub App
- Enable **Redirect on update** so repository selection changes return users to Obiente

## Connecting A Workspace

1. Open `Settings -> Integrations`
2. Select the Obiente workspace
3. Click `Install GitHub App`
4. Choose the GitHub personal account or organization in GitHub
5. Select all repositories or selected repositories
6. Return to Obiente

Personal Obiente accounts are also represented as an Obiente workspace, so they
use this same install flow.

The dashboard sends users through GitHub's target-selection install URL so
existing GitHub App installations can be selected and returned to Obiente with
the workspace state. If the app is already installed on a GitHub account, choose
that account from GitHub's install target picker and save/update the installation.

## Auto-Deploy Webhooks

GitHub App installations manage the webhook centrally. Obiente no longer creates
per-repository webhooks.

Once the app is installed:

1. Configure the repository and branch on a deployment
2. Leave **Auto Deploy** enabled, or turn it back on in deployment settings
3. Push to the configured branch

GitHub calls:

```text
https://YOUR-API-DOMAIN/webhooks/github
```

Obiente verifies `X-Hub-Signature-256`, matches the repository and branch, and
triggers deployments that have auto-deploy enabled.

## Troubleshooting

### GitHub App is not configured

Check:

- `GITHUB_APP_SLUG`
- `NUXT_PUBLIC_GITHUB_APP_SLUG`
- Dashboard was restarted after changing env vars

### Installation saves fail

Check:

- `GITHUB_APP_ID`
- `GITHUB_APP_PRIVATE_KEY_BASE64`
- The private key belongs to the same GitHub App
- `auth-service` and `deployments-service` were restarted

### Private repositories are missing

Check:

- The repository was included in the GitHub App installation
- The app has `Contents: read`
- The user returned through the setup callback after installing or updating repository access
- The install flow was started from `Settings -> Integrations` for the correct Obiente workspace

### Auto-deploy does not trigger

Check:

- The GitHub App subscribes to the `push` event
- `GITHUB_WEBHOOK_SECRET` matches the GitHub App webhook secret
- GitHub can reach `https://YOUR-API-DOMAIN/webhooks/github`
- The deployment repository and branch match the pushed repository and branch
- Auto Deploy is enabled on the deployment

## Security Notes

- No GitHub user tokens are stored or refreshed
- The one-time GitHub App user authorization code is exchanged server-side and discarded
- The installer's user token is used only to confirm the installation is visible to that GitHub user
- Installation IDs are verified with GitHub before persistence and are not trusted from query strings alone
- Installation state is validated server-side with a short-lived cookie
- GitHub App installation tokens are minted on demand
- Webhook payloads are verified with `X-Hub-Signature-256`
- Production should always use HTTPS for dashboard and webhook endpoints
