#!/usr/bin/env sh
set -eu

URL="${1:-}"
TIMEOUT_SECONDS="${2:-60}"

if [ -z "$URL" ]; then
  printf '%s\n' 'Usage: infra/scripts/wait-for-http.sh <url> [timeout-seconds]' >&2
  exit 2
fi

end_time=$(( $(date +%s) + TIMEOUT_SECONDS ))

while [ "$(date +%s)" -le "$end_time" ]; do
  if node -e "fetch(process.argv[1]).then(function(r){process.exit(r.ok?0:1)}).catch(function(){process.exit(1)})" "$URL"; then
    printf '%s\n' "Ready: $URL"
    exit 0
  fi
  sleep 1
done

printf '%s\n' "Timed out waiting for $URL" >&2
exit 1
