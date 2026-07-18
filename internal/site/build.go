package site

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"html/template"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

// Config del sitio; BaseURL alimenta canonicals, OG y sitemap.
const (
	BaseURL  = "https://erikaamontero.github.io"
	SiteName = "Erika Montero · Cátedra de Física Básica UASD 2026-2030"
	OGImage  = "/img/og.jpg"
)

// NavItem es un elemento del menú principal.
type NavItem struct {
	Label  string
	URL    string
	Active bool
}

// RenderData es el contexto que reciben las plantillas.
type RenderData struct {
	*Page
	Site struct {
		Name    string
		BaseURL string
		OGImage string
	}
	Nav       []NavItem
	CSS       template.CSS
	Photos    []Photo
	Canonical string
	Version   string
}

// Build genera el sitio completo: root es la raíz del repo, outName el
// directorio de salida relativo a root (p. ej. "public") o una ruta absoluta.
func Build(root, outName string) error {
	outDir := outName
	if !filepath.IsAbs(outDir) {
		outDir = filepath.Join(root, outName)
	}
	if err := os.RemoveAll(outDir); err != nil {
		return err
	}
	if err := os.MkdirAll(outDir, 0o755); err != nil {
		return err
	}

	pages, err := LoadPages(filepath.Join(root, "content"))
	if err != nil {
		return err
	}
	photos, err := ProcessPhotos(filepath.Join(root, "fotos"), outDir)
	if err != nil {
		return err
	}

	rawCSS, err := os.ReadFile(filepath.Join(root, "static", "css", "site.css"))
	if err != nil {
		return fmt.Errorf("css: %w", err)
	}
	css := template.CSS(MinifyCSS(string(rawCSS)))

	tpls, err := template.ParseGlob(filepath.Join(root, "templates", "*.html"))
	if err != nil {
		return fmt.Errorf("plantillas: %w", err)
	}

	version := buildVersion(pages)

	// Render de cada página.
	for _, p := range pages {
		p.Content = enhanceImages(p.Content, photos)
		data := RenderData{Page: p, CSS: css, Photos: photos, Version: version}
		data.Site.Name = SiteName
		data.Site.BaseURL = BaseURL
		data.Site.OGImage = OGImage
		data.Canonical = BaseURL + p.URL()
		for _, n := range pages {
			data.Nav = append(data.Nav, NavItem{
				Label:  n.NavLabel,
				URL:    n.URL(),
				Active: n.Slug == p.Slug,
			})
		}

		var sb strings.Builder
		tplName := p.Template + ".html"
		if tpls.Lookup(tplName) == nil {
			return fmt.Errorf("%s: plantilla %q no existe", p.SrcPath, tplName)
		}
		if err := tpls.ExecuteTemplate(&sb, tplName, data); err != nil {
			return fmt.Errorf("%s: render: %w", p.SrcPath, err)
		}

		outPath := filepath.Join(outDir, p.OutPath())
		if err := os.MkdirAll(filepath.Dir(outPath), 0o755); err != nil {
			return err
		}
		if err := os.WriteFile(outPath, []byte(MinifyHTML(sb.String())), 0o644); err != nil {
			return err
		}
	}

	// Índice de búsqueda.
	idx := BuildSearchIndex(pages)
	idxJSON, err := json.Marshal(idx)
	if err != nil {
		return err
	}
	if err := os.WriteFile(filepath.Join(outDir, "search-index.json"), idxJSON, 0o644); err != nil {
		return err
	}

	// Sitemap y robots.
	if err := writeSitemap(outDir, pages); err != nil {
		return err
	}
	robots := "User-agent: *\nAllow: /\nSitemap: " + BaseURL + "/sitemap.xml\n"
	if err := os.WriteFile(filepath.Join(outDir, "robots.txt"), []byte(robots), 0o644); err != nil {
		return err
	}

	// Copia static/ (js, img, fonts, manifest, sw) — el CSS ya va inline.
	if err := copyStatic(filepath.Join(root, "static"), outDir, version); err != nil {
		return err
	}
	return nil
}

