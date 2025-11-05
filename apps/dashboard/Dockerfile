# syntax=docker/dockerfile:1.4
FROM node:20-alpine AS base

# Install pnpm
RUN corepack enable && corepack prepare pnpm@10.17.0 --activate

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
RUN --mount=type=cache,id=pnpm-store,target=/root/.local/share/pnpm/store \
    pnpm install --frozen-lockfile

# Copy TypeScript config files needed for builds
COPY nx.json tsconfig.base.json tsconfig.json ./

# Copy source code for workspace packages (needed for building dependencies)
# This layer invalidates when source code changes (but dependencies are already installed)
# Note: packages were already copied above for package.json resolution, 
# but we ensure they're up-to-date here for the build
COPY apps/dashboard ./apps/dashboard

# Build workspace dependencies if needed (this uses Nx to build dependencies)
# Packages source code is already available from the earlier COPY packages command
RUN --mount=type=cache,id=pnpm-store,target=/root/.local/share/pnpm/store \
    pnpm --filter "@obiente/dashboard" build-deps || true

# Build stage
FROM base AS builder
WORKDIR /app

# Copy dependencies from deps stage
COPY --from=deps /app/node_modules ./node_modules
COPY --from=deps /app/apps/dashboard/node_modules ./apps/dashboard/node_modules

# Copy config files
COPY --from=deps /app/nx.json ./nx.json
COPY --from=deps /app/tsconfig.base.json ./tsconfig.base.json
COPY --from=deps /app/tsconfig.json ./tsconfig.json

# Copy source code from deps stage (already includes packages and dashboard)
COPY --from=deps /app/apps/dashboard ./apps/dashboard
COPY --from=deps /app/packages ./packages
COPY --from=deps /app/tools ./tools

# Build the application with cache mounts for build artifacts
RUN --mount=type=cache,id=nuxt-build,target=/app/apps/dashboard/.nuxt \
    --mount=type=cache,id=nuxt-nitro,target=/app/apps/dashboard/.nitro \
    cd apps/dashboard && \
    pnpm run build

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

