# GitHub Integration

Use the Obiente GitHub App to import repositories, deploy from branches, and trigger automatic redeploys on push.

## What This Enables

- Install GitHub access per Obiente workspace
- Use the same flow for personal and organization workspaces
- Let GitHub owners choose all repositories or selected repositories
- Browse repositories in deployment setup
- Trigger auto-deploys from GitHub App `push` webhooks

## Prerequisites

- The owner of the target GitHub personal account, or an owner of the target GitHub organization
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
- Disable **Request user authorization (OAuth) during installation**. Obiente
  starts the user authorization step after GitHub returns the installation ID so
  it can bind the workspace, installation, signed state, and PKCE verifier.
- `Webhook URL`: `https://YOUR-API-DOMAIN/webhooks/github`
- `Webhook secret`: the same value as `GITHUB_WEBHOOK_SECRET`

Generate the webhook secret yourself. GitHub does not create it for you:

```bash
openssl rand -hex 32
```

Required repository permissions:

- Metadata: read
- Contents: read
- Pull requests: read
- Deployments: write
- Checks: write
- Issues: write

Required organization permissions:

- Members: read

Subscribe to events:

- Push
- Pull request

After adding permissions to an existing GitHub App, every existing installation
must approve the requested update. Until it does, repository imports and push
deployments continue to use the old permissions, but pull request environments
cannot publish all GitHub statuses and comments.

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
DASHBOARD_URL=https://YOUR-DASHBOARD-DOMAIN
GITHUB_APP_SLUG=your-github-app-slug
NUXT_PUBLIC_GITHUB_APP_SLUG=your-github-app-slug
GITHUB_APP_ID=123456
GITHUB_APP_CLIENT_ID=your-github-app-client-id
GITHUB_APP_CLIENT_SECRET=your-github-app-client-secret
GITHUB_APP_PRIVATE_KEY_BASE64="$(base64 -w0 path/to/private-key.pem)"
GITHUB_WEBHOOK_SECRET="$(openssl rand -hex 32)"
DEPLOYMENTS_INTERNAL_SERVICE_SECRET="$(openssl rand -base64 32)"
NUXT_SESSION_PASSWORD="$(openssl rand -hex 32)"
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
- `DEPLOYMENTS_INTERNAL_SERVICE_SECRET` must be identical on every deployments-service node and must not be shared with unrelated services
- `NUXT_SESSION_PASSWORD` must be the same strong value on every dashboard replica
- Enable **Redirect on update** so repository selection changes return users to Obiente
- `DASHBOARD_URL` must exactly match the public origin used by the setup and callback URLs
- Deploy the updated dashboard and auth service together. The PKCE authorization
  code and verifier are one flow and mixed old/new service versions cannot
  complete it.

## Connecting A Workspace

1. Open `Settings -> Integrations`
2. Select the Obiente workspace
3. Click `Install GitHub App`
4. Choose the GitHub personal account or organization in GitHub
5. Select all repositories or selected repositories
6. Return to Obiente

Personal Obiente accounts are also represented as an Obiente workspace, so they
use this same install flow.

The dashboard sends users through GitHub's supported
`/apps/APP/installations/new?state=...` URL so the workspace state survives the
installation flow. If the app is already installed on a GitHub account, GitHub
opens that installation for updating and returns it to Obiente.

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

Obiente verifies `X-Hub-Signature-256` and binds the event to the stored GitHub
App installation, workspace, repository, and branch. It ignores branch deletion
events. Each accepted push builds the exact commit from the event rather than a
later branch head.

## Pull Request Environments

Enable pull request environments in a deployment's settings. The deployment is
the template: Obiente copies its repository and build configuration into a
disposable deployment, checks out the webhook's exact head SHA, and gives the
preview its own `*.my.obiente.cloud` hostname.

The settings cover:

- target branch and changed-path filters
- draft and new-push behavior
- automatic cleanup, maximum lifetime, and maximum active previews
- a separate short lifetime for temporarily restored merged previews
- hostname templates using `{pr}`, `{deployment}`, and `{branch}`
- a maintained pull request comment, GitHub Deployment, and Check Run
- approval for every PR, or for forks only
- approval of only the current SHA, or the entire PR
- explicit environment-variable and build-argument allowlists

Fork previews are always isolated. They never receive deployment environment
variables, build arguments, or persistent volumes, including after a maintainer
approves them. Same-repository previews receive only explicitly allowlisted
environment variables and build arguments. Persistent volumes are never copied
to any pull request environment.

By default, fork previews are disabled and approvals apply only to the current
head SHA. When a new commit is pushed, Obiente retires the prior GitHub
Deployment, invalidates the approval, and reports the new revision separately.

Maintainers approve, reject, redeploy, or remove a preview in Obiente. Approval
requires edit access to the source deployment, so a webhook sender cannot
approve their own code.

Before a preview is ready, its GitHub deployment links to a small public state
page instead of a dead application URL. The page shows only whether the preview
is waiting, building, unavailable, or offline; it does not expose repository,
organization, approver, or build-error data. The public origin defaults to
`https://$DOMAIN`; set `PREVIEW_STATUS_BASE_URL` when preview status pages use
another public HTTPS origin.

After an approved pull request is merged and its preview has been removed, a
maintainer can restore that exact recorded revision from the deployment
settings. Obiente creates a fresh disposable deployment without changing the
approval or copying additional values. Restored previews use the configured
restored-preview lifetime (four hours by default) and are automatically removed
again by the preview janitor.

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
- The GitHub App has `Members: read` organization permission
- Existing organization installations approved the `Members: read` permission update
- **Request user authorization (OAuth) during installation** is disabled in the GitHub App settings
- `DASHBOARD_URL` exactly matches the public dashboard origin used by the setup and callback URLs
- `NUXT_SESSION_PASSWORD` is set to the same value on every dashboard replica
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
- `DEPLOYMENTS_INTERNAL_SERVICE_SECRET` is configured consistently on every deployments-service node in a multi-node installation

### Pull request status or comment is missing

Check:

- The GitHub App subscribes to the `pull_request` event
- The installation approved `Pull requests: read`, `Deployments: write`, `Checks: write`, and `Issues: write`
- Pull request environments are enabled on the source deployment
- The PR target branch and changed files match the configured scope
- The deployment has not reached its active preview limit
- `PREVIEW_STATUS_BASE_URL` points to the public base domain when it is overridden

## Security Notes

- No GitHub user tokens are stored or refreshed
- The one-time GitHub App user authorization code is exchanged server-side and discarded
- The installer's user token is used only to confirm personal-account ownership or active organization ownership
- Installation IDs are verified with GitHub before persistence and are not trusted from query strings alone
- Installation and authorization state is signed, expires after ten minutes, and must match a short-lived HTTP-only cookie
- GitHub user authorization uses PKCE and an exact configured callback URL
- GitHub App installation tokens are minted on demand
- Webhook payloads are verified with `X-Hub-Signature-256`
- Pushes can only match deployments bound to the webhook's verified GitHub App installation and Obiente workspace
- Webhook-triggered builds fetch and record the exact pushed commit
- Pull request approvals are bound to the head SHA by default and are invalidated by new commits
- Fork previews cannot inherit deployment values, build arguments, or persistent volumes
- Pull request environments have independent deployment IDs, hostnames, build history, expiry, and cleanup
- Restoring a merged preview is limited to a previously approved recorded revision and always receives a new short expiry
- Public preview state pages use opaque URLs, are not indexed, and do not expose private build details
- Production should always use HTTPS for dashboard and webhook endpoints
