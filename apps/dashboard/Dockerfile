# syntax=docker/dockerfile:1.4
FROM node:22-alpine AS base

# Install pnpm
# Use pnpm@10.20.0 to match workspace root package.json
# Disable strict signature verification to avoid corepack signature issues
RUN corepack enable && \
    COREPACK_ENABLE_STRICT=0 corepack prepare pnpm@10.20.0 --activate

# Set working directory
WORKDIR /app

# Install dependencies only when needed
FROM base AS deps
# Copy workspace configuration files first (for better caching)
# These files change infrequently, so this layer will be cached most of the time
COPY pnpm-workspace.yaml pnpm-lock.yaml package.json ./

# Copy package.json files from all workspace packages (for dependency resolution)
# This layer will be cached unless package.json files change
COPY packages ./packages
COPY apps/dashboard/package.json ./apps/dashboard/
COPY tools ./tools

# Use BuildKit cache mount for pnpm store (persists across builds)
# This layer is cached unless lockfile or package.json files change
# The pnpm store cache significantly speeds up installs when dependencies haven't changed
RUN --mount=type=cache,id=pnpm-store,target=/root/.local/share/pnpm/store \
    pnpm store prune || true && \
    pnpm install --frozen-lockfile --prefer-offline --fetch-timeout=60000 || \
    (echo "First install attempt failed, clearing cache and retrying..." && \
     pnpm store prune && \
     pnpm install --frozen-lockfile --prefer-offline --fetch-timeout=60000)

# Copy TypeScript config files needed for builds
COPY nx.json tsconfig.base.json tsconfig.json ./

# Copy source code for workspace packages (needed for building dependencies)
# This layer invalidates when source code changes (but dependencies are already installed)
# Note: packages were already copied above for package.json resolution, 
# but we ensure they're up-to-date here for the build
COPY apps/dashboard ./apps/dashboard
COPY packages ./packages
COPY tools ./tools

ENV NODE_OPTIONS="--max-old-space-size=8192"

# Build workspace dependencies if needed (this uses Nx to build dependencies)
# Note: Proto files should already be built beforehand - they are not rebuilt here
# Cache Nx computation cache for faster builds (TypeScript build info is handled by Nx and Docker layer caching)
RUN --mount=type=cache,id=pnpm-store,target=/root/.local/share/pnpm/store \
    --mount=type=cache,id=nx-cache,target=/app/.nx/cache \
    NODE_OPTIONS="--max-old-space-size=8192" pnpm --filter "@obiente/dashboard" build-deps || true

# Build stage
FROM base AS builder
WORKDIR /app

# Increase Node.js heap size to prevent out-of-memory errors during build
# 8GB should be sufficient
ENV NODE_OPTIONS="--max-old-space-size=8192"

# Copy dependencies from deps stage
COPY --from=deps /app/node_modules ./node_modules
COPY --from=deps /app/apps/dashboard/node_modules ./apps/dashboard/node_modules

# Copy config files
COPY --from=deps /app/nx.json ./nx.json
COPY --from=deps /app/tsconfig.base.json ./tsconfig.base.json
COPY --from=deps /app/tsconfig.json ./tsconfig.json

# Copy workspace configuration files for pnpm workspace resolution
COPY --from=deps /app/package.json ./package.json
COPY --from=deps /app/pnpm-workspace.yaml ./pnpm-workspace.yaml
COPY --from=deps /app/pnpm-lock.yaml ./pnpm-lock.yaml

# Copy source code from deps stage (already includes packages and dashboard)
COPY --from=deps /app/apps/dashboard ./apps/dashboard
COPY --from=deps /app/packages ./packages
COPY --from=deps /app/tools ./tools

# Build the application with cache mounts for build artifacts
# Run from workspace root - pnpm exec will find nx in root node_modules/.bin
# First ensure nuxt prepare runs to generate .nuxt directory (needed for tsconfig files)
# Skip type checking during Docker build for faster builds (can be done in CI)
# Set NODE_OPTIONS directly in the commands to ensure it's used by all child processes
# Cache multiple build artifacts for maximum speed:
# - .nuxt: Nuxt build cache (persists across builds)
# - .nitro: Nitro build cache (persists across builds)
# - .vite: Vite dependency pre-bundling cache (speeds up dependency processing)
# - Nx cache: Nx computation cache (speeds up task execution)
# TypeScript build info is handled by Docker layer caching and Nx cache
RUN --mount=type=cache,id=nuxt-build,target=/app/apps/dashboard/.nuxt \
    --mount=type=cache,id=nuxt-nitro,target=/app/apps/dashboard/.nitro \
    --mount=type=cache,id=vite-cache,target=/app/apps/dashboard/node_modules/.vite \
    --mount=type=cache,id=nx-cache,target=/app/.nx/cache \
    sh -c "NODE_OPTIONS='--max-old-space-size=8192' cd apps/dashboard && pnpm exec nuxt prepare || true" && \
    NODE_OPTIONS="--max-old-space-size=8192" SKIP_TYPE_CHECK=true pnpm --filter "@obiente/dashboard" exec nx nuxt:build

# Production image
FROM node:20-alpine AS runner
WORKDIR /app

# Accept build arguments for commit hash and message
ARG DASHBOARD_COMMIT
ARG DASHBOARD_COMMIT_MESSAGE

# Don't run as root
RUN addgroup --system --gid 1001 nodejs && \
    adduser --system --uid 1001 nuxtjs

# Copy only production dependencies (if needed)
COPY --from=deps --chown=nuxtjs:nodejs /app/node_modules ./node_modules

# Copy built application
COPY --from=builder --chown=nuxtjs:nodejs /app/apps/dashboard/.output ./apps/dashboard/.output

# Switch to non-root user
USER nuxtjs

# Expose port
EXPOSE 3000

# Set environment variables
ENV NODE_ENV=production
ENV HOST=0.0.0.0
ENV PORT=3000
# Set commit hash and message from build args (available at runtime via process.env)
ENV DASHBOARD_COMMIT=${DASHBOARD_COMMIT}
ENV DASHBOARD_COMMIT_MESSAGE=${DASHBOARD_COMMIT_MESSAGE}

# Start the application
WORKDIR /app/apps/dashboard
CMD ["node", ".output/server/index.mjs"]