// buildVersion deriva una versión corta del contenido (para cache-busting del sw).
func buildVersion(pages []*Page) string {
	h := sha256.New()
	for _, p := range pages {
		io.WriteString(h, p.Title)
		io.WriteString(h, string(p.Content))
	}
	return hex.EncodeToString(h.Sum(nil))[:12]
}

func writeSitemap(outDir string, pages []*Page) error {
	var sb strings.Builder
	sb.WriteString(`<?xml version="1.0" encoding="UTF-8"?>` + "\n")
	sb.WriteString(`<urlset xmlns="http://www.sitemaps.org/schemas/sitemap/0.9">` + "\n")
	for _, p := range pages {
		sb.WriteString("  <url><loc>" + BaseURL + p.URL() + "</loc></url>\n")
	}
	sb.WriteString("</urlset>\n")
	return os.WriteFile(filepath.Join(outDir, "sitemap.xml"), []byte(sb.String()), 0o644)
}

// copyStatic copia static/ hacia la salida, omitiendo css/ (inline) y
// reemplazando %%VERSION%% en sw.js.
func copyStatic(staticDir, outDir, version string) error {
	return filepath.WalkDir(staticDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			if os.IsNotExist(err) && path == staticDir {
				return filepath.SkipAll
			}
			return err
		}
		rel, _ := filepath.Rel(staticDir, path)
		if d.IsDir() {
			if rel == "css" {
				return filepath.SkipDir
			}
			return nil
		}
		dst := filepath.Join(outDir, rel)
		if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
			return err
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		if filepath.Base(path) == "sw.js" {
			data = []byte(strings.ReplaceAll(string(data), "%%VERSION%%", version))
		}
		return os.WriteFile(dst, data, 0o644)
	})
}

var reImgTag = regexp.MustCompile(`<img src="([^"]+)" alt="([^"]*)"\s*/?>`)

// enhanceImages completa las <img> del markdown con dimensiones (CLS = 0),
// carga diferida y decodificación asíncrona.
func enhanceImages(content template.HTML, photos []Photo) template.HTML {
	dims := make(map[string]Photo, len(photos))
	for _, p := range photos {
		dims[p.Src] = p
	}
	out := reImgTag.ReplaceAllStringFunc(string(content), func(tag string) string {
		m := reImgTag.FindStringSubmatch(tag)
		src, alt := m[1], m[2]
		attrs := fmt.Sprintf(`<img src="%s" alt="%s" loading="lazy" decoding="async"`, src, alt)
		if p, ok := dims[src]; ok {
			attrs += fmt.Sprintf(` width="%d" height="%d"`, p.Width, p.Height)
		}
		return attrs + ">"
	})
	return template.HTML(out)
}

var (
	reHTMLComment = regexp.MustCompile(`<!--[^\[](?s:.*?)-->`)
	reBlankLines  = regexp.MustCompile(`\n\s*\n+`)
	reLeadingWS   = regexp.MustCompile(`(?m)^[ \t]+`)
)

// MinifyHTML aplica una minificación conservadora: quita comentarios,
// sangrías y líneas en blanco. No toca contenido inline.
func MinifyHTML(s string) string {
	s = reHTMLComment.ReplaceAllString(s, "")
	s = reLeadingWS.ReplaceAllString(s, "")
	s = reBlankLines.ReplaceAllString(s, "\n")
	return strings.TrimSpace(s) + "\n"
}

var (
	reCSSComment = regexp.MustCompile(`/\*(?s:.*?)\*/`)
	reCSSSpace   = regexp.MustCompile(`\s+`)
	reCSSPunct   = regexp.MustCompile(`\s*([{}:;,>])\s*`)
)

// MinifyCSS compacta el CSS: sin comentarios ni espacios redundantes.
func MinifyCSS(s string) string {
	s = reCSSComment.ReplaceAllString(s, "")
	s = reCSSSpace.ReplaceAllString(s, " ")
	s = reCSSPunct.ReplaceAllString(s, "$1")
	s = strings.ReplaceAll(s, ";}", "}")
	return strings.TrimSpace(s)
}
