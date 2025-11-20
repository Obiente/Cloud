# Deploying with .env File

Docker Swarm doesn't automatically load `.env` files like `docker-compose` does. However, the deploy script handles this automatically.

## How Environment Variables Work in Docker Swarm

**Important**: Environment variables are substituted on the **manager node** before deployment:

1. The deploy script loads variables from `.env` into the shell environment
2. Docker Compose processes the compose file and substitutes `${VARIABLE:-default}` with actual values
3. The **final service definition** (with resolved values) is sent to the Swarm
4. The Swarm distributes this definition to **all nodes** (managers and workers)

**Result**: Worker nodes receive the actual environment variable values, not variable references. The `.env` file only needs to exist on the manager node where you run the deploy command.

## Quick Start

### 1. Create your `.env` file

```bash
# Copy the example file
cp .env.example .env

# Edit with your configuration
nano .env
```

### 2. Deploy using the script (recommended)

The deploy script automatically loads `.env` files:

```bash
./scripts/deploy-swarm.sh
```

Or specify a custom stack name and compose file:

```bash
./scripts/deploy-swarm.sh obiente docker-compose.swarm.yml
```

### 3. Deploy manually with environment variables

If you prefer to deploy manually:

```bash
# Export variables from .env file
export $(cat .env | grep -v '^#' | xargs)

# Deploy the stack
docker stack deploy -c docker-compose.swarm.yml obiente
```

Or use a one-liner:

```bash
set -a && source .env && set +a && docker stack deploy -c docker-compose.swarm.yml obiente
```

## Environment Variable Loading

The `deploy-swarm.sh` script automatically:
- Checks for `.env` file in the current directory
- Loads all environment variables from it
- Exports them so `docker stack deploy` can use them

## Important Notes

1. **Never commit `.env` files** to version control
   - Add `.env` to `.gitignore`
   - Use `.env.example` as a template

2. **Variable substitution**: Docker Compose files use `${VARIABLE:-default}` syntax
   - If variable is set, it uses that value
   - If not set, it uses the default value

3. **Priority**: Environment variables override defaults
   - Command line exports > `.env` file > defaults in compose file

## Example .env Configuration

See `.env.example` for a complete list of available variables.

### Minimal Configuration

```bash
# Required
POSTGRES_PASSWORD=your-secure-password
DOMAIN=yourdomain.com

# Authentication
ZITADEL_URL=https://auth.yourdomain.com

# DNS (required for deployments and game servers)
NODE_IPS="us-east-1:1.2.3.4"
```

## Verifying Configuration

After deployment, verify environment variables are set on **all nodes**:

### On Manager Node

```bash
# Check service environment variables
docker service inspect obiente_api --pretty | grep -A 20 Environment

# Or check a running container (on any node)
docker exec $(docker ps -q -f name=obiente_api) env | grep POSTGRES
```

### On Worker Nodes

You can verify variables are set on worker nodes by checking a container running there:

```bash
# SSH to a worker node
ssh worker-node-1

# Find a container running on this node
docker ps | grep obiente_api

# Check environment variables
docker exec <container-id> env | grep POSTGRES
docker exec <container-id> env | grep DOMAIN
```

All nodes should show the same environment variable values from your `.env` file.

## Updating Configuration

To update environment variables:

1. Edit your `.env` file
2. Redeploy the stack:

```bash
./scripts/deploy-swarm.sh
```

Docker Swarm will perform a rolling update automatically.

## Troubleshooting

### Variables not taking effect

1. Ensure `.env` file exists in the project root
2. Check variable names match exactly (case-sensitive)
3. Verify no spaces around `=` in `.env` file
4. Restart services: `docker service update --force obiente_api`

### Comments in .env file

The deploy script handles comments (lines starting with `#`) automatically.

### Multi-line values

For multi-line values, use quotes:

```bash
NODE_IPS="region1:ip1,ip2;region2:ip3,ip4"
```

