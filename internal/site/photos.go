package site

import (
	"fmt"
	"image"
	"image/jpeg"
	_ "image/png"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"

	"golang.org/x/image/draw"
	_ "golang.org/x/image/webp"
	"gopkg.in/yaml.v3"
)

const (
	photoMaxWidth  = 1600
	thumbMaxWidth  = 480
	photoQuality   = 82
	photoOutSubdir = "img"
)

// Photo es una foto procesada, lista para la galería y para usarse en markdown.
type Photo struct {
	Name    string `json:"name"`    // nombre base sin extensión
	Src     string `json:"src"`     // ruta web de la imagen optimizada
	Thumb   string `json:"thumb"`   // ruta web del thumbnail
	Caption string `json:"caption"` // pie de foto derivado del nombre
	Date    string `json:"date"`    // fecha legible si el nombre la trae
	Width   int    `json:"width"`   // dimensiones de la imagen optimizada
	Height  int    `json:"height"`  //
	ThumbW  int    `json:"thumbW"`  // dimensiones del thumbnail
	ThumbH  int    `json:"thumbH"`  //
	SortKey string `json:"-"`       // para orden estable (fecha desc, nombre)
}

var photoExts = map[string]bool{".jpg": true, ".jpeg": true, ".png": true, ".webp": true}

var datePrefix = regexp.MustCompile(`^(\d{4})-(\d{2})-(\d{2})[-_]?`)

var spanishMonths = [...]string{"", "enero", "febrero", "marzo", "abril", "mayo", "junio",
	"julio", "agosto", "septiembre", "octubre", "noviembre", "diciembre"}

// captionFromFilename deriva pie de foto y fecha legible del nombre del archivo.
// "2026-02-04-lanzamiento-rediseno.jpeg" → ("Lanzamiento rediseno", "4 de febrero de 2026").
func captionFromFilename(name string) (caption, date string) {
	base := strings.TrimSuffix(name, filepath.Ext(name))
	if m := datePrefix.FindStringSubmatch(base); m != nil {
		y, _ := strconv.Atoi(m[1])
		mo, _ := strconv.Atoi(m[2])
		d, _ := strconv.Atoi(m[3])
		if mo >= 1 && mo <= 12 {
			date = fmt.Sprintf("%d de %s de %d", d, spanishMonths[mo], y)
		}
		base = base[len(m[0]):]
	}
	words := strings.FieldsFunc(base, func(r rune) bool { return r == '-' || r == '_' })
	caption = strings.Join(words, " ")
	if caption != "" {
		r := []rune(caption)
		caption = strings.ToUpper(string(r[0])) + string(r[1:])
	}
	return caption, date
}

// fitWidth calcula dimensiones manteniendo proporción con ancho máximo maxW.
func fitWidth(w, h, maxW int) (int, int) {
	if w <= maxW {
		return w, h
	}
	nh := int(float64(h)*float64(maxW)/float64(w) + 0.5)
	return maxW, nh
}

// resizeTo escala img al tamaño dado con interpolación CatmullRom (alta calidad).
func resizeTo(src image.Image, w, h int) image.Image {
	dst := image.NewRGBA(image.Rect(0, 0, w, h))
	draw.CatmullRom.Scale(dst, dst.Bounds(), src, src.Bounds(), draw.Over, nil)
	return dst
}

// loadCaptions lee fotos/captions.yaml (opcional): un mapa de nombre de archivo
// (sin extensión) a pie de foto con tildes y puntuación correctas.
func loadCaptions(fotosDir string) (map[string]string, error) {
	data, err := os.ReadFile(filepath.Join(fotosDir, "captions.yaml"))
	if os.IsNotExist(err) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	captions := map[string]string{}
	if err := yaml.Unmarshal(data, &captions); err != nil {
		return nil, fmt.Errorf("captions.yaml: %w", err)
	}
	return captions, nil
}

// ProcessPhotos optimiza cada foto de fotosDir hacia outDir/img y devuelve la
// lista para la galería, ordenada por fecha descendente y luego por nombre.
func ProcessPhotos(fotosDir, outDir string) ([]Photo, error) {
	entries, err := os.ReadDir(fotosDir)
	if os.IsNotExist(err) {
		return nil, nil // sin carpeta de fotos: galería vacía
	}
	if err != nil {
		return nil, err
	}
	captions, err := loadCaptions(fotosDir)
	if err != nil {
		return nil, err
	}
	imgOut := filepath.Join(outDir, photoOutSubdir)
	if err := os.MkdirAll(imgOut, 0o755); err != nil {
		return nil, err
	}

	var photos []Photo
	for _, e := range entries {
		if e.IsDir() || !photoExts[strings.ToLower(filepath.Ext(e.Name()))] {
			continue
		}
		p, err := processOne(filepath.Join(fotosDir, e.Name()), imgOut)
		if err != nil {
			return nil, fmt.Errorf("foto %s: %w", e.Name(), err)
		}
		if c, ok := captions[p.Name]; ok {
			p.Caption = c
		}
		photos = append(photos, p)
	}
	sort.Slice(photos, func(i, j int) bool {
		if photos[i].SortKey != photos[j].SortKey {
			return photos[i].SortKey > photos[j].SortKey // fecha más reciente primero
		}
		return photos[i].Name < photos[j].Name
	})
	return photos, nil
}

func processOne(srcPath, imgOut string) (Photo, error) {
	f, err := os.Open(srcPath)
	if err != nil {
		return Photo{}, err
	}
	defer f.Close()
	img, _, err := image.Decode(f)
	if err != nil {
		return Photo{}, fmt.Errorf("decodificando: %w", err)
	}

	name := filepath.Base(srcPath)
	base := strings.TrimSuffix(name, filepath.Ext(name))
	caption, date := captionFromFilename(name)

	b := img.Bounds()
	w, h := fitWidth(b.Dx(), b.Dy(), photoMaxWidth)
	tw, th := fitWidth(b.Dx(), b.Dy(), thumbMaxWidth)

	full := resizeTo(img, w, h)
	thumb := resizeTo(img, tw, th)

	fullName := base + ".jpg"
	thumbName := base + "-thumb.jpg"
	if err := writeJPEG(filepath.Join(imgOut, fullName), full); err != nil {
		return Photo{}, err
	}
	if err := writeJPEG(filepath.Join(imgOut, thumbName), thumb); err != nil {
		return Photo{}, err
	}

	sortKey := ""
	if m := datePrefix.FindStringSubmatch(base); m != nil {
		sortKey = m[1] + m[2] + m[3]
	}
	return Photo{
		Name:    base,
		Src:     "/" + photoOutSubdir + "/" + fullName,
		Thumb:   "/" + photoOutSubdir + "/" + thumbName,
		Caption: caption,
		Date:    date,
		Width:   w,
		Height:  h,
		ThumbW:  tw,
		ThumbH:  th,
		SortKey: sortKey,
	}, nil
}

func writeJPEG(path string, img image.Image) error {
	out, err := os.Create(path)
	if err != nil {
		return err
	}
	defer out.Close()
	return jpeg.Encode(out, img, &jpeg.Options{Quality: photoQuality})
}
