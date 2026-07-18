//go:build e2e

// Pruebas end-to-end del sitio con Playwright.
// Ejecutar:  go test -tags e2e ./e2e
// Requiere:  go run github.com/playwright-community/playwright-go/cmd/playwright install chromium
package e2e

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	"github.com/playwright-community/playwright-go"

	"github.com/ErikaaMontero/erikaamontero.github.io/internal/site"
)

var (
	serverURL string
	pw        *playwright.Playwright
	browser   playwright.Browser
)

type viewport struct {
	name string
	w, h int
}

var viewports = []viewport{
	{"movil", 390, 844},
	{"tableta", 820, 1180},
	{"escritorio", 1440, 900},
}

var pages = []string{"/", "/propuesta/", "/trayectoria/", "/cv/", "/galeria/", "/contacto/"}

func TestMain(m *testing.M) {
	root, err := filepath.Abs("..")
	if err != nil {
		panic(err)
	}
	out, err := os.MkdirTemp("", "site-e2e-*")
	if err != nil {
		panic(err)
	}
	defer os.RemoveAll(out)
	if err := site.Build(root, filepath.Join(out, "public")); err != nil {
		panic(fmt.Sprintf("build: %v", err))
	}

	srv := httptest.NewServer(http.FileServer(http.Dir(filepath.Join(out, "public"))))
	defer srv.Close()
	serverURL = srv.URL

	pw, err = playwright.Run()
	if err != nil {
		panic(fmt.Sprintf("playwright: %v (¿falta 'playwright install chromium'?)", err))
	}
	defer pw.Stop()
	browser, err = pw.Chromium.Launch()
	if err != nil {
		panic(fmt.Sprintf("chromium: %v", err))
	}
	defer browser.Close()

	os.Exit(m.Run())
}

// newPage abre una página con el viewport dado y acumula errores de consola,
// errores de página y respuestas fallidas.
func newPage(t *testing.T, vp viewport) (playwright.Page, *[]string) {
	t.Helper()
	ctx, err := browser.NewContext(playwright.BrowserNewContextOptions{
		Viewport: &playwright.Size{Width: vp.w, Height: vp.h},
	})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { ctx.Close() })
	page, err := ctx.NewPage()
	if err != nil {
		t.Fatal(err)
	}

	var mu sync.Mutex
	problems := &[]string{}
	page.OnConsole(func(msg playwright.ConsoleMessage) {
		if msg.Type() == "error" {
			mu.Lock()
			*problems = append(*problems, "console: "+msg.Text())
			mu.Unlock()
		}
	})
	page.OnPageError(func(err error) {
		mu.Lock()
		*problems = append(*problems, "pageerror: "+err.Error())
		mu.Unlock()
	})
	page.OnResponse(func(res playwright.Response) {
		if res.Status() >= 400 {
			mu.Lock()
			*problems = append(*problems, fmt.Sprintf("HTTP %d: %s", res.Status(), res.URL()))
			mu.Unlock()
		}
	})
	return page, problems
}

func goTo(t *testing.T, page playwright.Page, path string) {
	t.Helper()
	if _, err := page.Goto(serverURL+path, playwright.PageGotoOptions{
		WaitUntil: playwright.WaitUntilStateNetworkidle,
	}); err != nil {
		t.Fatalf("goto %s: %v", path, err)
	}
}

// TestPaginasSinErroresNiDesbordes carga cada página en cada viewport y
// verifica: sin errores de consola, sin requests fallidos y sin scroll
// horizontal.
func TestPaginasSinErroresNiDesbordes(t *testing.T) {
	for _, vp := range viewports {
		for _, path := range pages {
			t.Run(vp.name+strings.ReplaceAll(path, "/", "_"), func(t *testing.T) {
				page, problems := newPage(t, vp)
				goTo(t, page, path)

				for _, p := range *problems {
					t.Errorf("%s [%s]: %s", path, vp.name, p)
				}

				overflow, err := page.Evaluate(
					`document.documentElement.scrollWidth - document.documentElement.clientWidth`)
				if err != nil {
					t.Fatal(err)
				}
				if n, ok := overflow.(int); ok && n > 1 {
					t.Errorf("%s [%s]: desborde horizontal de %dpx", path, vp.name, n)
				}
			})
		}
	}
}

