package site

import (
	"image"
	"image/color"
	"image/jpeg"
	"os"
	"path/filepath"
	"testing"
)

func writeTestJPEG(t *testing.T, path string, w, h int) {
	t.Helper()
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			img.Set(x, y, color.RGBA{uint8(x % 256), uint8(y % 256), 128, 255})
		}
	}
	f, err := os.Create(path)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	if err := jpeg.Encode(f, img, nil); err != nil {
		t.Fatal(err)
	}
}

func TestCaptionFromFilename(t *testing.T) {
	cases := []struct {
		in, caption, date string
	}{
		{"2026-02-04-lanzamiento-rediseno.jpeg", "Lanzamiento rediseno", "4 de febrero de 2026"},
		{"retrato-erika-montero.jpg", "Retrato erika montero", ""},
		{"2025-09-30_discurso_graduacion.png", "Discurso graduacion", "30 de septiembre de 2025"},
	}
	for _, c := range cases {
		caption, date := captionFromFilename(c.in)
		if caption != c.caption || date != c.date {
			t.Errorf("%s → (%q, %q); se esperaba (%q, %q)", c.in, caption, date, c.caption, c.date)
		}
	}
}

func TestFitWidth(t *testing.T) {
	if w, h := fitWidth(3200, 1600, 1600); w != 1600 || h != 800 {
		t.Errorf("fitWidth reduce mal: %dx%d", w, h)
	}
	if w, h := fitWidth(800, 600, 1600); w != 800 || h != 600 {
		t.Errorf("fitWidth no debe ampliar: %dx%d", w, h)
	}
}

func TestProcessPhotos(t *testing.T) {
	root := t.TempDir()
	fotos := filepath.Join(root, "fotos")
	out := filepath.Join(root, "public")
	if err := os.MkdirAll(fotos, 0o755); err != nil {
		t.Fatal(err)
	}
	writeTestJPEG(t, filepath.Join(fotos, "2026-01-15-acto-academico.jpeg"), 2400, 1200)
	writeFile(t, filepath.Join(fotos, "captions.yaml"),
		"2026-01-15-acto-academico: \"Acto académico de prueba\"\n")

	photos, err := ProcessPhotos(fotos, out)
	if err != nil {
		t.Fatal(err)
	}
	if len(photos) != 1 {
		t.Fatalf("photos = %d, se esperaba 1", len(photos))
	}
	p := photos[0]
	if p.Caption != "Acto académico de prueba" {
		t.Errorf("caption override falló: %q", p.Caption)
	}
	if p.Date != "15 de enero de 2026" {
		t.Errorf("fecha = %q", p.Date)
	}
	if p.Width != 1600 || p.Height != 800 {
		t.Errorf("dimensiones = %dx%d, se esperaba 1600x800", p.Width, p.Height)
	}
	if p.ThumbW != 480 || p.ThumbH != 240 {
		t.Errorf("thumb = %dx%d, se esperaba 480x240", p.ThumbW, p.ThumbH)
	}
	for _, rel := range []string{"img/2026-01-15-acto-academico.jpg", "img/2026-01-15-acto-academico-thumb.jpg"} {
		if _, err := os.Stat(filepath.Join(out, rel)); err != nil {
			t.Errorf("no existe %s: %v", rel, err)
		}
	}
}

func TestProcessPhotosSinCarpeta(t *testing.T) {
	photos, err := ProcessPhotos(filepath.Join(t.TempDir(), "no-existe"), t.TempDir())
	if err != nil || photos != nil {
		t.Errorf("carpeta ausente debe dar (nil, nil); got (%v, %v)", photos, err)
	}
}

func TestProcessPhotosOrdenFechaDesc(t *testing.T) {
	root := t.TempDir()
	fotos := filepath.Join(root, "fotos")
	if err := os.MkdirAll(fotos, 0o755); err != nil {
		t.Fatal(err)
	}
	writeTestJPEG(t, filepath.Join(fotos, "2025-01-01-viejo.jpeg"), 100, 100)
	writeTestJPEG(t, filepath.Join(fotos, "2026-01-01-nuevo.jpeg"), 100, 100)
	writeTestJPEG(t, filepath.Join(fotos, "sin-fecha.jpeg"), 100, 100)

	photos, err := ProcessPhotos(fotos, filepath.Join(root, "public"))
	if err != nil {
		t.Fatal(err)
	}
	if len(photos) != 3 {
		t.Fatalf("photos = %d", len(photos))
	}
	if photos[0].Name != "2026-01-01-nuevo" || photos[1].Name != "2025-01-01-viejo" {
		t.Errorf("orden: %s, %s, %s", photos[0].Name, photos[1].Name, photos[2].Name)
	}
}
