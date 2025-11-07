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
COPY pnpm-workspace.yaml pnpm-lock.yaml package.json ./

# Copy package.json files from all workspace packages (for dependency resolution)
# This layer will be cached unless package.json files change
COPY packages ./packages
COPY apps/dashboard/package.json ./apps/dashboard/
COPY tools ./tools

# Use BuildKit cache mount for pnpm store (persists across builds)
# This layer is cached unless lockfile or package.json files change
# Clear pnpm store cache if corrupted and retry install in case of registry issues
RUN --mount=type=cache,id=pnpm-store,target=/root/.local/share/pnpm/store \
    pnpm store prune || true && \
    pnpm install --frozen-lockfile --fetch-timeout=60000 || \
    (echo "First install attempt failed, clearing cache and retrying..." && \
     pnpm store prune && \
     pnpm install --frozen-lockfile --fetch-timeout=60000)

# Copy TypeScript config files needed for builds
COPY nx.json tsconfig.base.json tsconfig.json ./

# Copy source code for workspace packages (needed for building dependencies)
# This layer invalidates when source code changes (but dependencies are already installed)
# Note: packages were already copied above for package.json resolution, 
# but we ensure they're up-to-date here for the build
COPY apps/dashboard ./apps/dashboard

ENV NODE_OPTIONS="--max-old-space-size=8192"

# Build workspace dependencies if needed (this uses Nx to build dependencies)
# Note: Proto files should already be built beforehand - they are not rebuilt here
# Set NODE_OPTIONS directly in the command to ensure it's used
RUN --mount=type=cache,id=pnpm-store,target=/root/.local/share/pnpm/store \
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
RUN --mount=type=cache,id=nuxt-build,target=/app/apps/dashboard/.nuxt \
    --mount=type=cache,id=nuxt-nitro,target=/app/apps/dashboard/.nitro \
    sh -c "NODE_OPTIONS='--max-old-space-size=8192' cd apps/dashboard && pnpm exec nuxt prepare || true" && \
    NODE_OPTIONS="--max-old-space-size=8192" SKIP_TYPE_CHECK=true pnpm --filter "@obiente/dashboard" exec nx nuxt:build

# Production image
FROM node:20-alpine AS runner
WORKDIR /app

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

# Start the application
WORKDIR /app/apps/dashboard
CMD ["node", ".output/server/index.mjs"]

