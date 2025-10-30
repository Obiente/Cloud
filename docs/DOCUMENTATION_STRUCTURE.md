# Documentation Structure

This document explains the organization of the Obiente Cloud documentation.

## Directory Structure

```
docs/
├── README.md                    # Main documentation index
├── getting-started/             # Getting started guides
│   ├── index.md                # Getting started overview
│   ├── installation.md         # Installation guide
│   ├── development.md          # Development setup
│   └── configuration.md        # Configuration guide
├── architecture/                # Architecture documentation
│   ├── index.md                # Architecture overview page
│   ├── overview.md             # System architecture
│   ├── components.md           # Component details (coming soon)
│   └── deployment-model.md     # Deployment model (coming soon)
├── deployment/                  # Deployment guides
│   ├── index.md                # Deployment options
│   ├── docker-compose.md       # Docker Compose setup (coming soon)
│   ├── docker-swarm.md         # Docker Swarm deployment
│   └── high-availability.md    # HA setup (coming soon)
├── guides/                      # How-to guides
│   ├── index.md                # Guides overview
│   ├── authentication.md       # Zitadel authentication setup
│   ├── routing.md              # Traffic routing and domains
│   └── troubleshooting.md      # Common issues
├── self-hosting/               # Self-hosting guides
│   ├── index.md                # Self-hosting overview
│   ├── requirements.md         # Requirements (coming soon)
│   ├── configuration.md        # Configuration (coming soon)
│   └── upgrading.md            # Upgrade guide (coming soon)
└── reference/                   # Reference documentation
    ├── index.md                # Reference overview
    └── environment-variables.md # Environment variables
```

## Documentation Philosophy

### Target Audiences

1. **Self-Hosters** - Running Obiente Cloud at home
2. **Developers** - Contributing to Obiente Cloud
3. **DevOps Engineers** - Deploying in production
4. **End Users** - Using Obiente Cloud to deploy apps

### Organization Principles

1. **Progressive Disclosure** - Start simple, go deep
2. **Cross-Linked** - All docs link to related content
3. **Wiki-Style** - Easy navigation between topics
4. **Multiple Paths** - Different entry points for different users

## Navigation

### Getting Started

New users should start here:

1. [Installation Guide](getting-started/installation.md)
2. [Configuration Guide](getting-started/configuration.md)
3. [Architecture Overview](architecture/overview.md)

### Self-Hosting

For self-hosting enthusiasts:

1. [Self-Hosting Guide](self-hosting/index.md)
2. [Requirements](self-hosting/requirements.md)
3. [Deployment Guide](deployment/index.md)

### Production Deployment

For production deployments:

1. [Deployment Methods](deployment/index.md)
2. [High Availability](deployment/high-availability.md)
3. [Monitoring Guide](guides/monitoring.md)

## Contributing to Documentation

When adding new documentation:

1. **Choose the right location** - Place in appropriate section
2. **Create an index entry** - Add to the section's index.md
3. **Cross-link** - Link to related documentation
4. **Update main index** - Add to docs/README.md if needed
5. **Follow the style** - Use consistent formatting

## Markdown Conventions

### Headers

```markdown
# Main Title

## Section

### Subsection
```

### Links

```markdown
[Link Text](path/to/file.md)
[External Link](https://example.com)
[Relative Link](../other-file.md)
```

### Code Blocks

```bash
# Commands
```

```yaml
# Configurations
```

```javascript
// Code examples
```

### Callouts

```markdown
✅ Good practice
❌ Bad practice
⚠️ Warning
💡 Tip
```

## Updating Documentation

When updating documentation:

1. Update the relevant file
2. Check cross-links still work
3. Update table of contents if needed
4. Test all code examples

---

[← Back to Documentation](README.md)
