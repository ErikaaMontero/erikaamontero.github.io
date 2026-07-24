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
2. `make publish MSG="Describe tu cambio"`.
3. GitHub Actions reconstruye y publica el sitio automáticamente (1-2 min);
   confirma con `make verify-live`.

Todos los comandos del proyecto están en el `Makefile` (lista completa con
`make help`):

| Comando | Qué hace |
|---|---|
| `make build` | Genera el sitio en `public/` |
| `make serve` | Genera y sirve en `http://localhost:8080` (`PORT=…` para cambiarlo) |
| `make test` | Unit tests del generador |
| `make e2e` | Pruebas de interfaz con Playwright (antes, una vez: `make browsers`) |
| `make lighthouse` | Auditoría de rendimiento/accesibilidad (falla si < 95) |
| `make check` | `test` + `e2e` + `lighthouse`: verificación completa |
| `make publish` | Prueba, comitea (`MSG="…"`) y hace push para desplegar |
| `make verify-live` | Comprueba que el sitio publicado responde |
| `make clean` | Borra `public/` y reportes |

Notas:

- **Fotos**: el generador las optimiza (1600 px máx., thumbnail y pie de foto). Si
  el nombre inicia con `AAAA-MM-DD-`, la fecha se muestra en la galería. Para un
  pie con tildes, agrega la entrada en `fotos/captions.yaml`.
- **Nueva página**: crea `content/nombre.md` con frontmatter (`title`,
  `description`, `slug`, `order`) y entrará sola al menú y al índice de búsqueda.

## Previsualizar localmente

```bash
make serve        # genera y sirve en http://localhost:8080
```

## Pruebas

```bash
make test         # unit tests del generador (corren también en CI)
make browsers     # una sola vez: instala el Chromium de Playwright
make e2e          # E2E: navegación, búsqueda, galería, responsive, PWA
make lighthouse   # auditoría (falla si Performance o Accessibility < 95)
make check        # todo lo anterior junto
```

<details>
<summary>Comandos equivalentes sin make</summary>

```bash
go run ./cmd/build
go test ./...
go run github.com/playwright-community/playwright-go/cmd/playwright install chromium
go test -tags e2e ./e2e
./scripts/lighthouse.sh
```
</details>

## Publicación

El workflow [deploy.yml](.github/workflows/deploy.yml) corre las pruebas, genera
`public/` y lo despliega en GitHub Pages en cada push a `main`.

> ⚙️ Configuración única del repositorio: en **Settings → Pages** la fuente debe
> ser **GitHub Actions**.
