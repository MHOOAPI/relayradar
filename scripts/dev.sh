#!/usr/bin/env sh
set -eu
cd "$(dirname "$0")/.."

ours=$(docker ps --filter 'name=relayradar' --format '{{.Ports}}' 2>/dev/null || true)
conflicts=""
for port in 80 443 8088; do
  if command -v ss >/dev/null 2>&1; then
    in_use=$(ss -ltn "sport = :$port" 2>/dev/null | tail -n +2)
  else
    in_use=$(netstat -ltn 2>/dev/null | grep -E "[:.]$port " || true)
  fi
  [ -z "$in_use" ] && continue
  echo "$ours" | grep -q ":$port->" && continue
  conflicts="$conflicts  port $port is in use (not RelayRadar)\n"
done
if [ -n "$conflicts" ]; then
  printf "Port conflict:\n$conflicts"
  echo "Stop that process, or change the port mappings in docker-compose.yml."
  exit 1
fi

if [ ! -f .env ]; then
  cp .env.example .env
  pw=$(head -c 24 /dev/urandom | base64 | tr -dc 'A-Za-z0-9' | head -c 24)
  sed -i "s|^POSTGRES_PASSWORD=.*|POSTGRES_PASSWORD=$pw|" .env
  sed -i "s|^PUBLIC_BASE_URL=.*|PUBLIC_BASE_URL=https://localhost|" .env
  echo "Created .env with a generated database password."
  echo "If a PostgreSQL volume already existed, run: docker compose down -v"
fi

docker compose up -d

i=0
while [ "$i" -lt 45 ]; do
  if curl -k -s -o /dev/null -w '%{http_code}' --max-time 3 https://localhost/ | grep -q 200; then
    echo "RelayRadar is up:"
    echo "  HTTPS  https://localhost/"
    echo "  HTTP   http://localhost:8088/"
    docker compose ps --format '{{.Service}} {{.Status}}'
    exit 0
  fi
  i=$((i + 1))
  sleep 2
done

echo "Stack started but HTTPS did not answer 200 within 90s."
echo "Check: docker compose ps && docker compose logs api nginx"
exit 1
