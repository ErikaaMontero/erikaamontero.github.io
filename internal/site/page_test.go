package site

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func writeFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestSplitFrontmatter(t *testing.T) {
	fm, body, err := splitFrontmatter([]byte("---\ntitle: Hola\n---\n# Cuerpo\n"))
	if err != nil {
		t.Fatalf("error inesperado: %v", err)
	}
	if !strings.Contains(string(fm), "title: Hola") {
		t.Errorf("frontmatter = %q", fm)
	}
	if !strings.HasPrefix(string(body), "# Cuerpo") {
		t.Errorf("body = %q", body)
	}
}

func TestSplitFrontmatterMissing(t *testing.T) {
	for _, src := range []string{"# Sin frontmatter", "---\nsin cierre"} {
		if _, _, err := splitFrontmatter([]byte(src)); err == nil {
			t.Errorf("se esperaba error para %q", src)
		}
	}
}

func TestParsePage(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "prueba.md")
	writeFile(t, path, `---
title: "Física Educativa"
description: "Una página de prueba."
slug: "fisica"
order: 3
---

## Sección Única {#seccion-unica}

Texto con **negritas** y [enlace](/otra/).
`)
	p, err := ParsePage(path)
	if err != nil {
		t.Fatal(err)
	}
	if p.Title != "Física Educativa" || p.Slug != "fisica" || p.Order != 3 {
		t.Errorf("frontmatter mal parseado: %+v", p.Frontmatter)
	}
	if p.URL() != "/fisica/" {
		t.Errorf("URL() = %q", p.URL())
	}
	if p.OutPath() != filepath.Join("fisica", "index.html") {
		t.Errorf("OutPath() = %q", p.OutPath())
	}
	html := string(p.Content)
	if !strings.Contains(html, `<h2 id="seccion-unica">`) {
		t.Errorf("falta id de encabezado en HTML: %s", html)
	}
	if !strings.Contains(html, "<strong>negritas</strong>") {
		t.Errorf("markdown no renderizado: %s", html)
	}
}

func TestParsePageErrores(t *testing.T) {
	dir := t.TempDir()
	cases := map[string]string{
		"sin-frontmatter.md": "# Solo cuerpo\n",
		"sin-title.md":       "---\ndescription: x\n---\ncuerpo\n",
		"sin-descr.md":       "---\ntitle: x\n---\ncuerpo\n",
	}
	for name, content := range cases {
		path := filepath.Join(dir, name)
		writeFile(t, path, content)
		if _, err := ParsePage(path); err == nil {
			t.Errorf("%s: se esperaba error", name)
		}
	}
}

func TestExtractHeadingsYSearchIndex(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "p.md")
	writeFile(t, path, `---
title: "Propuesta"
description: "Descripción de prueba."
slug: "propuesta"
---

## Laboratorios y Equipos {#eje-b}

Gestionar recursos para renovar los laboratorios de Física Básica.

- Mantenimiento preventivo y correctivo.

## Otra Sección {#otra}

Texto de la otra sección.
`)
	p, err := ParsePage(path)
	if err != nil {
		t.Fatal(err)
	}
	if len(p.Headings) != 2 {
		t.Fatalf("headings = %d, se esperaban 2", len(p.Headings))
	}
	h := p.Headings[0]
	if h.ID != "eje-b" || h.Text != "Laboratorios y Equipos" {
		t.Errorf("heading mal extraído: %+v", h)
	}
	if !strings.Contains(h.Body, "renovar los laboratorios") ||
		!strings.Contains(h.Body, "Mantenimiento preventivo") {
		t.Errorf("cuerpo de sección incompleto: %q", h.Body)
	}
	if strings.Contains(h.Body, "Otra Sección") || strings.Contains(h.Body, "otra sección") {
		t.Errorf("el cuerpo invadió la sección siguiente: %q", h.Body)
	}

	idx := BuildSearchIndex([]*Page{p})
	// 1 entrada por página + 2 por encabezados.
	if len(idx) != 3 {
		t.Fatalf("índice = %d entradas, se esperaban 3", len(idx))
	}
	if idx[1].URL != "/propuesta/#eje-b" {
		t.Errorf("URL con ancla = %q", idx[1].URL)
	}
	if idx[1].Page != "Propuesta" {
		t.Errorf("Page = %q", idx[1].Page)
	}
}

func TestLoadPagesOrden(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "b.md"), "---\ntitle: B\ndescription: d\nslug: b\norder: 2\n---\ncuerpo\n")
	writeFile(t, filepath.Join(dir, "a.md"), "---\ntitle: A\ndescription: d\nslug: a\norder: 1\n---\ncuerpo\n")
	pages, err := LoadPages(dir)
	if err != nil {
		t.Fatal(err)
	}
	if pages[0].Title != "A" || pages[1].Title != "B" {
		t.Errorf("orden incorrecto: %s, %s", pages[0].Title, pages[1].Title)
	}
}

func TestMinifyCSS(t *testing.T) {
	got := MinifyCSS("/* comentario */\nbody {\n  color: red;\n}\n")
	if got != "body{color:red}" {
		t.Errorf("MinifyCSS = %q", got)
	}
}

func TestMinifyHTML(t *testing.T) {
	got := MinifyHTML("<div>\n    <p>Hola</p>\n\n  <!-- fuera -->\n</div>")
	if strings.Contains(got, "<!--") || strings.Contains(got, "    ") {
		t.Errorf("MinifyHTML = %q", got)
	}
	if !strings.Contains(got, "<p>Hola</p>") {
		t.Errorf("contenido perdido: %q", got)
	}
}
