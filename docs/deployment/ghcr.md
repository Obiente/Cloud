# GitHub Container Registry Setup

This repository uses GitHub Container Registry (`ghcr.io`) for Docker image builds and distribution.

## Overview

Docker images for the dashboard and individual backend services are built and pushed by dedicated GitHub Actions workflows on:
- Push to `main` branch
- Push of version tags (e.g., `v1.0.0`)
- Manual workflow dispatch

## Image Locations

### API Gateway Images
- `ghcr.io/obiente/cloud-api-gateway:latest` - Latest build from main branch
- `ghcr.io/obiente/cloud-api-gateway:main` - Latest build from main branch (alternative tag)
- `ghcr.io/obiente/cloud-api-gateway:v1.0.0` - Tagged versions
- `ghcr.io/obiente/cloud-api-gateway:main-<sha>` - Build-specific tags

### Dashboard Images
- `ghcr.io/obiente/cloud-dashboard:latest` - Latest build from main branch
- `ghcr.io/obiente/cloud-dashboard:main` - Latest build from main branch (alternative tag)
- `ghcr.io/obiente/cloud-dashboard:v1.0.0` - Tagged versions
- `ghcr.io/obiente/cloud-dashboard:main-<sha>` - Build-specific tags

### Other Service Images

Other services follow the same pattern, for example:

- `ghcr.io/obiente/cloud-auth-service:latest`
- `ghcr.io/obiente/cloud-deployments-service:latest`
- `ghcr.io/obiente/cloud-vps-service:latest`
- `ghcr.io/obiente/cloud-<service>:<tag>`

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
2. Navigate to Packages → `obiente/cloud-api-gateway`, `obiente/cloud-dashboard`, or another `obiente/cloud-<service>` package
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

Or set a custom registry prefix:

```bash
REGISTRY=ghcr.io/obiente ./scripts/deploy-swarm.sh
```

### Docker Compose

The docker-compose files support environment variables for image selection:

#### API Gateway Images
```bash
# Build and tag the API gateway image explicitly
docker build -f apps/api-gateway/Dockerfile -t ghcr.io/obiente/cloud-api-gateway:latest .
```

#### Dashboard Images
```bash
# Use registry image (default)
DASHBOARD_IMAGE=ghcr.io/obiente/cloud-dashboard:latest docker compose -f docker-compose.dashboard.yml up -d

# Use local build
DASHBOARD_IMAGE=obiente/cloud-dashboard:latest docker compose -f docker-compose.dashboard.yml up -d
```

## CI/CD Workflow

The GitHub Actions workflows in `.github/workflows/` automatically:
- Build individual service images with BuildKit
- Push to `ghcr.io` on pushes to `main`
- Create tags for version releases
- Use GitHub Actions cache for faster builds
- Run separate workflows such as `build-api-gateway.yml`, `build-dashboard.yml`, and service-specific `build-*.yml` files

Example jobs:
- `build-and-push-api-gateway` - Builds and pushes the API gateway image
- `build-and-push-dashboard` - Builds and pushes the Dashboard image

## Manual Image Push

To manually push locally built images:

### API Gateway Image
```bash
# Build locally
docker build -f apps/api-gateway/Dockerfile -t ghcr.io/obiente/cloud-api-gateway:latest .

# Login
docker login ghcr.io

# Push
docker push ghcr.io/obiente/cloud-api-gateway:latest
```

### Dashboard Image
```bash
# Build locally
docker build -f apps/dashboard/Dockerfile -t ghcr.io/obiente/cloud-dashboard:latest .

# Login
docker login ghcr.io

# Push
docker push ghcr.io/obiente/cloud-dashboard:latest
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
   - Look for `obiente/cloud-api-gateway`, `obiente/cloud-dashboard`, or the specific service package you expect
3. Make sure you're using the correct image name and tag

### Permission Denied

If you get permission errors:
1. Ensure your GitHub token has the right permissions
2. For private packages, make sure you have access to the repository
3. Consider making the package public if appropriate
