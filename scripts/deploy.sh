#!/usr/bin/env bash
#
# Deploy the bot to the server: pull the latest branch, rebuild the bot
# container, and print HEAD, container status, and recent logs.
#
# Config via environment (defaults match the current stand):
#   DEPLOY_SERVER    ssh target            (default: root@146.103.104.106)
#   DEPLOY_DIR       app dir on server     (default: /opt/irregular-verbs-tgbot)
#   DEPLOY_BRANCH    branch to deploy       (default: main)
#   DEPLOY_PASSWORD  ssh password          (optional; needs `sshpass`. Read from
#                                            the repo .env if not set here.
#                                            Prefer SSH keys and leave it empty.)
#
# Usage:
#   scripts/deploy.sh                 # SSH key (recommended), deploys main
#   DEPLOY_PASSWORD='...' scripts/deploy.sh
#   DEPLOY_BRANCH=feat/x scripts/deploy.sh

set -euo pipefail

REPO_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"

# Fall back to DEPLOY_PASSWORD from the repo's .env (gitignored) when it is not
# already set in the environment.
if [[ -z "${DEPLOY_PASSWORD:-}" && -f "$REPO_ROOT/.env" ]]; then
  DEPLOY_PASSWORD="$(grep -E '^DEPLOY_PASSWORD=' "$REPO_ROOT/.env" | tail -n1 | cut -d= -f2-)"
fi

SERVER="${DEPLOY_SERVER:-root@146.103.104.106}"
APP_DIR="${DEPLOY_DIR:-/opt/irregular-verbs-tgbot}"
BRANCH="${DEPLOY_BRANCH:-main}"

# Build the ssh command. With DEPLOY_PASSWORD set, use sshpass; otherwise rely
# on an SSH key or interactive auth.
ssh_base=(ssh -o StrictHostKeyChecking=accept-new -o ConnectTimeout=20 "$SERVER")
if [[ -n "${DEPLOY_PASSWORD:-}" ]]; then
  if ! command -v sshpass >/dev/null 2>&1; then
    echo "error: DEPLOY_PASSWORD is set but 'sshpass' is not installed (brew install sshpass)" >&2
    exit 1
  fi
  export SSHPASS="$DEPLOY_PASSWORD"
  ssh_base=(sshpass -e "${ssh_base[@]}")
fi

echo "▶ Deploying '$BRANCH' to $SERVER:$APP_DIR"

# The remote script. $APP_DIR / $BRANCH expand locally; everything else runs
# on the server.
"${ssh_base[@]}" bash -euo pipefail -s <<EOF
cd "$APP_DIR"
echo "--- pull ---"
git fetch --quiet origin
git checkout --quiet "$BRANCH"
git pull --ff-only
echo "--- ensure shared network ---"
docker network inspect geoirb_network >/dev/null 2>&1 || docker network create geoirb_network
echo "--- rebuild bot ---"
docker compose up -d --build bot
echo "--- HEAD ---"
git rev-parse --short HEAD
echo "--- containers ---"
docker compose ps --format '{{.Service}} {{.Status}}'
echo "--- bot logs ---"
docker compose logs --tail=6 bot
EOF

echo "✔ Deploy finished"
