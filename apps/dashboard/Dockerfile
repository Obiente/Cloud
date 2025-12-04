# syntax=docker/dockerfile:1.4
FROM node:22-alpine AS base

# Install pnpm in a single layer
RUN corepack enable && \
    COREPACK_ENABLE_STRICT=0 corepack prepare pnpm@10.20.0 --activate

WORKDIR /app

# Dependencies stage - optimized for caching
FROM base AS deps
# Copy only files needed for dependency resolution (changes infrequently)
# Copy workspace root files first - this layer is cached unless lockfile changes
COPY pnpm-workspace.yaml pnpm-lock.yaml package.json ./
# Copy package.json files from packages (for workspace dependency resolution)
# Copy entire packages directory but .dockerignore excludes source files
# This is still needed for workspace resolution, but .dockerignore minimizes what's copied
COPY packages ./packages
COPY apps/dashboard/package.json ./apps/dashboard/
COPY tools ./tools

# Install dependencies with aggressive caching
# Use BuildKit cache mounts for pnpm store
RUN --mount=type=cache,id=pnpm-store,target=/root/.local/share/pnpm/store \
    pnpm install --frozen-lockfile --prefer-offline

# Copy config files needed for builds
COPY nx.json tsconfig.base.json tsconfig.json ./

# Copy source code (this layer invalidates when source changes)
# Copy in order of change frequency (least frequent first for better caching)
# Copy packages source (needed for dashboard build, changes less frequently)
COPY packages ./packages
# Copy tools source (needed for dashboard build, changes less frequently)
COPY tools ./tools
# Copy dashboard source last (changes most frequently, invalidates this layer most often)
COPY apps/dashboard ./apps/dashboard

# Set Node options once
# Increase memory for large builds and add timeout protection
ENV NODE_OPTIONS="--max-old-space-size=8192 --no-warnings" \
    SKIP_TYPE_CHECK=true \
    NITRO_PRESET=node-server \
    NODE_ENV=production

# Build workspace dependencies if needed
# Skip build-deps if packages are already built (faster)
# Only build if package.json files changed or if explicitly needed
RUN --mount=type=cache,id=pnpm-store,target=/root/.local/share/pnpm/store \
    (pnpm --filter "@obiente/dashboard" build-deps 2>/dev/null || echo "build-deps skipped or not needed") || true

# Build stage - reuse everything from deps
FROM deps AS builder

# Build the application with optimized cache mounts
# Use all available cache mounts for maximum speed
# Skip nuxt prepare if .nuxt directory exists and is valid (faster)
# Only run prepare if needed (when source files changed significantly)
RUN --mount=type=cache,id=nuxt-build,target=/app/apps/dashboard/.nuxt \
    --mount=type=cache,id=nuxt-nitro,target=/app/apps/dashboard/.nitro \
    --mount=type=cache,id=vite-cache,target=/app/apps/dashboard/node_modules/.vite \
    --mount=type=cache,id=pnpm-store,target=/root/.local/share/pnpm/store \
    cd apps/dashboard && \
    (test -d .nuxt && test -f .nuxt/nuxt.d.ts && echo "Skipping nuxt prepare (cache hit)" || pnpm exec nuxt prepare) && \
    echo "Starting Nuxt build (client + server)..." && \
    NUXT_TELEMETRY_DISABLED=1 \
    NITRO_PRESET=node-server \
    NODE_OPTIONS="--max-old-space-size=8192" \
    pnpm exec nuxt build

# Production image - minimal runtime
FROM node:20-alpine AS runner

# Accept build arguments
ARG DASHBOARD_COMMIT
ARG DASHBOARD_COMMIT_MESSAGE

# Create user in a single layer
RUN addgroup --system --gid 1001 nodejs && \
    adduser --system --uid 1001 nuxtjs

WORKDIR /app

# Copy only what's needed for runtime (minimize image size)
# Only copy node_modules if runtime dependencies are needed
# Most Nuxt apps bundle everything, so we might not need node_modules
COPY --from=builder --chown=nuxtjs:nodejs /app/apps/dashboard/.output ./apps/dashboard/.output

# Set environment variables
ENV NODE_ENV=production \
    HOST=0.0.0.0 \
    PORT=3000 \
    DASHBOARD_COMMIT=${DASHBOARD_COMMIT} \
    DASHBOARD_COMMIT_MESSAGE=${DASHBOARD_COMMIT_MESSAGE}

# Switch to non-root user
USER nuxtjs

EXPOSE 3000

WORKDIR /app/apps/dashboard
CMD ["node", ".output/server/index.mjs"]

