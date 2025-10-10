# Obiente Cloud Dashboard

Frontend dashboard application for the Obiente Cloud platform built with Nuxt 3.

## Overview

This is the main user interface for the Obiente Cloud multi-tenant dashboard. It provides a modern, responsive web application for managing cloud resources including deployments, VPS instances, databases, and billing.

## Tech Stack

### Core Framework

- **Nuxt 3** - Vue.js meta-framework with SSR/SSG
- **Vue 3** - Progressive JavaScript framework
- **TypeScript** - Type safety throughout the application
- **Pinia** - State management for Vue

### UI & Styling

- **Nuxt UI** - Component library built on Tailwind CSS
- **Tailwind CSS** - Utility-first CSS framework
- **Headless UI** - Unstyled, accessible UI components
- **Heroicons** - Beautiful hand-crafted SVG icons
- **@obiente/ui** - Custom component library with Ark UI

### Development Tools

- **Vue DevTools** - Browser extension for debugging
- **ESLint** - Code linting and formatting
- **TypeScript compiler** - Type checking

## Features

### Multi-tenant Architecture

- Organization-based tenant isolation
- Dynamic organization switching
- Role-based access control (RBAC)

### Authentication & Authorization

- Zitadel OIDC/OAuth2 integration
- JWT token management
- SSO (Single Sign-On) support
- Secure session handling

### Resource Management

- **Deployments**: Web application hosting management
- **VPS Instances**: Virtual private server administration
- **Databases**: Managed database service control
- **Billing**: Subscription and usage tracking

### User Experience

- Responsive design for all devices
- Real-time status updates
- Intuitive navigation and workflows
- Comprehensive search and filtering

## Project Structure

```
├── assets/              # Static assets (CSS, images)
├── components/          # Vue components
├── composables/         # Vue composables (business logic)
├── layouts/             # Nuxt layouts
├── middleware/          # Route middleware
├── pages/               # File-based routing
├── plugins/             # Nuxt plugins
├── stores/              # Pinia stores (state management)
├── types/               # TypeScript type definitions
├── utils/               # Utility functions
├── nuxt.config.ts       # Nuxt configuration
└── package.json         # Dependencies and scripts
```

## Development

### Prerequisites

- Node.js 18+
- pnpm (package manager)

### Environment Setup

1. Copy `.env.example` to `.env`
2. Configure API and authentication endpoints
3. Set up Zitadel OIDC configuration

### Available Scripts

```bash
# Development
pnpm dev              # Start development server
pnpm build            # Build for production
pnpm preview          # Preview production build
pnpm generate         # Generate static site

# Code Quality
pnpm typecheck        # Run TypeScript checks
pnpm lint             # Run ESLint
pnpm lint:fix         # Fix ESLint issues
pnpm clean            # Clean build artifacts
```

### Development Server

The application runs on `http://localhost:3000` by default.

## Configuration

### Runtime Config

Environment variables are managed through Nuxt's runtime config:

```typescript
// nuxt.config.ts
runtimeConfig: {
  // Private (server-side only)
  apiSecret: process.env.API_SECRET,

  // Public (client-side)
  public: {
    apiBaseUrl: process.env.API_BASE_URL || 'http://localhost:3001',
    zitadelUrl: process.env.ZITADEL_URL,
    zitadelClientId: process.env.ZITADEL_CLIENT_ID,
  }
}
```

### Key Environment Variables

- `API_BASE_URL` - Backend API endpoint
- `ZITADEL_URL` - Zitadel authentication server
- `ZITADEL_CLIENT_ID` - OAuth2 client identifier

## Authentication Flow

1. **Login Page**: User enters credentials or uses SSO
2. **OIDC Redirect**: Zitadel handles authentication
3. **Token Exchange**: Receive JWT tokens
4. **Dashboard Access**: Authenticated user accesses resources

## State Management

### Pinia Stores

- **Auth Store**: User authentication and session
- **Organization Store**: Current organization context
- **Resource Stores**: Deployments, VPS, databases state

### Composables

- `useAuth()` - Authentication logic
- `useApi()` - API communication
- `useNotifications()` - User feedback system

## Styling Guidelines

### Tailwind CSS Classes

- Use utility classes for styling
- Custom components in `@obiente/ui` package
- Consistent color palette and spacing

### Component Architecture

- Small, focused components
- Props and events for communication
- TypeScript interfaces for type safety

## Deployment

### Production Build

```bash
pnpm build
pnpm preview
```

### Static Generation

```bash
pnpm generate
```

### Environment Variables

Ensure all required environment variables are set in production.

## Security

### Content Security Policy

- Configured in `nuxt.config.ts`
- Prevents XSS attacks
- Restricts resource loading

### Authentication

- JWT tokens stored securely
- Automatic token refresh
- Secure HTTP-only cookies

## Future Enhancements

1. **Real-time Updates**: WebSocket integration for live status
2. **PWA Support**: Offline capabilities and app-like experience
3. **Advanced Analytics**: Resource usage dashboards and insights
4. **Mobile App**: React Native companion application
5. **Internationalization**: Multi-language support
6. **Advanced Search**: Full-text search across resources
7. **Team Collaboration**: Real-time collaboration features