// TestNavegacionEscritorio: los enlaces del header llevan a cada sección.
func TestNavegacionEscritorio(t *testing.T) {
	page, _ := newPage(t, viewports[2])
	goTo(t, page, "/")
	targets := map[string]string{
		"Propuesta":   "/propuesta/",
		"Trayectoria": "/trayectoria/",
		"CV":          "/cv/",
		"Galería":     "/galeria/",
		"Contacto":    "/contacto/",
	}
	for label, want := range targets {
		goTo(t, page, "/")
		if err := page.Locator(".site-nav a", playwright.PageLocatorOptions{
			HasText: label,
		}).Click(); err != nil {
			t.Fatalf("click %s: %v", label, err)
		}
		if err := page.WaitForURL("**" + want); err != nil {
			t.Errorf("nav %s: no llegó a %s (url=%s)", label, want, page.URL())
		}
	}
}

// TestMenuMovil: el menú hamburguesa abre, cierra y navega.
func TestMenuMovil(t *testing.T) {
	page, _ := newPage(t, viewports[0])
	goTo(t, page, "/")

	nav := page.Locator("#nav")
	visible, _ := nav.IsVisible()
	if visible {
		t.Fatal("el menú móvil no debería estar visible al cargar")
	}
	if err := page.Locator("#nav-toggle").Click(); err != nil {
		t.Fatal(err)
	}
	if err := nav.WaitFor(playwright.LocatorWaitForOptions{
		State: playwright.WaitForSelectorStateVisible,
	}); err != nil {
		t.Fatal("el menú móvil no se abrió")
	}
	// Cierra y reabre.
	if err := page.Locator("#nav-toggle").Click(); err != nil {
		t.Fatal(err)
	}
	if v, _ := nav.IsVisible(); v {
		t.Error("el menú móvil no se cerró")
	}
	if err := page.Locator("#nav-toggle").Click(); err != nil {
		t.Fatal(err)
	}
	if err := page.Locator("#nav a", playwright.PageLocatorOptions{
		HasText: "Propuesta",
	}).Click(); err != nil {
		t.Fatal(err)
	}
	if err := page.WaitForURL("**/propuesta/"); err != nil {
		t.Errorf("menú móvil: no navegó a propuesta (url=%s)", page.URL())
	}
}

// TestBusqueda: overlay con botón y atajo, resultados con y sin acentos,
// Enter navega al primer resultado.
func TestBusqueda(t *testing.T) {
	page, _ := newPage(t, viewports[2])
	goTo(t, page, "/")

	// Abrir con el botón.
	if err := page.Locator("#search-open").Click(); err != nil {
		t.Fatal(err)
	}
	dialog := page.Locator("#search-dialog")
	if err := dialog.WaitFor(playwright.LocatorWaitForOptions{
		State: playwright.WaitForSelectorStateVisible,
	}); err != nil {
		t.Fatal("el diálogo de búsqueda no abrió con el botón")
	}

	// "laboratorios" debe traer resultados (eje B).
	if err := page.Locator("#search-input").Fill("laboratorios"); err != nil {
		t.Fatal(err)
	}
	results := page.Locator("#search-results li a")
	if err := results.First().WaitFor(playwright.LocatorWaitForOptions{
		State: playwright.WaitForSelectorStateVisible,
	}); err != nil {
		t.Fatal("sin resultados para 'laboratorios'")
	}

	// "fisica" sin acento también debe encontrar "Física".
	if err := page.Locator("#search-input").Fill("fisica"); err != nil {
		t.Fatal(err)
	}
	if err := results.First().WaitFor(playwright.LocatorWaitForOptions{
		State: playwright.WaitForSelectorStateVisible,
	}); err != nil {
		t.Fatal("sin resultados para 'fisica' (búsqueda insensible a acentos)")
	}

	// Cerrar con Esc; reabrir con el atajo "/".
	if err := page.Keyboard().Press("Escape"); err != nil {
		t.Fatal(err)
	}
	if err := dialog.WaitFor(playwright.LocatorWaitForOptions{
		State: playwright.WaitForSelectorStateHidden,
	}); err != nil {
		t.Error("Esc no cerró el diálogo")
	}
	if err := page.Keyboard().Press("/"); err != nil {
		t.Fatal(err)
	}
	if err := dialog.WaitFor(playwright.LocatorWaitForOptions{
		State: playwright.WaitForSelectorStateVisible,
	}); err != nil {
		t.Fatal("el atajo '/' no abrió el diálogo")
	}

	// Enter navega al primer resultado.
	if err := page.Locator("#search-input").Fill("laboratorios"); err != nil {
		t.Fatal(err)
	}
	if err := results.First().WaitFor(playwright.LocatorWaitForOptions{
		State: playwright.WaitForSelectorStateVisible,
	}); err != nil {
		t.Fatal(err)
	}
	href, err := results.First().GetAttribute("href")
	if err != nil {
		t.Fatal(err)
	}
	if err := page.Keyboard().Press("Enter"); err != nil {
		t.Fatal(err)
	}
	base := strings.SplitN(href, "#", 2)[0]
	if err := page.WaitForURL("**" + base + "*"); err != nil {
		t.Errorf("Enter no navegó a %s (url=%s)", href, page.URL())
	}
}

