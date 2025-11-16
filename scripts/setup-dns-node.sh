#!/bin/bash
# Script to set up DNS node labeling for Docker Swarm

set -e

if [ -z "$1" ]; then
    echo "Usage: $0 <node-name>"
    echo ""
    echo "This script labels a node to run the DNS service."
    echo ""
    echo "Example:"
    echo "  $0 docker-node-1"
    echo ""
    echo "To find node names, run: docker node ls"
    exit 1
fi

NODE_NAME=$1

echo "Setting up DNS node: $NODE_NAME"
echo ""

# Check if node exists
if ! docker node inspect "$NODE_NAME" &>/dev/null; then
    echo "Error: Node '$NODE_NAME' not found"
    echo ""
    echo "Available nodes:"
    docker node ls
    exit 1
fi

# Label the node
echo "Labeling node with dns.enabled=true..."
docker node update --label-add dns.enabled=true "$NODE_NAME"

echo ""
echo "✓ Node labeled successfully!"
echo ""
echo "Node labels:"
docker node inspect "$NODE_NAME" --format '{{range $k, $v := .Spec.Labels}}{{$k}}={{$v}}{{"\n"}}{{end}}' | grep dns || echo "  (no dns labels found)"

echo ""
echo "Next steps:"
echo "1. On the DNS node, disable systemd-resolved:"
echo "   sudo sed -i 's/#DNSStubListener=yes/DNSStubListener=no/' /etc/systemd/resolved.conf"
echo "   sudo sed -i 's/DNSStubListener=yes/DNSStubListener=no/' /etc/systemd/resolved.conf"
echo "   sudo systemctl restart systemd-resolved"
echo ""
echo "2. Deploy the stack:"
echo "   ./scripts/deploy-swarm.sh obiente docker-compose.swarm.yml"
echo ""
echo "3. Verify DNS is running on the labeled node:"
echo "   docker service ps obiente_dns"

