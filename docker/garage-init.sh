#!/usr/bin/env bash
# One-time Garage cluster initialization.
# Run after `docker compose up -d`.
set -euo pipefail

COMPOSE="docker compose exec -T garage"
GARAGE="$COMPOSE /garage -c /etc/garage.toml"

echo "=== Waiting for Garage RPC ==="
for i in $(seq 1 30); do
	if $GARAGE status >/dev/null 2>&1; then
		echo "[OK] Garage RPC is ready"
		break
	fi
	echo "  ...waiting ($i/30)"
	sleep 2
done

echo ""
echo "=== Initializing Garage single-node cluster ==="

NODE_ID=$($GARAGE node id 2>/dev/null | head -1 | cut -d'@' -f1)
echo "Node ID: $NODE_ID"

echo "Assigning layout..."
$GARAGE layout assign "$NODE_ID" -z dc1 -c 10G

LAYOUT_VER=$($GARAGE layout show 2>/dev/null | grep 'Cluster layout version' | grep -o '[0-9]\+' || echo 0)
NEW_VER=$((LAYOUT_VER + 1))
echo "Applying layout version $NEW_VER..."
$GARAGE layout apply --version "$NEW_VER"

echo ""
echo "Creating bucket 'petrosync'..."
$GARAGE bucket create petrosync 2>/dev/null || echo "[OK] already exists"

echo "Creating API key 'petrosync-key'..."
$GARAGE key create petrosync-key 2>/dev/null || echo "[OK] already exists"

echo "Granting read/write on bucket..."
$GARAGE bucket allow petrosync --read --write --key petrosync-key

echo ""
echo "============================================"
echo "  Garage S3 credentials → copy to .env"
echo "============================================"
$GARAGE key info petrosync-key
echo ""
echo "============================================"
echo "  GARAGE_ENDPOINT=http://localhost:3900"
echo "  GARAGE_BUCKET=petrosync"
echo "============================================"
