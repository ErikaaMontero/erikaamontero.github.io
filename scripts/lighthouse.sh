#!/usr/bin/env bash
# Auditoría Lighthouse del sitio generado en public/.
# Falla si Performance < 95 o Accessibility < 95 en alguna página/preset.
# Uso: ./scripts/lighthouse.sh  (requiere npx; usa el Chromium de Playwright)
set -euo pipefail
cd "$(dirname "$0")/.."

PORT="${PORT:-8907}"
PAGES=("/" "/propuesta/" "/trayectoria/" "/galeria/" "/contacto/")
MIN_PERF="${MIN_PERF:-95}"
MIN_A11Y="${MIN_A11Y:-95}"
OUT_DIR="lighthouse-reports"

command -v npx >/dev/null || { echo "npx no disponible: instala Node.js"; exit 1; }

# Chromium instalado por Playwright (evita depender de un Chrome del sistema).
CHROME_PATH="${CHROME_PATH:-$(find "$HOME/.cache/ms-playwright" -maxdepth 3 \
  -path '*chromium-*/chrome-linux/chrome' 2>/dev/null | sort | tail -1)}"
[ -x "$CHROME_PATH" ] || { echo "Chromium de Playwright no encontrado; ejecuta:
  go run github.com/playwright-community/playwright-go/cmd/playwright install chromium"; exit 1; }
export CHROME_PATH

go run ./cmd/build

mkdir -p "$OUT_DIR"
(cd public && python3 -m http.server "$PORT" >/dev/null 2>&1) &
SERVER_PID=$!
trap 'kill "$SERVER_PID" 2>/dev/null || true' EXIT
sleep 1

fail=0
for preset in mobile desktop; do
  extra=()
  [ "$preset" = "desktop" ] && extra=(--preset=desktop)
  for page in "${PAGES[@]}"; do
    slug=$(echo "${preset}${page}" | tr '/' '-' | sed 's/-$//')
    json="$OUT_DIR/$slug.json"
    npx --yes lighthouse "http://localhost:$PORT$page" \
      --quiet --chrome-flags="--headless=new --no-sandbox" \
      "${extra[@]}" \
      --only-categories=performance,accessibility,best-practices,seo \
      --output=json --output=html \
      --output-path="$OUT_DIR/$slug" >/dev/null 2>&1 || { echo "✗ lighthouse falló en $page ($preset)"; fail=1; continue; }
    # lighthouse escribe .report.json / .report.html
    json="$OUT_DIR/$slug.report.json"
    read -r perf a11y bp seo < <(python3 - "$json" <<'EOF'
import json, sys
d = json.load(open(sys.argv[1]))
c = d["categories"]
def s(k):
    v = c[k]["score"]
    # None = auditoría con error (p. ej. bug de Lighthouse con Node < 22); se marca n/a.
    return "n/a" if v is None else round(v * 100)
print(*(s(k) for k in ("performance", "accessibility", "best-practices", "seo")))
EOF
)
    status="✓"
    if [ "$perf" -lt "$MIN_PERF" ] || [ "$a11y" -lt "$MIN_A11Y" ]; then
      status="✗"; fail=1
    fi
    printf "%s %-9s %-14s perf=%-3s a11y=%-3s bp=%-3s seo=%s\n" \
      "$status" "$preset" "$page" "$perf" "$a11y" "$bp" "$seo"
  done
done

[ "$fail" -eq 0 ] && echo "Lighthouse OK (umbral: perf ≥ $MIN_PERF, a11y ≥ $MIN_A11Y)"
exit "$fail"
