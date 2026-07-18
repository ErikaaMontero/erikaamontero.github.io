package site

import (
	"bytes"
	"fmt"
	"html/template"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/extension"
	"github.com/yuin/goldmark/parser"
	"github.com/yuin/goldmark/renderer/html"
	"gopkg.in/yaml.v3"
)

// Frontmatter es la cabecera YAML de cada archivo de content/.
type Frontmatter struct {
	Title       string `yaml:"title"`
	Description string `yaml:"description"`
	Slug        string `yaml:"slug"`
	Template    string `yaml:"template"`
	Order       int    `yaml:"order"`
	NavLabel    string `yaml:"nav_label"`
	Hero        Hero   `yaml:"hero"`
}

// Hero contiene los datos del encabezado principal de la página de inicio.
type Hero struct {
	Kicker   string   `yaml:"kicker"`
	Name     string   `yaml:"name"`
	Role     string   `yaml:"role"`
	Period   string   `yaml:"period"`
	Photo    string   `yaml:"photo"`
	Lead     string   `yaml:"lead"`
	CTALabel string   `yaml:"cta_label"`
	CTAURL   string   `yaml:"cta_url"`
	Stats    []Stat   `yaml:"stats"`
	Ejes     []Eje    `yaml:"ejes"`
	Quotes   []string `yaml:"quotes"`
}

// Stat es una cifra destacada del hero.
type Stat struct {
	Value string `yaml:"value"`
	Label string `yaml:"label"`
}

// Eje es una card resumen de un eje de compromiso.
type Eje struct {
	ID    string `yaml:"id"`
	Title string `yaml:"title"`
	Desc  string `yaml:"desc"`
}

// Letter devuelve la letra del eje para la ficha ("eje-a" → "A").
func (e Eje) Letter() string {
	if i := strings.LastIndex(e.ID, "-"); i >= 0 && i+1 < len(e.ID) {
		return strings.ToUpper(e.ID[i+1:])
	}
	return strings.ToUpper(e.ID)
}

// Page es una página lista para renderizar.
type Page struct {
	Frontmatter
	Content  template.HTML // cuerpo markdown ya convertido a HTML
	Headings []Heading     // encabezados H2/H3 con su texto asociado (para búsqueda)
	SrcPath  string
}

// Heading es una sección indexable de una página.
type Heading struct {
	Level int
	ID    string
	Text  string
	Body  string // texto plano de la sección, sin HTML
}

// URL devuelve la ruta absoluta de la página dentro del sitio ("/", "/propuesta/").
func (p *Page) URL() string {
	if p.Slug == "" {
		return "/"
	}
	return "/" + p.Slug + "/"
}

// OutPath devuelve la ruta del index.html de salida relativa a public/.
func (p *Page) OutPath() string {
	if p.Slug == "" {
		return "index.html"
	}
	return filepath.Join(p.Slug, "index.html")
}

var errNoFrontmatter = fmt.Errorf("frontmatter ausente: se esperaba un bloque '---' al inicio")

// splitFrontmatter separa el bloque YAML inicial del cuerpo markdown.
func splitFrontmatter(src []byte) (fm, body []byte, err error) {
	const delim = "---"
	s := string(src)
	if !strings.HasPrefix(s, delim+"\n") && !strings.HasPrefix(s, delim+"\r\n") {
		return nil, nil, errNoFrontmatter
	}
	rest := s[len(delim):]
	rest = strings.TrimPrefix(rest, "\r\n")
	rest = strings.TrimPrefix(rest, "\n")
	idx := strings.Index(rest, "\n"+delim)
	if idx < 0 {
		return nil, nil, errNoFrontmatter
	}
	fm = []byte(rest[:idx])
	after := rest[idx+1+len(delim):]
	after = strings.TrimPrefix(after, "\r")
	after = strings.TrimPrefix(after, "\n")
	return fm, []byte(after), nil
}

// newMarkdown construye el conversor goldmark del sitio.
func newMarkdown() goldmark.Markdown {
	return goldmark.New(
		goldmark.WithExtensions(extension.GFM, extension.Typographer),
		goldmark.WithParserOptions(
			parser.WithAutoHeadingID(),
			parser.WithAttribute(), // habilita "## Título {#id-propio}"
		),
		goldmark.WithRendererOptions(html.WithUnsafe()), // el contenido es propio, permite HTML embebido
	)
}

// ParsePage lee un archivo markdown de content/ y lo convierte en Page.
func ParsePage(path string) (*Page, error) {
	src, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	fmRaw, body, err := splitFrontmatter(src)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", path, err)
	}
	var fm Frontmatter
	if err := yaml.Unmarshal(fmRaw, &fm); err != nil {
		return nil, fmt.Errorf("%s: frontmatter inválido: %w", path, err)
	}
	if fm.Title == "" {
		return nil, fmt.Errorf("%s: frontmatter sin 'title'", path)
	}
	if fm.Description == "" {
		return nil, fmt.Errorf("%s: frontmatter sin 'description'", path)
	}
	if fm.Template == "" {
		fm.Template = "page"
	}
	if fm.NavLabel == "" {
		fm.NavLabel = fm.Title
	}

	md := newMarkdown()
	ctx := parser.NewContext()
	var buf bytes.Buffer
	if err := md.Convert(body, &buf, parser.WithContext(ctx)); err != nil {
		return nil, fmt.Errorf("%s: markdown: %w", path, err)
	}

	headings, err := extractHeadings(md, body)
	if err != nil {
		return nil, fmt.Errorf("%s: índice: %w", path, err)
	}

	return &Page{
		Frontmatter: fm,
		Content:     template.HTML(buf.String()),
		Headings:    headings,
		SrcPath:     path,
	}, nil
}

// LoadPages carga todas las páginas de un directorio content/ y las ordena por Order.
func LoadPages(contentDir string) ([]*Page, error) {
	entries, err := os.ReadDir(contentDir)
	if err != nil {
		return nil, err
	}
	var pages []*Page
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".md") {
			continue
		}
		p, err := ParsePage(filepath.Join(contentDir, e.Name()))
		if err != nil {
			return nil, err
		}
		pages = append(pages, p)
	}
	if len(pages) == 0 {
		return nil, fmt.Errorf("no hay páginas .md en %s", contentDir)
	}
	sort.Slice(pages, func(i, j int) bool { return pages[i].Order < pages[j].Order })
	return pages, nil
}
