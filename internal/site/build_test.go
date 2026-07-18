package site

import (
	"encoding/json"
	"html/template"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

// repoRoot localiza la raíz del repositorio desde el directorio del paquete.
func repoRoot(t *testing.T) string {
	t.Helper()
	root, err := filepath.Abs(filepath.Join("..", ".."))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(root, "go.mod")); err != nil {
		t.Skipf("raíz del repo no encontrada: %v", err)
	}
	return root
}

// TestBuildIntegral construye el sitio real hacia un directorio temporal y
// valida páginas, metadatos, sitemap, índice, PWA y enlaces internos.
func TestBuildIntegral(t *testing.T) {
	root := repoRoot(t)
	out := filepath.Join(t.TempDir(), "public")
	if err := Build(root, out); err != nil {
		t.Fatal(err)
	}

	pages := []string{
		"index.html",
		"propuesta/index.html",
		"trayectoria/index.html",
		"cv/index.html",
		"galeria/index.html",
		"contacto/index.html",
	}
	for _, rel := range pages {
		path := filepath.Join(out, rel)
		data, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("página faltante %s: %v", rel, err)
		}
		html := string(data)
		for _, needle := range []string{
			"<title>",
			`name="description"`,
			`rel="canonical"`,
			`property="og:title"`,
			`property="og:image" content="` + BaseURL + OGImage + `"`,
			`name="twitter:card"`,
			`rel="manifest"`,
			"<style>", // CSS inline
		} {
			if !strings.Contains(html, needle) {
				t.Errorf("%s: falta %q", rel, needle)
			}
		}
	}

	// Anclas de los seis ejes en la propuesta.
	propuesta, _ := os.ReadFile(filepath.Join(out, "propuesta/index.html"))
	for _, id := range []string{"eje-a", "eje-b", "eje-c", "eje-d", "eje-e", "eje-f"} {
		if !strings.Contains(string(propuesta), `id="`+id+`"`) {
			t.Errorf("propuesta: falta ancla #%s", id)
		}
	}

	// Sitemap con las cinco URLs.
	sitemap, err := os.ReadFile(filepath.Join(out, "sitemap.xml"))
	if err != nil {
		t.Fatal(err)
	}
	for _, u := range []string{"/", "/propuesta/", "/trayectoria/", "/cv/", "/galeria/", "/contacto/"} {
		if !strings.Contains(string(sitemap), "<loc>"+BaseURL+u+"</loc>") {
			t.Errorf("sitemap: falta %s", u)
		}
	}

	// robots.txt
	robots, err := os.ReadFile(filepath.Join(out, "robots.txt"))
	if err != nil || !strings.Contains(string(robots), "Sitemap: ") {
		t.Errorf("robots.txt inválido: %v", err)
	}

	// Índice de búsqueda no vacío y bien formado.
	var idx []SearchEntry
	idxData, err := os.ReadFile(filepath.Join(out, "search-index.json"))
	if err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(idxData, &idx); err != nil {
		t.Fatalf("search-index.json inválido: %v", err)
	}
	if len(idx) < 15 {
		t.Errorf("índice sospechosamente pequeño: %d entradas", len(idx))
	}

	// PWA: manifest y sw con versión inyectada.
	if _, err := os.Stat(filepath.Join(out, "manifest.webmanifest")); err != nil {
		t.Error("falta manifest.webmanifest")
	}
	sw, err := os.ReadFile(filepath.Join(out, "sw.js"))
	if err != nil {
		t.Fatal("falta sw.js")
	}
	if strings.Contains(string(sw), "%%VERSION%%") {
		t.Error("sw.js: versión sin inyectar")
	}

	// Sin restos de comentarios HTML (minificación).
	for _, rel := range pages {
		data, _ := os.ReadFile(filepath.Join(out, rel))
		if strings.Contains(string(data), "<!-- ") {
			t.Errorf("%s: comentarios HTML sin minificar", rel)
		}
	}

	checkInternalLinks(t, out, pages)
}

func TestEnhanceImages(t *testing.T) {
	photos := []Photo{{Src: "/img/foto.jpg", Width: 1600, Height: 900}}
	in := `<p><img src="/img/foto.jpg" alt="Una foto"></p><img src="/img/otra.jpg" alt="">`
	out := string(enhanceImages(template.HTML(in), photos))
	if !strings.Contains(out, `<img src="/img/foto.jpg" alt="Una foto" loading="lazy" decoding="async" width="1600" height="900">`) {
		t.Errorf("img conocida sin dimensiones: %s", out)
	}
	if !strings.Contains(out, `<img src="/img/otra.jpg" alt="" loading="lazy" decoding="async">`) {
		t.Errorf("img desconocida sin lazy: %s", out)
	}
}

var reHref = regexp.MustCompile(`(?:href|src)="(/[^"#]*)`)

// checkInternalLinks verifica que todo enlace/recurso local apunte a un
// archivo generado (detección de enlaces rotos).
func checkInternalLinks(t *testing.T, out string, pages []string) {
	t.Helper()
	for _, rel := range pages {
		data, _ := os.ReadFile(filepath.Join(out, rel))
		for _, m := range reHref.FindAllStringSubmatch(string(data), -1) {
			u := m[1]
			target := filepath.Join(out, filepath.FromSlash(strings.TrimPrefix(u, "/")))
			if strings.HasSuffix(u, "/") {
				target = filepath.Join(target, "index.html")
			}
			if _, err := os.Stat(target); err != nil {
				t.Errorf("%s: enlace roto %q", rel, u)
			}
		}
	}
}

// TestBuildAnclasDeBusquedaExisten valida que cada URL con ancla del índice
// de búsqueda tenga su id correspondiente en el HTML generado.
func TestBuildAnclasDeBusquedaExisten(t *testing.T) {
	root := repoRoot(t)
	out := filepath.Join(t.TempDir(), "public")
	if err := Build(root, out); err != nil {
		t.Fatal(err)
	}
	var idx []SearchEntry
	data, err := os.ReadFile(filepath.Join(out, "search-index.json"))
	if err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(data, &idx); err != nil {
		t.Fatal(err)
	}
	htmlCache := map[string]string{}
	for _, e := range idx {
		parts := strings.SplitN(e.URL, "#", 2)
		if len(parts) != 2 {
			continue
		}
		page := parts[0]
		rel := strings.TrimPrefix(page, "/")
		file := filepath.Join(out, rel, "index.html")
		html, ok := htmlCache[file]
		if !ok {
			b, err := os.ReadFile(file)
			if err != nil {
				t.Fatalf("página %s del índice no existe: %v", page, err)
			}
			html = string(b)
			htmlCache[file] = html
		}
		if !strings.Contains(html, `id="`+parts[1]+`"`) {
			t.Errorf("índice: ancla %q no existe en %s", parts[1], page)
		}
	}
}
