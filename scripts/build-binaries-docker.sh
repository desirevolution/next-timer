#!/usr/bin/env bash
set -euo pipefail

root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
output="${1:-${root}/dist}"
mkdir -p "$output"
output="$(cd "$output" && pwd)"

if ! command -v docker >/dev/null 2>&1; then
  echo "Fehler: docker wurde nicht gefunden." >&2
  exit 1
fi

# Use an isolated BuildKit builder for multi-platform binary exports.
# Do not change the user's globally selected Buildx builder.
builder="${NEXTTIMER_BUILDER:-nexttimer-multiarch}"
if docker buildx inspect "$builder" >/dev/null 2>&1; then
  driver="$(docker buildx inspect "$builder" | sed -nE 's/^Driver:[[:space:]]*([^[:space:]]+).*/\1/p' | head -n 1)"
  if [[ "$driver" != "docker-container" ]]; then
    echo "Fehler: Buildx-Builder '$builder' hat Treiber '$driver' statt 'docker-container'." >&2
    echo "Bitte NEXTTIMER_BUILDER auf einen anderen Namen setzen." >&2
    exit 1
  fi
else
  echo "Erstelle Buildx-Builder '$builder' mit docker-container-Treiber ..."
  docker buildx create --name "$builder" --driver docker-container >/dev/null
fi

docker buildx inspect --bootstrap "$builder" >/dev/null

echo "Baue Linux AMD64 und ARM64 mit Buildx-Builder '$builder' ..."
docker buildx build \
  --builder "$builder" \
  --platform linux/amd64,linux/arm64 \
  --target binary \
  --output "type=local,dest=${output}/binaries" \
  "$root"

for arch in amd64 arm64; do
  dir="${output}/binaries/linux_${arch}"
  for name in nexttimer-server nexttimer; do
    if [[ ! -f "$dir/$name" ]]; then
      echo "Fehler: Build-Ausgabe fehlt: $dir/$name" >&2
      exit 1
    fi
    chmod +x "$dir/$name"
    tar -czf "${output}/${name}-linux-${arch}.tar.gz" -C "$dir" "$name"
    echo "OK: ${output}/${name}-linux-${arch}.tar.gz"
  done
done
