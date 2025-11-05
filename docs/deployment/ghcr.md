# GitHub Container Registry Setup

This repository uses GitHub Container Registry (ghcr.io) for Docker image builds and distribution.

## Overview

The Docker image for the API is automatically built and pushed to GitHub Container Registry on:
- Push to `main` branch
- Push of version tags (e.g., `v1.0.0`)
- Manual workflow dispatch

## Image Location

Images are available at:
- `ghcr.io/obiente/cloud-api:latest` - Latest build from main branch
- `ghcr.io/obiente/cloud-api:main` - Latest build from main branch (alternative tag)
- `ghcr.io/obiente/cloud-api:v1.0.0` - Tagged versions
- `ghcr.io/obiente/cloud-api:main-<sha>` - Build-specific tags

## Authentication

To pull images from GitHub Container Registry, you need to authenticate:

### Using Personal Access Token

1. Create a Personal Access Token (PAT) with `read:packages` scope:
   - Go to GitHub Settings → Developer settings → Personal access tokens → Tokens (classic)
   - Generate a new token with `read:packages` permission

2. Login to Docker:
```bash
echo $GITHUB_TOKEN | docker login ghcr.io -u USERNAME --password-stdin
```

Or interactively:
```bash
docker login ghcr.io
# Username: your-github-username
# Password: your-personal-access-token
```

### Using GitHub CLI (gh)

```bash
gh auth token | docker login ghcr.io -u USERNAME --password-stdin
```

### Public Packages

If you want to make packages public (no authentication required):
1. Go to your repository on GitHub
2. Navigate to Packages → `obiente/cloud-api`
3. Click "Package settings"
4. Scroll down to "Danger Zone" → "Change visibility"
5. Select "Public"

## Deployment

### Using Registry Images (Recommended)

By default, the deployment scripts pull images from ghcr.io:

```bash
# Make sure you're authenticated
docker login ghcr.io

# Deploy using registry images
./scripts/deploy-swarm.sh
```

### Using Local Builds

To build images locally instead:

```bash
BUILD_LOCAL=true ./scripts/deploy-swarm.sh
```

Or set a custom image:

```bash
API_IMAGE=ghcr.io/obiente/cloud-api:v1.0.0 ./scripts/deploy-swarm.sh
```

### Docker Compose

The docker-compose files support the `API_IMAGE` environment variable:

```bash
# Use registry image (default)
API_IMAGE=ghcr.io/obiente/cloud-api:latest docker stack deploy -c docker-compose.swarm.yml obiente

# Use local build
API_IMAGE=obiente/cloud-api:latest docker stack deploy -c docker-compose.swarm.yml obiente
```

## CI/CD Workflow

The GitHub Actions workflow (`.github/workflows/docker-build.yml`) automatically:
- Builds the Docker image with BuildKit
- Pushes to ghcr.io on pushes to main
- Creates tags for version releases
- Uses GitHub Actions cache for faster builds
- Supports multi-platform builds (currently linux/amd64)

## Manual Image Push

To manually push a locally built image:

```bash
# Build locally
docker build -f apps/api/Dockerfile -t ghcr.io/obiente/cloud-api:latest .

# Login
docker login ghcr.io

# Push
docker push ghcr.io/obiente/cloud-api:latest
```

## Troubleshooting

### Authentication Issues

If you get authentication errors:
1. Make sure you're logged in: `docker login ghcr.io`
2. Check your token has `read:packages` scope
3. Verify the package is public or you have access

### Image Not Found

If the image doesn't exist:
1. Check the workflow ran successfully in GitHub Actions
2. Verify the package exists at https://github.com/orgs/obiente/packages
3. Make sure you're using the correct image name and tag

### Permission Denied

If you get permission errors:
1. Ensure your GitHub token has the right permissions
2. For private packages, make sure you have access to the repository
3. Consider making the package public if appropriate

