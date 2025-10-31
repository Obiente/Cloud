# GitHub Integration Setup

This guide explains how to set up GitHub OAuth integration to enable repository imports and deployments from GitHub.

## Prerequisites

- A GitHub account
- Access to your Obiente Cloud dashboard
- Admin or organization owner permissions

## Creating a GitHub OAuth App

1. **Navigate to GitHub Developer Settings**

   - Go to [GitHub Settings > Developer settings](https://github.com/settings/developers)
   - Click **OAuth Apps** in the left sidebar
   - Click **New OAuth App**

2. **Configure OAuth App Details**

   - **Application name**: `Obiente Cloud` (or your custom name)
   - **Homepage URL**: Your Obiente Cloud dashboard URL
     - Production: `https://your-domain.com`
     - Development: `http://localhost:3000`
   - **Authorization callback URL**: **IMPORTANT** - Use one of these:

     **For Production:**

     ```
     https://your-domain.com/api/github/callback
     ```

     **For Development:**

     ```
     http://localhost:3000/api/github/callback
     ```

     > **Note**: The callback URL must match exactly what you configure in your `.env` file. This is where GitHub will redirect users after they authorize your application.

3. **Register the Application**

   - Click **Register application**
   - GitHub will generate a **Client ID** and **Client Secret**

4. **Save Your Credentials**
   - **Client ID**: Copy this immediately (you can always see it later)
   - **Client Secret**: Click **Generate a new client secret** and copy it immediately
     - ⚠️ **Warning**: You can only see the client secret once. Save it securely!

## Required OAuth Scopes

Your GitHub OAuth App will request the following scopes:

- **`repo`** - Full control of private repositories
  - Required for accessing private repositories
  - Allows reading and writing repository contents, branches, and files
  - Required for fetching `docker-compose.yml` files from private repos
  - Includes read access to repository webhooks

- **`read:user`** - Read user profile data
  - Required to identify which GitHub user is connecting
  - Allows reading basic user information (username, email, etc.)

- **`admin:repo_hook`** - Full control of repository hooks
  - Required for creating and managing repository webhooks
  - Enables automatic deployments on push events
  - Allows configuring webhook URLs, events, and secrets

### Scope Breakdown

| Scope            | Purpose                                    | Required                                     |
| ---------------- | ------------------------------------------ | -------------------------------------------- |
| `repo`            | Access to private repositories             | ✅ Yes (for full functionality)              |
| `read:user`       | Read GitHub user profile                   | ✅ Yes (to identify connected account)       |
| `admin:repo_hook` | Full control of repository hooks/webhooks | ✅ Yes (for autodeploy on push functionality) |

**Why these scopes?**

- **`repo` scope** provides:
  - List all repositories (public and private)
  - Read repository contents and branches
  - Access `docker-compose.yml` files from private repositories
  - Read repository webhooks

- **`admin:repo_hook` scope** is required for:
  - Creating repository webhooks automatically
  - Configuring webhook endpoints for push events
  - Managing webhook secrets for secure delivery
  - Enabling automatic deployments when code is pushed

**Without `admin:repo_hook`:**
- You won't be able to set up automatic deployments on push
- Manual deployment triggering will still work
- Repository browsing and manual deployments will function normally

**Alternative (Limited Functionality):**
If you only want to support public repositories and don't need autodeploy, you can use `public_repo` scope instead of `repo`, but this will limit functionality to public repositories only and autodeploy will not be available.

## Configuration

### 1. Update Environment Variables

Add the following to your `.env` file (see `.env.example`):

```bash
# GitHub OAuth Client ID (public - exposed to client-side)
NUXT_PUBLIC_GITHUB_CLIENT_ID=your_client_id_here

# GitHub OAuth Client Secret (server-side only - NEVER expose in client code)
GITHUB_CLIENT_SECRET=your_client_secret_here
```

### 2. Update Dashboard Configuration

The dashboard will automatically pick up these environment variables. No additional configuration needed.

### 3. Verify Callback URL

Ensure your GitHub OAuth App's **Authorization callback URL** matches:

**Production:**

```
https://your-domain.com/api/github/callback
```

**Development:**

```
http://localhost:3000/api/github/callback
```

> **Important**: The callback URL must:
>
> - Be an absolute URL (include `http://` or `https://`)
> - Match exactly (including protocol, domain, port, and path)
> - Be accessible from the internet (for production)

## Connecting Your GitHub Account

1. Navigate to **Settings** > **Integrations** in your dashboard
2. Click **Connect GitHub**
3. You'll be redirected to GitHub to authorize the application
4. Review the requested permissions and click **Authorize**
5. You'll be redirected back to your dashboard with GitHub connected

## Troubleshooting

### "Redirect URI mismatch" Error

This error means the callback URL in your GitHub OAuth App doesn't match what's configured in your code.

**Solution:**

1. Check your GitHub OAuth App settings
2. Ensure the **Authorization callback URL** matches exactly:
   - `http://localhost:3000/api/github/callback` (development)
   - `https://your-domain.com/api/github/callback` (production)
3. The URL is case-sensitive and must match exactly

### "Bad credentials" Error

This usually means the Client ID or Client Secret is incorrect.

**Solution:**

1. Verify `NUXT_PUBLIC_GITHUB_CLIENT_ID` in your `.env` matches your GitHub OAuth App's Client ID
2. Verify `GITHUB_CLIENT_SECRET` in your `.env` matches your GitHub OAuth App's Client Secret
3. If you regenerated the client secret, make sure you're using the new one (old secrets are invalidated)

### Cannot See Private Repositories

If you can only see public repositories:

**Solution:**

1. Verify your OAuth App requests the `repo` scope (not just `public_repo`)
2. Re-authorize the connection if you initially authorized with limited scopes
3. Check that your GitHub account has access to the private repositories

### Autodeploy Not Working / Webhook Creation Failed

If automatic deployments on push are not working:

**Solution:**

1. Verify your OAuth App has the `admin:repo_hook` scope
2. Re-authorize your GitHub connection to grant webhook permissions
3. Check your repository's webhook settings in GitHub (`Settings > Webhooks`)
4. Verify the webhook URL is correct: `https://your-domain.com/api/webhooks/github`
5. Check webhook delivery logs in GitHub for any errors
6. Ensure the deployment has autodeploy enabled in its settings

### Callback Endpoint Not Found

If you get a 404 when GitHub tries to redirect back:

**Solution:**

1. Ensure the callback endpoint is implemented: `/api/github/callback`
2. Check that your server is running and accessible
3. Verify the callback URL in your GitHub OAuth App matches your server URL

## Security Considerations

1. **Client Secret**: Never expose `GITHUB_CLIENT_SECRET` in client-side code. It should only be used server-side.

2. **HTTPS in Production**: Always use HTTPS in production for OAuth callbacks to protect user credentials.

3. **State Parameter**: The OAuth flow uses a state parameter to prevent CSRF attacks. This is handled automatically.

4. **Token Storage**: GitHub access tokens are stored securely on the server and never exposed to the client.

## Automatic Deployments on Push

Once your GitHub account is connected, you can enable automatic deployments that trigger whenever code is pushed to your repository.

### How It Works

1. **Webhook Creation**: When you configure a deployment to use a GitHub repository with autodeploy enabled, Obiente Cloud automatically creates a webhook in your GitHub repository
2. **Event Monitoring**: The webhook listens for `push` events on the configured branch (e.g., `main`, `master`)
3. **Automatic Trigger**: When a push occurs, GitHub sends a webhook event to Obiente Cloud
4. **Deployment**: Obiente Cloud automatically triggers a deployment build and deploys the latest code

### Enabling Autodeploy

1. **Create or Edit a Deployment**
   - Navigate to your deployment settings
   - Select a GitHub repository and branch
   - Enable "Auto-deploy on push" option (coming in UI)

2. **Webhook Setup** (Automatic)
   - The system automatically creates a webhook in your GitHub repository
   - Webhook URL: `https://your-domain.com/api/webhooks/github`
   - Events: `push` events for the configured branch

3. **Verification**
   - Check your repository's webhook settings: `Settings > Webhooks`
   - You should see a webhook configured for Obiente Cloud
   - Test by pushing a commit to your repository

### Webhook Security

- Webhooks are secured with a secret that's automatically generated and stored
- GitHub signs webhook payloads using HMAC-SHA256
- Obiente Cloud validates webhook signatures before processing deployments

### Manual Deployment

Even with autodeploy enabled, you can still trigger deployments manually:
- Use the "Deploy" button in the deployment overview
- Manual deployments allow you to deploy specific commits or branches

## Next Steps

After connecting your GitHub account:

- ✅ Browse and select repositories directly in the deployment overview
- ✅ Auto-detect branches from connected repositories  
- ✅ Load `docker-compose.yml` files from GitHub repositories
- ✅ **Automatic deployments on push** - Enabled with `admin:repo_hook` scope

## Related Documentation

- [Environment Variables Reference](../reference/environment-variables.md)
- [Deployment Guide](./deployment.md)
