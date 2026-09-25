#!/usr/bin/env bash
#
# Deploy the bot to the server — the same command for the first deploy and
# every update:
#   1. create the app dir on the server (sudo once, if missing);
#   2. upload the committed code of DEPLOY_BRANCH (git archive — no git or
#      GitHub deploy key needed on the server; uncommitted changes are NOT sent);
#   3. write the server's .env = the local .env minus DEPLOY_* keys;
#   4. rebuild and start the bot, then print status and recent logs.
#
# Needs on the server: Docker, and server-infra's Mongo up on geoirb_network
# with this bot's Mongo user (MONGO_USERNAME/MONGO_PASSWORD in the local .env).
#
# Where to deploy (host, user, password, dir) comes from
# secret/server-access.yaml (gitignored; format in scripts/lib.sh). Pass the
# server block's name as the first argument when the file has more than one.
# Environment overrides:
#   DEPLOY_SERVER    ssh target            (instead of user@host from the yaml)
#   DEPLOY_DIR       app dir on server     (default: deploy_dir, else /opt/irregular-verbs-tgbot)
#   DEPLOY_BRANCH    local branch to ship  (default: main)
#   DEPLOY_PASSWORD  ssh/sudo password     (default: password from the yaml;
#                                           needs `sshpass`. Prefer SSH keys.)
#
# Usage:
#   scripts/deploy.sh                 # the only server in the yaml, ships main
#   scripts/deploy.sh geoirb-bots     # a named server block
#   DEPLOY_BRANCH=feat/x scripts/deploy.sh

set -euo pipefail

REPO_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
source "$REPO_ROOT/scripts/lib.sh"
resolve_deploy_target "${1:-}"
BRANCH="${DEPLOY_BRANCH:-main}"
COMMIT="$(git -C "$REPO_ROOT" rev-parse --short "$BRANCH")"

ENV_CONTENT="$(grep -vE '^DEPLOY_' "$REPO_ROOT/.env" 2>/dev/null || true)"
if [[ -z "$ENV_CONTENT" ]]; then
  echo "error: $REPO_ROOT/.env is missing or has no bot vars (BOT_TOKEN, MONGO_USERNAME, MONGO_PASSWORD)" >&2
  exit 1
fi

# No -tt anywhere: secrets (sudo password, .env) go through stdin, and a
# pseudo-terminal would echo them back to this terminal.
ssh_cmd=(ssh -o StrictHostKeyChecking=accept-new -o ConnectTimeout=20 "$SERVER")
if [[ -n "${DEPLOY_PASSWORD:-}" ]]; then
  if ! command -v sshpass >/dev/null 2>&1; then
    echo "error: DEPLOY_PASSWORD is set but 'sshpass' is not installed (brew install sshpass)" >&2
    exit 1
  fi
  export SSHPASS="$DEPLOY_PASSWORD"
  ssh_cmd=(sshpass -e "${ssh_cmd[@]}")
fi

echo "▶ Deploying '$BRANCH' ($COMMIT) to $SERVER:$APP_DIR"

# 1. App dir. /opt is root-owned, so the first deploy needs sudo once; the
# password goes to sudo -S on stdin. Skipped once the dir exists.
# shellcheck disable=SC2016 # $u expands on the server
printf '%s\n' "${DEPLOY_PASSWORD:-}" | "${ssh_cmd[@]}" \
  "if [ ! -d '$APP_DIR' ]; then echo '--- creating $APP_DIR ---'; u=\$(whoami); sudo -S -p '' sh -c \"mkdir -p '$APP_DIR' && chown \$u:\$u '$APP_DIR'\"; fi"

# 2. Code. Everything in the dir except .env is replaced, so files deleted
# from the repo don't linger on the server.
echo "--- uploading code ---"
git -C "$REPO_ROOT" archive --format=tar "$BRANCH" | "${ssh_cmd[@]}" \
  "cd '$APP_DIR' && find . -mindepth 1 -maxdepth 1 ! -name .env -exec rm -rf {} + && tar -xf -"

# 3. .env, readable by the owner only.
echo "--- writing .env ---"
printf '%s\n' "$ENV_CONTENT" | "${ssh_cmd[@]}" "umask 077 && cat > '$APP_DIR/.env'"

# 4. Build and start. Build cache older than a week and dangling images are
# pruned: nothing else removes them, and they filled the old server's disk.
"${ssh_cmd[@]}" bash -euo pipefail -s <<EOF
cd '$APP_DIR'
echo "--- ensure shared network ---"
docker network inspect geoirb_network >/dev/null 2>&1 || docker network create geoirb_network
echo "--- rebuild bot ---"
docker compose up -d --build bot
echo "--- prune old build cache and images ---"
docker builder prune -f --filter until=168h >/dev/null || echo 'warning: build-cache prune failed'
docker image prune -f >/dev/null || echo 'warning: image prune failed'
echo "--- containers ---"
docker compose ps --format '{{.Service}} {{.Status}}'
echo "--- bot logs ---"
sleep 3
docker compose logs --tail=10 bot
EOF

echo "✔ Deployed $COMMIT"
