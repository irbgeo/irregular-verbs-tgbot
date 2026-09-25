#!/usr/bin/env bash
# Shared helper: read the deploy target from secret/server-access.yaml (same
# format as blacks-law-bot) — one top-level "<server-name>:" block with
# indented "key: value" pairs. Example:
#
#   geoirb-bots:
#     host: 34.88.50.39
#     user: ai
#     password: change-me      # optional; prefer SSH keys
#     deploy_dir: /opt/irregular-verbs-tgbot
#
# ponytail: the parser doesn't handle quoted strings, multi-line values, or
# a ':' inside a value. Fine for four flat keys per server; swap in a real
# YAML parser if the schema ever grows beyond that.

LIB_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
SERVER_ACCESS_FILE="${SERVER_ACCESS_FILE:-$LIB_DIR/../secret/server-access.yaml}"
DEFAULT_DEPLOY_DIR=/opt/irregular-verbs-tgbot

# resolve_deploy_target [server-name] — sets SERVER (ssh target), APP_DIR and
# DEPLOY_PASSWORD from the named block of SERVER_ACCESS_FILE. The name can be
# omitted when the file has exactly one block. DEPLOY_SERVER, DEPLOY_DIR and
# DEPLOY_PASSWORD from the environment win over the yaml; the yaml may be
# missing only when DEPLOY_SERVER is set and no name is passed.
resolve_deploy_target() {
  local name="${1:-}" host="" user="" password="" dir=""
  if [[ -f "$SERVER_ACCESS_FILE" ]]; then
    if [[ -z "$name" ]]; then
      name="$(yaml_servers)"
      if [[ "$(grep -c . <<< "$name")" -ne 1 ]]; then
        echo "error: $SERVER_ACCESS_FILE must have exactly one server, or pass a name: $(tr '\n' ' ' <<< "$name")" >&2
        return 1
      fi
    fi
    host="$(yaml_value "$name" host)"
    user="$(yaml_value "$name" user)"
    password="$(yaml_value "$name" password)"
    dir="$(yaml_value "$name" deploy_dir)"
    if [[ -z "$host" || -z "$user" ]]; then
      echo "error: host or user missing for '$name' in $SERVER_ACCESS_FILE" >&2
      return 1
    fi
  elif [[ -n "$name" || -z "${DEPLOY_SERVER:-}" ]]; then
    echo "error: $SERVER_ACCESS_FILE not found (or set DEPLOY_SERVER)" >&2
    return 1
  fi
  SERVER="${DEPLOY_SERVER:-$user@$host}"
  APP_DIR="${DEPLOY_DIR:-${dir:-$DEFAULT_DEPLOY_DIR}}"
  DEPLOY_PASSWORD="${DEPLOY_PASSWORD:-$password}"
}

# yaml_servers — prints the names of all top-level blocks.
yaml_servers() {
  awk 'match($0, /^[A-Za-z0-9_.-]+:[[:space:]]*$/) {
    s = $0
    sub(/:[[:space:]]*$/, "", s)
    print s
  }' "$SERVER_ACCESS_FILE"
}

# yaml_value <server> <key> — prints one key's value from the named block,
# stripping any inline "# comment".
yaml_value() {
  awk -v server="$1:" -v key="$2" '
    match($0, "^" server "[[:space:]]*$") { block = 1; next }
    block && match($0, /^[A-Za-z0-9_.-]+:[[:space:]]*$/) { block = 0 }
    block && match($0, "^[[:space:]]+" key "[[:space:]]*:[[:space:]]*") {
      val = substr($0, RSTART + RLENGTH)
      sub(/[[:space:]]+#.*$/, "", val)
      print val
      exit
    }
  ' "$SERVER_ACCESS_FILE"
}
