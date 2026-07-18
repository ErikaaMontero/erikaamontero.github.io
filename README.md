# Candidatura — Prof. Erika Montero

**Coordinación de la Cátedra de Física Básica · UASD · Período 2026-2030**

Sitio web de campaña, publicado en <https://erikaamontero.github.io>. Es un sitio
estático generado con un programa propio en **Go** a partir de archivos
**Markdown**: PWA instalable, con búsqueda, previsualización al compartir enlaces
(Open Graph) y rendimiento Lighthouse ≥ 95.

## Estructura

```
├── content/            ← contenido del sitio (Markdown con frontmatter YAML)
│   ├── index.md          página de inicio (hero, cifras y ejes en el frontmatter)
│   ├── propuesta.md      los seis compromisos A–F
│   ├── trayectoria.md    semblanza y CV curado
│   ├── galeria.md        introducción de la galería
│   └── contacto.md
├── fotos/              ← carpeta de fotos: suelta aquí una imagen y aparece en la galería
│   └── captions.yaml     pies de foto con tildes (opcional, por nombre de archivo)
├── templates/          ← plantillas HTML (Go html/template)
├── static/             ← CSS, JS, fuentes, íconos, manifest y service worker
├── cmd/build/          ← generador estático (go run ./cmd/build → public/)
├── internal/site/      ← lógica del generador + unit tests
├── e2e/                ← pruebas de interfaz con Playwright
├── scripts/            ← lighthouse.sh (auditoría de rendimiento)
└── .github/workflows/  ← CI: construye y publica en GitHub Pages en cada push
```

## Cómo actualizar el contenido

1. Edita el archivo correspondiente en `content/` (o agrega fotos a `fotos/`).
2. `git add … && git commit && git push`.
3. GitHub Actions reconstruye y publica el sitio automáticamente (1-2 min).

Notas:

- **Fotos**: el generador las optimiza (1600 px máx., thumbnail y pie de foto). Si
  el nombre inicia con `AAAA-MM-DD-`, la fecha se muestra en la galería. Para un
  pie con tildes, agrega la entrada en `fotos/captions.yaml`.
- **Nueva página**: crea `content/nombre.md` con frontmatter (`title`,
  `description`, `slug`, `order`) y entrará sola al menú y al índice de búsqueda.

## Previsualizar localmente

```bash
go run ./cmd/build          # genera public/
cd public && python3 -m http.server 8080
# abrir http://localhost:8080
```

## Pruebas

```bash
go test ./...               # unit tests del generador (corren también en CI)

# E2E (navegación, búsqueda, galería, responsive, PWA) — requiere Chromium:
go run github.com/playwright-community/playwright-go/cmd/playwright install chromium
go test -tags e2e ./e2e

# Auditoría Lighthouse (falla si Performance o Accessibility < 95):
./scripts/lighthouse.sh
```

## Publicación

El workflow [deploy.yml](.github/workflows/deploy.yml) corre las pruebas, genera
`public/` y lo despliega en GitHub Pages en cada push a `main`.

> ⚙️ Configuración única del repositorio: en **Settings → Pages** la fuente debe
> ser **GitHub Actions**.
