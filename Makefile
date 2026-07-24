# Makefile del sitio de campaña — Prof. Erika Montero
# Uso rápido:  make help

PORT ?= 8080
MSG  ?= Actualiza el contenido del sitio
SITE := https://erikaamontero.github.io

.DEFAULT_GOAL := help
.PHONY: help build test e2e browsers lighthouse check serve publish verify-live clean

help: ## Lista los comandos disponibles
	@echo "Comandos disponibles:"
	@grep -E '^[a-zA-Z0-9-]+:.*## ' $(MAKEFILE_LIST) | \
		awk -F':.*## ' '{printf "  \033[36m%-12s\033[0m %s\n", $$1, $$2}'
	@echo ""
	@echo "Variables: PORT=$(PORT) · MSG=\"$(MSG)\""
	@echo "Ejemplo:   make publish MSG=\"Actualiza la propuesta\""

build: ## Genera el sitio en public/
	go run ./cmd/build

test: ## Unit tests del generador
	go test ./...

e2e: ## Pruebas de interfaz con Playwright (requiere: make browsers)
	go test -tags e2e ./e2e

browsers: ## Instala el Chromium de Playwright (una sola vez)
	go run github.com/playwright-community/playwright-go/cmd/playwright install chromium

lighthouse: ## Auditoría de rendimiento y accesibilidad (umbral 95)
	./scripts/lighthouse.sh

check: test e2e lighthouse ## Verificación completa antes de publicar

serve: build ## Sirve el sitio en localhost (PORT=8080 por defecto)
	@echo "Sirviendo en http://localhost:$(PORT) (Ctrl+C para detener)"
	@cd public && python3 -m http.server $(PORT)

publish: test ## Prueba, comitea (MSG="...") y publica vía GitHub Actions
	@git add -A
	@if git diff --cached --quiet; then \
		echo "Sin cambios nuevos que comitear; subiendo commits pendientes…"; \
	else \
		git commit -m "$(MSG)" -m "Co-Authored-By: Claude Opus 4.8 (1M context) <noreply@anthropic.com>"; \
	fi
	git push
	@echo "✓ Publicado. El workflow desplegará en ~1-2 min: $(SITE)"

verify-live: ## Verifica que el sitio publicado responde (tras el deploy)
	@fail=0; \
	for p in / /propuesta/ /trayectoria/ /cv/ /galeria/ /contacto/ \
	         /img/og.jpg /manifest.webmanifest /search-index.json /sitemap.xml; do \
		code=$$(curl -s -o /dev/null -w '%{http_code}' $(SITE)$$p); \
		printf "  %-25s %s\n" "$$p" "$$code"; \
		[ "$$code" = "200" ] || fail=1; \
	done; \
	[ $$fail -eq 0 ] && echo "✓ Sitio en vivo OK" || { echo "✗ Hay páginas fallando"; exit 1; }

clean: ## Borra los artefactos generados (public/, reportes)
	rm -rf public lighthouse-reports
