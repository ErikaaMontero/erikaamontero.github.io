package site

import (
	"strings"

	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/parser"
	"github.com/yuin/goldmark/text"
)

// extractHeadings recorre el AST del markdown y devuelve las secciones H2/H3
// con su texto plano, para alimentar el índice de búsqueda.
func extractHeadings(md goldmark.Markdown, body []byte) ([]Heading, error) {
	reader := text.NewReader(body)
	ctx := parser.NewContext()
	doc := md.Parser().Parse(reader, parser.WithContext(ctx))

	var headings []Heading
	for node := doc.FirstChild(); node != nil; node = node.NextSibling() {
		h, ok := node.(*ast.Heading)
		if !ok || (h.Level != 2 && h.Level != 3) {
			continue
		}
		id := ""
		if v, found := h.AttributeString("id"); found {
			if b, ok := v.([]byte); ok {
				id = string(b)
			} else if s, ok := v.(string); ok {
				id = s
			}
		}
		entry := Heading{
			Level: h.Level,
			ID:    id,
			Text:  nodeText(h, body),
		}
		// Texto de la sección: hermanos siguientes hasta el próximo encabezado
		// del mismo nivel o superior.
		var parts []string
		for sib := node.NextSibling(); sib != nil; sib = sib.NextSibling() {
			if nh, ok := sib.(*ast.Heading); ok && nh.Level <= h.Level {
				break
			}
			if t := nodeText(sib, body); t != "" {
				parts = append(parts, t)
			}
		}
		entry.Body = strings.Join(parts, " ")
		headings = append(headings, entry)
	}
	return headings, nil
}

// nodeText extrae el texto plano de un nodo del AST (sin marcado).
func nodeText(n ast.Node, source []byte) string {
	var sb strings.Builder
	_ = ast.Walk(n, func(node ast.Node, entering bool) (ast.WalkStatus, error) {
		if !entering {
			return ast.WalkContinue, nil
		}
		switch t := node.(type) {
		case *ast.Text:
			sb.Write(t.Segment.Value(source))
			if t.SoftLineBreak() || t.HardLineBreak() {
				sb.WriteByte(' ')
			}
		case *ast.AutoLink:
			sb.Write(t.URL(source))
		}
		return ast.WalkContinue, nil
	})
	return strings.Join(strings.Fields(sb.String()), " ")
}

// SearchEntry es una entrada del índice de búsqueda del sitio.
type SearchEntry struct {
	Page  string `json:"page"`  // título de la página
	Title string `json:"title"` // título de la sección
	URL   string `json:"url"`   // URL con ancla
	Text  string `json:"text"`  // texto plano de la sección
}

// BuildSearchIndex compone el índice de búsqueda de todas las páginas.
func BuildSearchIndex(pages []*Page) []SearchEntry {
	var idx []SearchEntry
	for _, p := range pages {
		// La página misma también es localizable por título/descr.
		idx = append(idx, SearchEntry{
			Page:  p.Title,
			Title: p.Title,
			URL:   p.URL(),
			Text:  p.Description,
		})
		for _, h := range p.Headings {
			url := p.URL()
			if h.ID != "" {
				url += "#" + h.ID
			}
			idx = append(idx, SearchEntry{
				Page:  p.Title,
				Title: h.Text,
				URL:   url,
				Text:  h.Body,
			})
		}
	}
	return idx
}