// TestGaleriaYLightbox: el grid muestra las fotos y el lightbox abre/cierra.
func TestGaleriaYLightbox(t *testing.T) {
	page, _ := newPage(t, viewports[2])
	goTo(t, page, "/galeria/")

	items := page.Locator(".galeria-item")
	n, err := items.Count()
	if err != nil {
		t.Fatal(err)
	}
	if n < 5 {
		t.Fatalf("galería con %d fotos; se esperaban al menos 5", n)
	}

	if err := page.Locator(".lightbox-link").First().Click(); err != nil {
		t.Fatal(err)
	}
	lightbox := page.Locator("#lightbox")
	if err := lightbox.WaitFor(playwright.LocatorWaitForOptions{
		State: playwright.WaitForSelectorStateVisible,
	}); err != nil {
		t.Fatal("el lightbox no abrió")
	}
	src, _ := page.Locator("#lightbox-img").GetAttribute("src")
	if !strings.HasPrefix(src, "/img/") {
		t.Errorf("lightbox img src = %q", src)
	}
	if err := page.Locator(".lightbox-close").Click(); err != nil {
		t.Fatal(err)
	}
	if v, _ := lightbox.IsVisible(); v {
		t.Error("el lightbox no cerró")
	}
}

// TestContenidoVisibleTrasScroll: las animaciones reveal dejan visible el
// contenido al recorrer la página.
func TestContenidoVisibleTrasScroll(t *testing.T) {
	page, _ := newPage(t, viewports[0])
	goTo(t, page, "/")

	if _, err := page.Evaluate(`window.scrollTo(0, document.body.scrollHeight)`); err != nil {
		t.Fatal(err)
	}
	page.WaitForTimeout(900) // deja terminar las transiciones

	hidden, err := page.Evaluate(`
		Array.from(document.querySelectorAll(".reveal"))
			.filter(el => getComputedStyle(el).opacity === "0").length`)
	if err != nil {
		t.Fatal(err)
	}
	if n, ok := hidden.(int); ok && n > 0 {
		t.Errorf("%d elementos .reveal siguen ocultos tras el scroll", n)
	}

	// El hero y las cards existen y son visibles.
	for _, sel := range []string{".hero h1", ".eje-card", ".stat"} {
		if v, _ := page.Locator(sel).First().IsVisible(); !v {
			t.Errorf("%s no es visible", sel)
		}
	}
}

// TestPWA: manifest accesible y service worker registrado.
func TestPWA(t *testing.T) {
	page, _ := newPage(t, viewports[2])
	goTo(t, page, "/")

	res, err := page.Request().Get(serverURL + "/manifest.webmanifest")
	if err != nil || !res.Ok() {
		t.Fatalf("manifest.webmanifest inaccesible: %v", err)
	}
	body, _ := res.Text()
	if !strings.Contains(body, `"name"`) || !strings.Contains(body, "icon-512.png") {
		t.Error("manifest incompleto")
	}

	ready, err := page.Evaluate(`
		navigator.serviceWorker.ready.then(reg => !!reg.active || !!reg.waiting || !!reg.installing)`)
	if err != nil {
		t.Fatalf("service worker no registró: %v", err)
	}
	if ok, isBool := ready.(bool); !isBool || !ok {
		t.Errorf("service worker sin activar: %v", ready)
	}
}

// TestSemblanzaTrayectoria: la semblanza tiene sus secciones y fotos, y enlaza
// al CV completo.
func TestSemblanzaTrayectoria(t *testing.T) {
	page, _ := newPage(t, viewports[1])
	goTo(t, page, "/trayectoria/")

	n, err := page.Locator(".prose h2").Count()
	if err != nil {
		t.Fatal(err)
	}
	if n < 6 {
		t.Errorf("semblanza con %d secciones; se esperaban al menos 6", n)
	}
	imgs, err := page.Locator(".prose img").Count()
	if err != nil {
		t.Fatal(err)
	}
	if imgs < 3 {
		t.Errorf("semblanza con %d fotos; se esperaban al menos 3", imgs)
	}
	if err := page.Locator(`.prose a[href="/cv/"]`).Last().ScrollIntoViewIfNeeded(); err != nil {
		t.Fatal(err)
	}
	if err := page.Locator(`.prose a[href="/cv/"]`).Last().Click(); err != nil {
		t.Fatal(err)
	}
	if err := page.WaitForURL("**/cv/"); err != nil {
		t.Errorf("el enlace al CV no navegó (url=%s)", page.URL())
	}
}
